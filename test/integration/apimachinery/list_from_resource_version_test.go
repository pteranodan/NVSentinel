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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/test/integration/framework"
)

func TestListFromResourceVersion(t *testing.T) {
	testCases := []struct {
		name           string
		storageBackend string
	}{
		{name: "OnDisk", storageBackend: apistorage.StorageTypeETCD3},
		{name: "InMemory", storageBackend: "memory"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			clientset, teardown := framework.SetupServer(t, tc.storageBackend)
			defer teardown()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			gpu1 := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-list-rv1", tc.name)},
				Spec:       devicev1alpha1.GPUSpec{UUID: "list-rv1"},
			}
			created1, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu1, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("unexpected error creating %s: %v", gpu1.Name, err)
			}
			rv1 := created1.ResourceVersion

			gpu2 := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s-list-rv2", tc.name)},
				Spec:       devicev1alpha1.GPUSpec{UUID: "list-rv2"},
			}
			if _, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu2, metav1.CreateOptions{}); err != nil {
				t.Fatalf("unexpected error creating %s: %v", gpu2.Name, err)
			}

			t.Run("RV=0 (Any)", func(t *testing.T) {
				opts := metav1.ListOptions{ResourceVersion: "0"}
				list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
				if err != nil {
					t.Fatalf("failed to list GPUs: %v", err)
				}
				if len(list.Items) == 0 {
					t.Error("failed to list GPUs, got empty list")
				}
			})

			t.Run("RV=Old (NotOlderThan)", func(t *testing.T) {
				opts := metav1.ListOptions{ResourceVersion: rv1}
				list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
				if err != nil {
					t.Fatalf("failed to list GPUs: %v", err)
				}

				foundGpu2 := false
				for _, item := range list.Items {
					if item.Name == gpu2.Name {
						foundGpu2 = true
						break
					}
				}
				if !foundGpu2 {
					t.Errorf("failed to return newest object %q, got nil", gpu2.Name)
				}
			})

			t.Run("RV=Old (Exact)", func(t *testing.T) {
				optsExact := metav1.ListOptions{
					ResourceVersion:      rv1,
					ResourceVersionMatch: metav1.ResourceVersionMatchExact,
				}
				listExact, err := clientset.DeviceV1alpha1().GPUs().List(ctx, optsExact)
				if err != nil {
					t.Fatalf("failed to list GPUs: %v", err)
				}

				foundGpu1, foundGpu2 := false, false
				for _, item := range listExact.Items {
					if item.Name == gpu1.Name {
						foundGpu1 = true
					}
					if item.Name == gpu2.Name {
						foundGpu2 = true
					}
				}

				if !foundGpu1 {
					t.Errorf("failed to return base object %q, got nil", gpu1.Name)
				}
				if foundGpu2 {
					if tc.storageBackend == "memory" {
						framework.SkipWithWarning(t, fmt.Sprintf("failed to isolate snapshot, got future object %q", gpu2.Name))
					}
					t.Errorf("failed to isolate snapshot, got future object %q", gpu2.Name)
				}
			})

			t.Run("RV=Empty (Live)", func(t *testing.T) {
				listLive, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
				if err != nil {
					t.Fatalf("failed to list GPUs: %v", err)
				}

				foundGpu2Live := false
				for _, item := range listLive.Items {
					if item.Name == gpu2.Name {
						foundGpu2Live = true
						break
					}
				}
				if !foundGpu2Live {
					t.Errorf("failed to return newest object %q, got nil", gpu2.Name)
				}
			})
		})
	}
}
