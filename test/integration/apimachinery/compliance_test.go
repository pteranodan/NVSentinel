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

package apimachinery_test

import (
	"context"
	"strconv"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestCompliance(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gpu := &devicev1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-comp"},
		Spec:       devicev1alpha1.GPUSpec{UUID: "comp-uuid"},
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

	t.Run("Immutability", func(t *testing.T) {
		original, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpu.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get GPU %s: %v", gpu.Name, err)
		}

		toUpdate := original.DeepCopy()

		toUpdate.UID = types.UID("immut-uuid")
		toUpdate.CreationTimestamp = metav1.Time{Time: time.Now()}

		updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, toUpdate, metav1.UpdateOptions{})
		if err != nil {
			if errors.IsInvalid(err) || errors.IsBadRequest(err) {
				return
			}
			t.Fatalf("failed to update GPU %s: %v", toUpdate.Name, err)
		}

		if updated.UID != original.UID {
			t.Errorf("unexpected UID %q, want %q", updated.UID, original.UID)
		}
		if !updated.CreationTimestamp.Equal(&original.CreationTimestamp) {
			t.Errorf("unexpected CreationTimestamp %s, want %s", updated.CreationTimestamp, original.CreationTimestamp)
		}
	})

	t.Run("Optimistic Concurrency", func(t *testing.T) {
		obj1, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpu.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get GPU %s: %v", gpu.Name, err)
		}

		obj1.Spec.UUID = "opt-con-uuid"
		obj2, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, obj1, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("failed to update GPU %s: %v", obj1.Name, err)
		}

		updatedUUID := "opt-con-uuid-new"
		obj1.Spec.UUID = updatedUUID
		_, err = clientset.DeviceV1alpha1().GPUs().Update(ctx, obj1, metav1.UpdateOptions{})
		if !errors.IsConflict(err) {
			t.Errorf("expected Conflict error (409), got: %v", err)
		}

		obj2.Spec.UUID = updatedUUID
		_, err = clientset.DeviceV1alpha1().GPUs().Update(ctx, obj2, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("failed to update GPU %s: %v", obj2.Name, err)
		}
	})

	t.Run("ResourceVersion Semantics", func(t *testing.T) {
		original, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpu.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get GPU %s: %v", gpu.Name, err)
		}

		oldRV, err := strconv.ParseUint(original.ResourceVersion, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse ResourceVersion: %v", err)
		}

		original.Spec.UUID = "rv-semantics-uuid"
		updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, original, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("failed to update GPU %s: %v", original.Name, err)
		}

		newRV, err := strconv.ParseUint(updated.ResourceVersion, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse ResourceVersion: %v", err)
		}

		if newRV <= oldRV {
			t.Errorf("expected ResourceVersion (%d) to increase, got %d)", oldRV, newRV)
		}
	})

	t.Run("List ResourceVersion Semantics", func(t *testing.T) {
		gpu := &devicev1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "list-rv-semantics"}}
		created, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("failed to create GPU %s: %v", gpu.Name, err)
		}

		itemRV, err := strconv.ParseUint(created.ResourceVersion, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse ResourceVersion: %v", err)
		}

		list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("failed to list GPUs: %v", err)
		}

		listRV, err := strconv.ParseUint(list.ResourceVersion, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse ResourceVersion: %v", err)
		}

		if listRV < itemRV {
			t.Errorf("expected List RV (%d) to be greater than last created item RV (%d).", listRV, itemRV)
		}
	})

	t.Run("Generation Semantics", func(t *testing.T) {
		obj, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpu.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get GPU %s: %v", gpu.Name, err)
		}

		initialGen := obj.Generation
		initialRV, err := strconv.ParseUint(obj.ResourceVersion, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse ResourceVersion: %v", err)
		}

		obj.Spec.UUID = "gen-semantics-uuid"
		updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("failed to update GPU %s: %v", obj.Name, err)
		}

		updatedRV, err := strconv.ParseUint(updated.ResourceVersion, 10, 64)
		if err != nil {
			t.Fatalf("failed to parse ResourceVersion: %v", err)
		}

		if updated.Generation != initialGen+1 {
			t.Errorf("expected Generation to increment on Spec change: got %d, want %d", updated.Generation, initialGen+1)
		}
		if updatedRV <= initialRV {
			t.Errorf("expected ResourceVersion to increment on Spec change: got %d, want > %d", updatedRV, initialRV)
		}
	})

	t.Run("Spec Separation", func(t *testing.T) {
		obj, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpu.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get GPU %s: %v", gpu.Name, err)
		}

		updatedUUID := "spec-separation-uuid"
		obj.Spec.UUID = updatedUUID
		obj.Status.Conditions = []metav1.Condition{{
			Type: "Ready", Status: "True", Reason: "Test", Message: "Compliance",
			LastTransitionTime: metav1.Now(),
		}}

		updated, err := clientset.DeviceV1alpha1().GPUs().Update(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("failed to update GPU %s: %v", obj.Name, err)
		}

		if len(updated.Status.Conditions) != 0 {
			t.Errorf("expected Update to ignore Status change: got %d conditions, want %d", len(updated.Status.Conditions), 0)
		}

		if updated.Spec.UUID != updatedUUID {
			t.Errorf("expected Update to accept Spec change: got %s, want %s", updated.Spec.UUID, updatedUUID)
		}
	})

	t.Run("Status Separation", func(t *testing.T) {
		obj, err := clientset.DeviceV1alpha1().GPUs().Get(ctx, gpu.Name, metav1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get GPU %s: %v", gpu.Name, err)
		}

		updatedUUID := "status-separation-uuid"
		obj.Spec.UUID = updatedUUID
		obj.Status.Conditions = []metav1.Condition{{
			Type: "Ready", Status: "True", Reason: "Test", Message: "Compliance",
			LastTransitionTime: metav1.Now(),
		}}

		updated, err := clientset.DeviceV1alpha1().GPUs().UpdateStatus(ctx, obj, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("failed to update GPU %s: %v", obj.Name, err)
		}

		if len(updated.Status.Conditions) != 1 {
			t.Errorf("expected UpdateStatus to accept Status change: got %d conditions, want %d", len(updated.Status.Conditions), 1)
		}

		if updated.Spec.UUID == updatedUUID {
			t.Errorf("expected Update to ignore Spec change: got %s, want %s", updated.Spec.UUID, updatedUUID)
		}
	})
}
