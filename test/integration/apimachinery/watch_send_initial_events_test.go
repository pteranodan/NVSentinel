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
	"k8s.io/utils/ptr"
)

func TestWatchSendInitialEvents(t *testing.T) {
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
			})
			defer teardown()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			gpuName := fmt.Sprintf("%s-initial", tc.name)
			gpu := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: gpuName},
				Spec:       devicev1alpha1.GPUSpec{UUID: "initial-uuid"},
			}
			created, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create pre-existing GPU: %v", err)
			}
			defer clientset.DeviceV1alpha1().GPUs().Delete(context.Background(), created.Name, metav1.DeleteOptions{})

			opts := metav1.ListOptions{
				SendInitialEvents:    ptr.To(true),
				AllowWatchBookmarks:  true,
				ResourceVersion:      "0",
				ResourceVersionMatch: metav1.ResourceVersionMatchNotOlderThan,
			}
			watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, opts)
			if err != nil {
				t.Fatalf("failed to start watch: %v", err)
			}
			defer watcher.Stop()

			var foundInitialGPU bool
			var foundSyncBookmark bool

			for {
				select {
				case event, ok := <-watcher.ResultChan():
					if !ok {
						t.Fatal("watch channel closed prematurely")
					}

					switch event.Type {
					case watch.Added:
						obj := event.Object.(*devicev1alpha1.GPU)
						if obj.Name == gpuName {
							foundInitialGPU = true
						}

					case watch.Bookmark:
						obj := event.Object.(*devicev1alpha1.GPU)

						if val, ok := obj.Annotations["k8s.io/initial-events-end"]; ok && val == "true" {
							foundSyncBookmark = true
						}

					case watch.Error:
						if tc.storageType == "memory" {
							framework.SkipWithWarning(t, fmt.Sprintf("Received watch error: %v", event.Object))
						}
						t.Fatalf("Received watch error: %v", event.Object)
					}

					if foundInitialGPU && foundSyncBookmark {
						return
					}

				case <-ctx.Done():
					if !foundInitialGPU {
						t.Error("failed to observe initial event")
					}
					if !foundSyncBookmark {
						t.Error("failed to observe bookmark event with 'k8s.io/initial-events-end' annotation")
					}
					t.Fatal("timed out waiting to observe initial event and bookmark event with 'k8s.io/initial-events-end' annotation")
				}
			}
		})
	}
}
