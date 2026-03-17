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
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestGetResourceVersion(t *testing.T) {
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

			gpuName := fmt.Sprintf("%s-get-rv", tc.name)
			gpu := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: gpuName},
				Spec:       devicev1alpha1.GPUSpec{UUID: "get-rv-uuid"},
			}

			created, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
			if err != nil {

				t.Fatalf("failed to create: %v", err)
			}
			createdRV := created.ResourceVersion

			created.Spec.UUID = "get-rv-uuid-updated"
			updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, created, metav1.UpdateOptions{})
			if err != nil {
				t.Fatalf("failed to update: %v", err)
			}
			updatedRV := updated.ResourceVersion

			// Requesting an old RV must return the latest state.
			fetched, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{
				ResourceVersion: createdRV,
			})
			if err != nil {
				t.Errorf("failed to get with old RV %s: %v", createdRV, err)
			}
			if fetched.ResourceVersion != updatedRV {
				t.Errorf("expected RV %s (v2), got %s", updatedRV, fetched.ResourceVersion)
			}
			if fetched.Spec.UUID != "get-rv-uuid-updated" {
				t.Errorf("expected Spec %s (v2), got %s", updated.Spec, fetched.Spec)
			}

			// Requesting future RV should fail
			futureRV := "999999"
			sctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()

			_, err = clientset.DeviceV1alpha1().GPUs().Get(sctx, gpuName, metav1.GetOptions{
				ResourceVersion: futureRV,
			})
			if err == nil {
				if tc.storageType == "memory" {
					framework.SkipWithWarning(t, "expected error when requesting a future RV, got nil")
				}
				t.Errorf("expected error when requesting a future RV, got nil")
			}
		})
	}
}
