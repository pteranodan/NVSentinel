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

package gpu_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGPU(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := clientset

	gpu1 := &devicev1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-1"},
		Spec:       devicev1alpha1.GPUSpec{UUID: "GPU-1"},
		Status: devicev1alpha1.GPUStatus{
			Conditions: []metav1.Condition{
				{
					Type:    "Ready",
					Status:  metav1.ConditionFalse,
					Reason:  "DriverNotReaady",
					Message: "Driver is posting ready status",
				},
			},
		},
	}

	var created *devicev1alpha1.GPU
	var err error

	t.Run("Create", func(t *testing.T) {
		created, err = client.DeviceV1alpha1().GPUs().Create(ctx, gpu1, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("failed to create GPU %s: %v", gpu1.Name, err)
		}

		// Client-side defaults and generation
		if created.Kind != "GPU" {
			t.Errorf("expected Kind %q, got %q", "GPU", created.Kind)
		}
		if created.APIVersion != "device.nvidia.com/v1alpha1" {
			t.Errorf("expected APIVersion %q, got %q", "device.nvidia.com/v1alpha1", created.APIVersion)
		}

		// Server-side defaults and generation
		if created.Namespace != "default" {
			t.Errorf("expected Namespace %q, got %q", "default", created.Namespace)
		}
		if created.Generation != 1 {
			t.Errorf("expected Generation %d, got %d", 1, created.Generation)
		}
		if created.UID == "" {
			t.Errorf("expected UID to be set, got %s", created.UID)
		}
		if created.ResourceVersion == "" {
			t.Errorf("expected ResourceVersion to be set, got %s", created.ResourceVersion)
		}
		if created.CreationTimestamp.IsZero() {
			t.Errorf("expected CreationTimestamp to be set, got %s", created.CreationTimestamp)
		}

		// Data integrity
		if created.Name != gpu1.Name {
			t.Errorf("expected Name %q, got %q", gpu1.Name, created.Name)
		}
		if created.Spec.UUID != gpu1.Spec.UUID {
			t.Errorf("expected Spec.UUID %q, got %q", gpu1.Spec.UUID, created.Spec.UUID)
		}

		expectedCondCount := len(gpu1.Status.Conditions)
		actualCondCount := len(created.Status.Conditions)
		if actualCondCount != expectedCondCount {
			t.Fatalf("expected %d condition, got %d", expectedCondCount, actualCondCount)
		}

		expectedCond := gpu1.Status.Conditions[0]
		actualCond := created.Status.Conditions[0]
		if actualCond.Type != expectedCond.Type {
			t.Errorf("expected Status.Condition.Type %q, got %q", expectedCond.Type, actualCond.Type)
		}
		if actualCond.Status != expectedCond.Status {
			t.Errorf("expected Status.Condition.Status %q, got %q", expectedCond.Status, actualCond.Status)
		}
		if actualCond.Reason != expectedCond.Reason {
			t.Errorf("expected Status.Condition.Reason %q, got %q", expectedCond.Reason, actualCond.Reason)
		}
		if actualCond.Message != expectedCond.Message {
			t.Errorf("expected Status.Condition.Message %q, got %q", expectedCond.Message, actualCond.Message)
		}
		if actualCond.LastTransitionTime.IsZero() {
			t.Errorf("expected Status.LastTransitionTime to be set, got %s", actualCond.LastTransitionTime)
		}
	})

	t.Run("Update", func(t *testing.T) {
		toUpdate := created.DeepCopy()
		toUpdate.Spec.UUID = "GPU-1-Updated"
		// status *must* be ignored on update
		toUpdate.Status.Conditions = []metav1.Condition{}

		updated, err := client.DeviceV1alpha1().GPUs().Update(ctx, toUpdate, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("unexpected error updating GPU: %v", err)
		}

		// Immutable fields *must* not change
		if updated.UID != created.UID {
			t.Errorf("got UID = %q, want %q", updated.UID, created.UID)
		}
		if !updated.CreationTimestamp.Equal(&created.CreationTimestamp) {
			t.Errorf("got CreationTimestamp = %s, want %s", updated.CreationTimestamp, created.CreationTimestamp)
		}

		// Versioning field *must* change
		if updated.ResourceVersion == created.ResourceVersion {
			t.Errorf("ResourceVersion did not change on update: %s", created.ResourceVersion)
		}
		if updated.Generation == created.Generation {
			t.Errorf("Generation did not change on update: %s", created.ResourceVersion)
		}

		// Spec
		if updated.Spec.UUID != "GPU-1-Updated" {
			t.Errorf("got Spec.UUID = %q, want %q", updated.Spec.UUID, "GPU-1-Updated")
		}

		// Status *must* not change
		if !reflect.DeepEqual(updated.Status, created.Status) {
			t.Errorf("got Status = %+v, want %+v", updated.Status, created.Status)
		}

		created = updated
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		toUpdateStatus := created.DeepCopy()

		newCondition := metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "DriverReady",
			Message:            "Driver is posting ready status",
			LastTransitionTime: metav1.Now(),
		}
		toUpdateStatus.Status.Conditions = []metav1.Condition{newCondition}

		updated, err := client.DeviceV1alpha1().GPUs().UpdateStatus(ctx, toUpdateStatus, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("unexpected error updating GPU status: %v", err)
		}

		if updated.ResourceVersion == created.ResourceVersion {
			t.Errorf("ResourceVersion did not change on status update")
		}

		if updated.Generation != created.Generation {
			t.Errorf("Generation changed from %d to %d; status updates must not increment generation",
				created.Generation, updated.Generation)
		}

		if !reflect.DeepEqual(updated.Spec, created.Spec) {
			t.Errorf("got Spec = %+v, want %+v", updated.Spec, created.Spec)
		}

		if len(updated.Status.Conditions) != 1 {
			t.Fatalf("expected 1 condition, got %d", len(updated.Status.Conditions))
		}

		updatedCond := updated.Status.Conditions[0]
		if updatedCond.Type != newCondition.Type {
			t.Errorf("expected Status.Condition.Type %q, got %q", newCondition.Type, updatedCond.Type)
		}
		if updatedCond.Status != newCondition.Status {
			t.Errorf("expected Status.Condition.Status %q, got %q", newCondition.Status, updatedCond.Status)
		}
		if updatedCond.Reason != newCondition.Reason {
			t.Errorf("expected Status.Condition.Reason %q, got %q", newCondition.Reason, updatedCond.Reason)
		}
		if updatedCond.Message != newCondition.Message {
			t.Errorf("expected Status.Condition.Message %q, got %q", newCondition.Message, updatedCond.Message)
		}
		if !updatedCond.LastTransitionTime.Round(time.Second).Equal(newCondition.LastTransitionTime.Round(time.Second)) {
			t.Errorf("expected Status.Condition.LastTransitionTime %q, got %q", newCondition.LastTransitionTime, updatedCond.LastTransitionTime)
		}

		created = updated
	})

	t.Run("List", func(t *testing.T) {
		gpu2 := &devicev1alpha1.GPU{
			ObjectMeta: metav1.ObjectMeta{Name: "gpu-2"},
			Spec:       devicev1alpha1.GPUSpec{UUID: "GPU-2"},
		}
		_, err := client.DeviceV1alpha1().GPUs().Create(ctx, gpu2, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("unexpected error creating second GPU: %v", err)
		}

		list, err := client.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("unexpected error listing GPUs: %v", err)
		}

		if len(list.Items) != 2 {
			t.Errorf("len(list.Items) = %d, want 2", len(list.Items))
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := client.DeviceV1alpha1().GPUs().Delete(ctx, created.Name, metav1.DeleteOptions{})
		if err != nil {
			t.Fatalf("unexpected error deleting GPU: %v", err)
		}

		_, err = client.DeviceV1alpha1().GPUs().Get(ctx, created.Name, metav1.GetOptions{})
		if !errors.IsNotFound(err) {
			t.Errorf("Get(%q) err = %v, want NotFound", created.Name, err)
		}

		err = client.DeviceV1alpha1().GPUs().Delete(ctx, created.Name, metav1.DeleteOptions{})
		if !errors.IsNotFound(err) {
			t.Errorf("Delete(%q) = nil, want NotFound", created.Name)
		}

		list, err := client.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("unexpected error listing GPUs after delete: %v", err)
		}

		for _, item := range list.Items {
			if item.Name == created.Name {
				t.Errorf("List() contains %q, want it removed", created.Name)
			}
		}
	})
}
