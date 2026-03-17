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

func TestWatchBookmarks(t *testing.T) {
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

			gpu1Name := fmt.Sprintf("%s-bookmark-1", tc.name)
			gpu1 := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: gpu1Name},
				Spec:       devicev1alpha1.GPUSpec{UUID: "uuid-1"},
			}

			created1, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu1, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create first GPU: %v", err)
			}
			defer clientset.DeviceV1alpha1().GPUs().Delete(context.Background(), created1.Name, metav1.DeleteOptions{})

			opts := metav1.ListOptions{
				ResourceVersion:     created1.ResourceVersion,
				AllowWatchBookmarks: true,
			}
			watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, opts)
			if err != nil {
				t.Fatalf("failed to start watch: %v", err)
			}
			defer watcher.Stop()

			gpu2Name := fmt.Sprintf("%s-bookmark-2", tc.name)
			gpu2 := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: gpu2Name},
				Spec:       devicev1alpha1.GPUSpec{UUID: "uuid-2"},
			}

			created2, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu2, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create second GPU: %v", err)
			}
			defer clientset.DeviceV1alpha1().GPUs().Delete(context.Background(), created2.Name, metav1.DeleteOptions{})

			for {
				select {
				case event, ok := <-watcher.ResultChan():
					if !ok {
						if tc.storageType == "memory" {
							framework.SkipWithWarning(t, "watch channel closed prematurely")
						}
						t.Fatal("watch channel closed prematurely")
					}

					if event.Type != watch.Bookmark {
						continue
					}

					obj, ok := event.Object.(*devicev1alpha1.GPU)
					if !ok {
						t.Fatalf("Unexpected bookmark event object: got %T, want *devicev1alpha1.GPU", event.Object)
					}

					expectedKind := "GPU"
					expectedAPIVersion := "device.nvidia.com/v1alpha1"
					if obj.Kind != expectedKind || obj.APIVersion != expectedAPIVersion {
						t.Errorf("Unexpected bookmark event object Type: got %s/%s, want %s/%s",
							expectedAPIVersion, expectedKind, obj.APIVersion, obj.Kind)
					}

					if obj.ResourceVersion == "" {
						t.Errorf("Missing bookmark event object RV: got %v, want 'non-empty'", obj.ResourceVersion)
					}

					m := obj.ObjectMeta
					if m.Name != "" || m.GenerateName != "" || m.Namespace != "" ||
						m.UID != "" || m.Generation > 0 || !m.CreationTimestamp.IsZero() ||
						len(m.Labels) > 0 || len(m.Annotations) > 0 ||
						len(m.OwnerReferences) > 0 || len(m.Finalizers) > 0 ||
						len(m.ManagedFields) > 0 {
						t.Errorf("Unexpected bookmark event object metadata; should only include ResourceVersion. Got: %+v", m)
					}

					if obj.Spec.UUID != "" ||
						len(obj.Status.Conditions) > 0 ||
						obj.Status.RecommendedAction != "" {
						t.Errorf("Unexpected bookmark event object Spec/Status data; should be omitted. Got: %#v", obj)
					}

					return

				case <-ctx.Done():
					t.Fatal("timed out waiting to observe bookmark event")
				}
			}
		})
	}
}
