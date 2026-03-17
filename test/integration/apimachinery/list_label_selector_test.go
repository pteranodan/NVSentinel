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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/test/integration/framework"
)

func TestListLabelSelector(t *testing.T) {
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

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			gpu1 := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{
					Name:   fmt.Sprintf("%s-list-label-selector-1", tc.name),
					Labels: map[string]string{"label-1": "true"},
				},
				Spec: devicev1alpha1.GPUSpec{UUID: "list-label-selector-1"},
			}
			_, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu1, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("unexpected error creating %s: %v", gpu1.Name, err)
			}

			gpu2 := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{
					Name:   fmt.Sprintf("%s-list-label-selector-2", tc.name),
					Labels: map[string]string{"label-2": "true"},
				},
				Spec: devicev1alpha1.GPUSpec{UUID: "list-label-selector-2"},
			}
			if _, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu2, metav1.CreateOptions{}); err != nil {
				t.Fatalf("unexpected error creating %s: %v", gpu2.Name, err)
			}

			t.Run("EqualityMatch", func(t *testing.T) {
				opts := metav1.ListOptions{LabelSelector: "label-1=true"}
				list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
				if err != nil {
					t.Fatalf("failed to list: %v", err)
				}
				if len(list.Items) != 1 || list.Items[0].Name != gpu1.Name {
					t.Errorf("expected only gpu1, got %d items", len(list.Items))
				}
			})

			t.Run("InequalityMatch", func(t *testing.T) {
				opts := metav1.ListOptions{LabelSelector: "label-1!=true"}
				list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
				if err != nil {
					t.Fatalf("failed to list: %v", err)
				}

				for _, item := range list.Items {
					if item.Name == gpu1.Name {
						t.Errorf("gpu1 should have been filtered out by label-1!=true")
					}
				}
			})

			t.Run("SetInMatch", func(t *testing.T) {
				opts := metav1.ListOptions{LabelSelector: "label-1 in (true, yes)"}
				list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
				if err != nil {
					t.Fatalf("failed to list: %v", err)
				}
				if len(list.Items) != 1 || list.Items[0].Name != gpu1.Name {
					t.Errorf("set-based match failed, got %d items", len(list.Items))
				}
			})

			t.Run("IntersectionMatch", func(t *testing.T) {
				opts := metav1.ListOptions{LabelSelector: "label-1=true,label-2=true"}
				list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
				if err != nil {
					t.Fatalf("failed to list: %v", err)
				}
				if len(list.Items) != 0 {
					t.Errorf("expected 0 items for impossible intersection, got %d", len(list.Items))
				}
			})

			t.Run("InvalidSelector", func(t *testing.T) {
				opts := metav1.ListOptions{LabelSelector: "label-1==="}
				_, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
				if err == nil {
					t.Fatal("expected error for invalid label selector, got nil")
				}
				if !errors.IsBadRequest(err) {
					t.Errorf("expected BadRequest error, got: %v", err)
				}
			})
		})
	}
}
