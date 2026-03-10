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

func TestPatchType(t *testing.T) {
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

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			gpuName := fmt.Sprintf("%s-patch-type", tc.name)
			gpu := &devicev1alpha1.GPU{
				ObjectMeta: metav1.ObjectMeta{Name: gpuName},
				Spec:       devicev1alpha1.GPUSpec{UUID: "patch-type-uuid"},
			}

			_, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create GPU: %v", err)
			}

			mergePatchData := []byte(`{"spec":{"uuid":"merge-patch-uuid"}}`)
			patched, err := clientset.DeviceV1alpha1().GPUs().Patch(ctx, gpuName, types.MergePatchType, mergePatchData, metav1.PatchOptions{})
			if err != nil {
				t.Fatalf("failed to patch GPU (%s): %v", types.MergePatchType, err)
			}
			if patched.Spec.UUID != "merge-patch-uuid" {
				t.Errorf("expected UUID 'merge-patch-uuid' after patch (%s), got %s", types.MergePatchType, patched.Spec.UUID)
			}

			jsonPatchData := []byte(`[{"op": "replace", "path": "/spec/uuid", "value": "json-patch-uuid"}]`)
			patched, err = clientset.DeviceV1alpha1().GPUs().Patch(ctx, gpuName, types.JSONPatchType, jsonPatchData, metav1.PatchOptions{})
			if err != nil {
				t.Fatalf("failed to patch GPU (%s): %v", types.JSONPatchType, err)
			}
			if patched.Spec.UUID != "json-patch-uuid" {
				t.Errorf("expected UUID 'json-patch-uuid' after patch (%s), got %s", types.JSONPatchType, patched.Spec.UUID)
			}

			strategicData := []byte(`{"spec":{"uuid":"strategic"}}`)
			_, err = clientset.DeviceV1alpha1().GPUs().Patch(ctx, gpuName, types.StrategicMergePatchType, strategicData, metav1.PatchOptions{})
			if err == nil {
				t.Fatalf("expected error for patch (%s), got nil", types.StrategicMergePatchType)
			}
			if !errors.IsUnsupportedMediaType(err) {
				t.Errorf("expected UnsupportedMediaType (415) for patch (%s), got: %v", types.StrategicMergePatchType, err)
			}

			applyData := []byte(`{"apiVersion":"device.nvidia.com/v1alpha1","kind":"GPU","metadata":{"name":"` + gpuName + `"},"spec":{"uuid":"ssa"}}`)
			_, err = clientset.DeviceV1alpha1().GPUs().Patch(ctx, gpuName, types.ApplyPatchType, applyData, metav1.PatchOptions{FieldManager: "test-manager"})
			if err == nil {
				t.Fatalf("expected error for patch (%s), got nil", types.ApplyPatchType)
			}
			if !errors.IsUnsupportedMediaType(err) {
				t.Errorf("expected UnsupportedMediaType (415) for patch (%s), got: %v", types.ApplyPatchType, err)
			}
		})
	}
}
