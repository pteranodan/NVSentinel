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
	"strconv"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/test/integration/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestWatchOrdering(t *testing.T) {
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
			clientset, teardown := framework.SetupServer(t, tc.storageType)
			defer teardown()

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
			if err != nil {
				if tc.storageType == "memory" {
					framework.SkipWithWarning(t, fmt.Sprintf("failed to list GPUs: %v", err))
				}
				t.Fatalf("failed to list GPUs: %v", err)
			}
			startRVStr := list.ResourceVersion
			startRV, _ := strconv.ParseUint(startRVStr, 10, 64)

			watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, metav1.ListOptions{
				ResourceVersion: startRVStr,
			})
			if err != nil {
				t.Fatalf("failed to start watch: %v", err)
			}
			defer watcher.Stop()

			names := []string{"gpu-seq-1", "gpu-seq-2", "gpu-seq-3", "gpu-seq-4"}
			for _, name := range names {
				uniqueName := fmt.Sprintf("%s-%s", tc.name, name)
				g := &devicev1alpha1.GPU{
					ObjectMeta: metav1.ObjectMeta{Name: uniqueName},
					Spec:       devicev1alpha1.GPUSpec{UUID: "seq-uuid"},
				}
				if _, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, g, metav1.CreateOptions{}); err != nil {
					t.Fatalf("failed to create %s: %v", uniqueName, err)
				}
			}

			lastRV := startRV
			for i, rawName := range names {
				expectedName := fmt.Sprintf("%s-%s", tc.name, rawName)
				select {
				case event, ok := <-watcher.ResultChan():
					if !ok {
						t.Fatal("watch channel closed prematurely")
					}

					if event.Type == watch.Error {
						t.Fatalf("received error event: %v", event.Object)
					}

					obj := event.Object.(*devicev1alpha1.GPU)
					currentRV, _ := strconv.ParseUint(obj.ResourceVersion, 10, 64)

					if obj.Name != expectedName {
						t.Errorf("expected %s as event %d, got %s", expectedName, i, obj.Name)
					}

					if currentRV <= lastRV {
						t.Fatalf("expected RV (%d) > previous (%d) for %s", currentRV, lastRV, obj.Name)
					}
					lastRV = currentRV

				case <-time.After(10 * time.Second):
					t.Fatalf("timed out waiting for event %d: %s", i, expectedName)
				}
			}
		})
	}
}
