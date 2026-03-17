// Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build integration

package apimachinery_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/test/integration/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestWatchResourceGone(t *testing.T) {
	testCases := []struct {
		name        string
		storageType string
	}{
		{name: "OnDisk", storageType: apistorage.StorageTypeETCD3},
		{name: "InMemory", storageType: "memory"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			clientset, teardown := framework.SetupServerWithOptions(t, framework.TestServerOptions{
				StorageType:                 tc.storageType,
				WatchProgressNotifyInterval: 500 * time.Millisecond,
				CompactionInterval:          50 * time.Millisecond,
				CompactionMinRetain:         1,
			})
			defer teardown()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			gpuName := fmt.Sprintf("%s-gone-gpu", tc.name)
			gpu := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: gpuName},
				Spec:       devicev1alpha1.GPUSpec{UUID: "gone-uuid"},
			}

			created, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create GPU: %v", err)
			}
			oldRV := created.ResourceVersion

			for i := 0; i < 5; i++ {
				created.Spec.UUID = fmt.Sprintf("gpu-%d", i)
				created, _ = clientset.DeviceV1alpha1().GPUs().Update(ctx, created, metav1.UpdateOptions{})
			}

			watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, metav1.ListOptions{
				ResourceVersion: oldRV,
			})
			if err != nil {
				t.Fatalf("failed to start watch: %v", err)
			}
			defer watcher.Stop()

			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					t.Fatal("watch channel closed prematurely")
				}

				if event.Type == watch.Error {
					status := event.Object.(*metav1.Status)
					if status.Code != 410 {
						t.Fatalf("expected 410 (Gone) error, got event type %v: %+v", event.Type, event.Object)
					}
				}

			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for 410 (Gone) event")
			}
		})
	}
}
