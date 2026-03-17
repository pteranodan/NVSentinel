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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestDeletePreconditions(t *testing.T) {
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

			gpuName := fmt.Sprintf("%s-delete-precond", tc.name)
			gpu := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: gpuName},
				Spec:       devicev1alpha1.GPUSpec{UUID: "delete-precond-uuid"},
			}

			current, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create GPU: %v", err)
			}

			wrongUID := types.UID("00000000-0000-0000-0000-000000000000")
			err = clientset.DeviceV1alpha1().GPUs().Delete(ctx, gpuName, metav1.DeleteOptions{
				Preconditions: &metav1.Preconditions{UID: &wrongUID},
			})
			if err == nil {
				t.Error("expected error when deleting with wrong UID, got nil")
			} else if !errors.IsConflict(err) {
				t.Errorf("expected Conflict (409) error for UID mismatch, got %v", err)
			}

			wrongRV := "999999"
			err = clientset.DeviceV1alpha1().GPUs().Delete(ctx, gpuName, metav1.DeleteOptions{
				Preconditions: &metav1.Preconditions{ResourceVersion: &wrongRV},
			})
			if err == nil {
				t.Error("expected error when deleting with wrong ResourceVersion, got nil")
			} else if !errors.IsConflict(err) {
				t.Errorf("expected Conflict (409) error for RV mismatch, got: %v", err)
			}

			err = clientset.DeviceV1alpha1().GPUs().Delete(ctx, gpuName, metav1.DeleteOptions{
				Preconditions: &metav1.Preconditions{
					UID:             &current.UID,
					ResourceVersion: &current.ResourceVersion,
				},
			})
			if err != nil {
				t.Fatalf("failed to delete GPU with correct preconditions: %v", err)
			}

			_, err = clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{})
			if !errors.IsNotFound(err) {
				t.Errorf("expected NotFound after deletion, got: %v", err)
			}
		})
	}
}
