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

package gpu_test

import (
	"context"
	"testing"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/test/integration/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestGPU(t *testing.T) {
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

			ctx := context.Background()
			gpu := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: "gpu-test"},
				Spec:       devicev1alpha1.GPUSpec{UUID: "GPU-12345"},
			}
			created, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("Failed to register GPU: %v", err)
			}

			if created.Kind != "GPU" {
				t.Errorf("expected Kind %q, got %q", "GPU", created.Kind)
			}
			if created.APIVersion != "device.nvidia.com/v1alpha1" {
				t.Errorf("expected APIVersion %q, got %q", "device.nvidia.com/v1alpha1", created.APIVersion)
			}

			created.Spec.UUID = "GPU-54321"
			updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, created, metav1.UpdateOptions{})
			if err != nil {
				t.Fatalf("Failed to update GPU spec: %v", err)
			}
			if updated.Spec.UUID != "GPU-54321" {
				t.Errorf("expected patchedSpec UUID %q, got %q", "GPU-54321", updated.Spec.UUID)
			}

			updated.Status.Conditions = []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					Reason:             "DriverReady",
					Message:            "Driver is posting ready status",
					LastTransitionTime: metav1.Now(),
				},
			}

			updatedStatus, err := clientset.DeviceV1alpha1().GPUs().UpdateStatus(ctx, updated, metav1.UpdateOptions{})
			if err != nil {
				t.Fatalf("Failed to update GPU status: %v", err)
			}
			updatedConds := updatedStatus.Status.Conditions
			if len(updatedConds) != 1 && updatedConds[0].Type != "Ready" && updatedConds[0].Status != metav1.ConditionTrue {
				t.Errorf("expected status %+v\n, got %+v\n", updated.Status, updatedStatus.Status)
			}

			patchSpecData := []byte(`{"spec":{"uuid":"GPU-PATCHED"}}`)
			patchedSpec, err := clientset.DeviceV1alpha1().GPUs().Patch(ctx, gpu.Name, types.MergePatchType, patchSpecData, metav1.PatchOptions{})
			if err != nil {
				t.Fatalf("Failed to patch GPU spec: %v", err)
			}
			if patchedSpec.Spec.UUID != "GPU-PATCHED" {
				t.Errorf("expected UUID %q, got %q", "GPU-PATCHED", patchedSpec.Spec.UUID)
			}

			patchStatusData := []byte(`{"status":{"conditions":[{"type":"Ready","status":"False","reason":"Patched","message":"Status was patched"}]}}`)
			patchedStatus, err := clientset.DeviceV1alpha1().GPUs().Patch(ctx, gpu.Name, types.MergePatchType, patchStatusData, metav1.PatchOptions{}, "status")
			if err != nil {
				t.Fatalf("Failed to patch GPU status: %v", err)
			}
			patchedConds := patchedStatus.Status.Conditions
			if len(patchedConds) != 1 && patchedConds[0].Status != metav1.ConditionTrue {
				t.Errorf("expected status %+v\n, got %+v\n", patchStatusData, patchedStatus.Status)
			}

			_, err = clientset.DeviceV1alpha1().GPUs().Get(ctx, gpu.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("Failed to get GPU: %v", err)
			}

			if err := clientset.DeviceV1alpha1().GPUs().Delete(ctx, gpu.Name, metav1.DeleteOptions{}); err != nil {
				t.Errorf("Failed to delete GPU: %v", err)
			}
		})
	}
}
