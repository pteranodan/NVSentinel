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
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListFromResourceVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gpu1 := &devicev1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: "list-from-rv"},
		Spec:       devicev1alpha1.GPUSpec{UUID: "list-from-rv"},
	}
	created1, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("unexpected error creating GPU: %v", err)
	}
	rv1 := created1.ResourceVersion

	gpu2 := &devicev1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: "list-from-rv-2"},
		Spec:       devicev1alpha1.GPUSpec{UUID: "list-from-rv-2"},
	}
	_, err = clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("unexpected error creating GPU: %v", err)
	}

	t.Run("RV=0 (Any)", func(t *testing.T) {
		opts := metav1.ListOptions{ResourceVersion: "0"}
		list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
		if err != nil {
			t.Fatalf("List(RV=0) failed: %v", err)
		}
		if len(list.Items) == 0 {
			t.Error("List(RV=0) returned empty list, expected items")
		}
	})

	t.Run("RV=Old (NotOlderThan)", func(t *testing.T) {
		opts := metav1.ListOptions{ResourceVersion: rv1}
		list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
		if err != nil {
			t.Fatalf("List(RV=%s) failed: %v", rv1, err)
		}

		foundGpu2 := false
		for _, item := range list.Items {
			if item.Name == gpu2.Name {
				foundGpu2 = true
				break
			}
		}

		if !foundGpu2 {
			t.Errorf("List(RV=%s, Match=Default) did not return newer item %q. It behaved like 'Exact' match!", rv1, gpu2.Name)
		}
	})

	t.Run("RV=Old (Exact)", func(t *testing.T) {
		optsExact := metav1.ListOptions{
			ResourceVersion:      rv1,
			ResourceVersionMatch: metav1.ResourceVersionMatchExact,
		}
		listExact, err := clientset.DeviceV1alpha1().GPUs().List(ctx, optsExact)
		if err != nil {
			t.Fatalf("unexpected error listing GPUs at exact RV %s: %v", rv1, err)
		}

		foundGpu1 := false
		foundGpu2 := false
		for _, item := range listExact.Items {
			if item.Name == gpu1.Name {
				foundGpu1 = true
			}
			if item.Name == gpu2.Name {
				foundGpu2 = true
			}
		}

		if !foundGpu1 {
			t.Errorf("List(RV=%s) missing baseline object %q", rv1, gpu1.Name)
		}
		if foundGpu2 {
			t.Errorf("List(RV=%s) leaked future object %q; snapshot isolation failed", rv1, gpu2.Name)
		}
	})

	t.Run("RV=Empty (Live)", func(t *testing.T) {
		listLive, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
		if err != nil {
			t.Fatalf("unexpected error listing GPUs: %v", err)
		}

		foundGpu2Live := false
		for _, item := range listLive.Items {
			if item.Name == gpu2.Name {
				foundGpu2Live = true
				break
			}
		}
		if !foundGpu2Live {
			t.Errorf("List() missing newest object %q", gpu2.Name)
		}
	})
}
