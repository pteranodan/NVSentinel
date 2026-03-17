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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestAPISemantics(t *testing.T) {
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

			gpuName := fmt.Sprintf("%s-gpu-comf", tc.name)
			gpu := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: gpuName},
				Spec:       devicev1alpha1.GPUSpec{UUID: "comf-uuid"},
			}

			t.Run("Metadata Generation", func(t *testing.T) {
				created, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
				if err != nil {
					t.Fatalf("failed to create GPU %s: %v", gpu.Name, err)
				}

				if created.UID == "" {
					t.Error("server failed to generate UID")
				}
				if created.CreationTimestamp.IsZero() {
					t.Error("server failed to set CreationTimestamp")
				}
				if created.Generation != 1 {
					t.Errorf("expected Generation=1, got %d", created.Generation)
				}
				if created.ResourceVersion == "" {
					t.Error("server failed to set ResourceVersion")
				}
			})

			t.Run("Metadata Only Update", func(t *testing.T) {
				original, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}

				toUpdate := original.DeepCopy()
				toUpdate.Annotations["updated"] = "true"

				updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, toUpdate, metav1.UpdateOptions{})
				if err != nil {
					t.Fatalf("failed to update metadata: %v", err)
				}

				if updated.Annotations["updated"] != "true" {
					t.Error("metadata (annotation) not updated")
				}
				if updated.Generation != original.Generation {
					t.Errorf("unexpected generation increase on metadata-only (annotation) update (got %d, want %d)", updated.Generation, original.Generation)
				}
			})

			t.Run("Immutability", func(t *testing.T) {
				original, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("failed to get GPU %s: %v", gpuName, err)
				}

				toUpdate := original.DeepCopy()
				toUpdate.UID = types.UID("immut-uuid")
				toUpdate.CreationTimestamp = metav1.Time{Time: time.Now()}

				_, err = clientset.DeviceV1alpha1().GPUs().Update(ctx, toUpdate, metav1.UpdateOptions{})
				if err == nil {
					t.Error("expected error when updating immutable field via Update, got nil")
				} else if !errors.IsInvalid(err) && !errors.IsBadRequest(err) {
					t.Errorf("expected Invalid or BadRequest error, got: %v", err)
				}

				toUpdateStatus := original.DeepCopy()
				toUpdateStatus.UID = types.UID("immut-status-uuid")
				toUpdateStatus.CreationTimestamp = metav1.Time{Time: time.Now()}

				_, err = clientset.DeviceV1alpha1().GPUs().UpdateStatus(ctx, toUpdateStatus, metav1.UpdateOptions{})
				if err == nil {
					t.Error("expected error when updating immutable field via UpdateStatus, got nil")
				} else if !errors.IsInvalid(err) && !errors.IsBadRequest(err) {
					t.Errorf("expected Invalid or BadRequest error, got: %v", err)
				}

				patchData := []byte(`{"metadata":{"uid":"immut-patch-uuid"}}`)

				_, err = clientset.DeviceV1alpha1().GPUs().Patch(ctx, gpuName, types.MergePatchType, patchData, metav1.PatchOptions{})
				if err == nil {
					t.Error("expected error when patching immutable field, got nil")
				} else if !errors.IsInvalid(err) && !errors.IsBadRequest(err) {
					t.Errorf("expected Invalid or BadRequest error, got: %v", err)
				}

				final, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{})
				if err != nil {
					t.Fatal(err)
				}
				if final.UID != original.UID {
					t.Errorf("unexpected update: UID was updated. Got %q, want %q", final.UID, original.UID)
				}
				if final.CreationTimestamp != original.CreationTimestamp {
					t.Errorf("unexpected update: CreationTimestamp was updated. Got %q, want %q", final.CreationTimestamp, original.CreationTimestamp)
				}
			})

			t.Run("Optimistic Concurrency", func(t *testing.T) {
				obj1, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("failed to get GPU %s: %v", gpuName, err)
				}

				obj1.Spec.UUID = "opt-con-uuid"
				obj2, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, obj1, metav1.UpdateOptions{})
				if err != nil {
					t.Fatalf("failed to update GPU %s: %v", obj1.Name, err)
				}

				// Try to update using the stale obj1 (old ResourceVersion)
				updatedUUID := "opt-con-uuid-new"
				obj1.Spec.UUID = updatedUUID
				_, err = clientset.DeviceV1alpha1().GPUs().Update(ctx, obj1, metav1.UpdateOptions{})
				if !errors.IsConflict(err) {
					t.Errorf("expected Conflict error (409) for stale Update, got: %v", err)
				}

				// Update using the fresh obj2 (new ResourceVersion)
				obj2.Spec.UUID = updatedUUID
				obj3, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, obj2, metav1.UpdateOptions{})
				if err != nil {
					t.Errorf("failed to update GPU %s: %v", obj2.Name, err)
				}

				obj2.Status.Conditions = []metav1.Condition{{
					Type: "Ready", Status: "True", Reason: "StaleStatusUpdate",
				}}
				_, err = clientset.DeviceV1alpha1().GPUs().UpdateStatus(ctx, obj2, metav1.UpdateOptions{})
				if !errors.IsConflict(err) {
					t.Errorf("expected Conflict error (409) for stale UpdateStatus, got: %v", err)
				}

				// UpdateStatus using the fresh obj3 (newest ResourceVersion)
				obj3.Status.Conditions = []metav1.Condition{{
					Type: "Ready", Status: "True", Reason: "FreshStatusUpdate",
				}}
				_, err = clientset.DeviceV1alpha1().GPUs().UpdateStatus(ctx, obj3, metav1.UpdateOptions{})
				if err != nil {
					t.Errorf("failed to update GPU status %s: %v", obj3.Name, err)
				}
			})

			t.Run("ResourceVersion Semantics", func(t *testing.T) {
				original, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("failed to get GPU %s: %v", gpuName, err)
				}

				oldRV, _ := strconv.ParseUint(original.ResourceVersion, 10, 64)
				original.Spec.UUID = "rv-semantics-uuid"
				updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, original, metav1.UpdateOptions{})
				if err != nil {
					t.Fatalf("failed to update GPU %s: %v", gpuName, err)
				}

				newRV, _ := strconv.ParseUint(updated.ResourceVersion, 10, 64)
				if newRV <= oldRV {
					t.Errorf("expected ResourceVersion (%d) to increase on Update, got %d", oldRV, newRV)
				}

				updated.Status.Conditions = []metav1.Condition{{
					Type: "Ready", Status: "True", Reason: "RVTest", Message: "Checking increment",
					LastTransitionTime: metav1.Now(),
				}}
				statusUpdated, err := clientset.DeviceV1alpha1().GPUs().UpdateStatus(ctx, updated, metav1.UpdateOptions{})
				if err != nil {
					t.Fatalf("failed to update GPU status: %v", err)
				}
				statusRV, _ := strconv.ParseUint(statusUpdated.ResourceVersion, 10, 64)
				if statusRV <= newRV {
					t.Errorf("expected ResourceVersion (%d) to increase after UpdateStatus, got %d", newRV, statusRV)
				}

				patchData := []byte(`{"status": {"conditions": [{"type": "Ready", "status": "False", "reason": "PatchRV"}]}}`)
				patchedStatus, err := clientset.DeviceV1alpha1().GPUs().Patch(ctx, gpuName, types.MergePatchType, patchData, metav1.PatchOptions{}, "status")
				if err != nil {
					t.Fatalf("failed to patch GPU status: %v", err)
				}
				patchRV, _ := strconv.ParseUint(patchedStatus.ResourceVersion, 10, 64)
				if patchRV <= statusRV {
					t.Errorf("expected ResourceVersion (%d) to increase after Patch of status, got %d", statusRV, patchRV)
				}
			})

			t.Run("Spec Separation", func(t *testing.T) {
				obj, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("failed to get GPU %s: %v", gpuName, err)
				}
				origCondLen := len(obj.Status.Conditions)

				updatedUUID := "spec-separation-uuid"
				obj.Spec.UUID = updatedUUID
				obj.Status.Conditions = append(obj.Status.Conditions, metav1.Condition{
					Type:               "SpecSeparation",
					Status:             "True",
					Reason:             "Test",
					Message:            "SpecSeparation",
					LastTransitionTime: metav1.Now(),
				})

				updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, obj, metav1.UpdateOptions{})
				if err != nil {
					t.Errorf("failed to update GPU %s: %v", obj.Name, err)
				}

				if len(updated.Status.Conditions) != origCondLen {
					t.Errorf("expected Update to ignore Status change: got %d conditions, want %d", len(updated.Status.Conditions), origCondLen)
				}
				if updated.Spec.UUID != updatedUUID {
					t.Errorf("expected Update to accept Spec change: got %s, want %s", updated.Spec.UUID, updatedUUID)
				}

				patchedUUID := "spec-separation-patch-uuid"
				patchData := []byte(`{"metadata":{"annotations":{"patch":"true"}},"spec":{"uuid":"spec-separation-patch-uuid"},"status":{"conditions":[{"type":"Ready","status":"False"},{"type":"Degraded","status":"False"}]}}`)

				patched, err := clientset.DeviceV1alpha1().GPUs().Patch(ctx, obj.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
				if err != nil {
					t.Errorf("failed to patch GPU %s: %v", obj.Name, err)
				}

				if len(patched.Status.Conditions) != origCondLen {
					t.Errorf("expected Patch to ignore Status change: got %d conditions, want %d", len(patched.Status.Conditions), origCondLen)
				}
				if patched.Spec.UUID != patchedUUID {
					t.Errorf("expected Patch to accept Spec change: got %s, want %s", patched.Spec.UUID, patchedUUID)
				}
				_, exists := patched.Annotations["patch"]
				if !exists {
					t.Errorf("expected Patch to accept Metadata change: got annotations %s, want 'patched:true'", patched.Annotations)
				}
			})

			t.Run("Status Separation", func(t *testing.T) {
				obj, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpuName, metav1.GetOptions{})
				if err != nil {
					t.Fatalf("failed to get GPU %s: %v", gpuName, err)
				}

				updatedUUID := "status-separation-uuid"
				obj.Spec.UUID = updatedUUID
				obj.Annotations = map[string]string{"update-status": "false"}
				obj.Status.Conditions = []metav1.Condition{{
					Type:               "Ready",
					Status:             "True",
					Reason:             "Test",
					Message:            "Compliance",
					LastTransitionTime: metav1.Now(),
				}}

				updated, err := clientset.DeviceV1alpha1().GPUs().UpdateStatus(ctx, obj, metav1.UpdateOptions{})
				if err != nil {
					t.Errorf("failed to update GPU status %s: %v", obj.Name, err)
				}

				if len(updated.Status.Conditions) != len(obj.Status.Conditions) {
					t.Errorf("expected UpdateStatus to accept Status change: got %d conditions, want %d", len(updated.Status.Conditions), len(obj.Status.Conditions))
				}
				_, exists := updated.Annotations["update-status"]
				if exists {
					t.Errorf("expected UpdateStatus to ignore Metadata changes: got annotations %s, want %s", updated.Annotations, obj.Annotations)
				}
				if updated.Spec.UUID == updatedUUID {
					t.Errorf("expected UpdateStatus to ignore Spec change: got %s, want original Spec", updated.Spec.UUID)
				}

				patchData := []byte(`{"metadata":{"annotations":{"status-patch":"false"}},"spec":{"uuid":"status-separation-patch-uuid"},"status":{"conditions":[{"type":"Ready","status":"False"},{"type":"Degraded","status":"False"},{"type":"FabricReady","status":"True"}]}}`)

				patched, err := clientset.DeviceV1alpha1().GPUs().Patch(ctx, obj.Name, types.MergePatchType, patchData, metav1.PatchOptions{}, "status")
				if err != nil {
					t.Errorf("failed to patch GPU status %s: %v", obj.Name, err)
				}
				if len(patched.Status.Conditions) != 3 {
					t.Errorf("expected Status Patch to accept Status change: got %d conditions, want 3", len(patched.Status.Conditions))
				}
				_, exists = patched.Annotations["status-patch"]
				if exists {
					t.Errorf("expected Status Patch to ignore Metadata changes: got annotations %s, want %s", patched.Annotations, obj.Annotations)
				}
				if patched.Spec.UUID != updated.Spec.UUID {
					t.Errorf("expected Status Patch to ignore Spec change: got %s, want %s", patched.Spec.UUID, updated.Spec.UUID)
				}
			})
		})
	}
}
