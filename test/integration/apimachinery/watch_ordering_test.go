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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWatchOrdering(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, _ := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
	startRVStr := list.ResourceVersion
	startRV, _ := strconv.ParseUint(startRVStr, 10, 64)

	watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, metav1.ListOptions{
		ResourceVersion: startRVStr,
	})
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}
	defer watcher.Stop()

	names := []string{"gpu-seq-1", "gpu-seq-2", "gpu-seq-3", "gpu-seq-4"}
	for _, name := range names {
		g := &devicev1alpha1.GPU{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       devicev1alpha1.GPUSpec{UUID: "seq-uuid"},
		}
		if _, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, g, metav1.CreateOptions{}); err != nil {
			t.Fatalf("failed to create GPU %s: %v", name, err)
		}
	}

	lastRV := startRV
	for i, expectedName := range names {
		select {
		case event := <-watcher.ResultChan():
			obj := event.Object.(*devicev1alpha1.GPU)
			currentRV, _ := strconv.ParseUint(obj.ResourceVersion, 10, 64)

			if obj.Name != expectedName {
				t.Errorf("expected %s as event %d, got %s", expectedName, i, obj.Name)
			}

			if currentRV <= lastRV {
				t.Fatalf("expected RV (%d) to be greater than previous RV (%d): %s",
					currentRV, lastRV, obj.Name)
			}
			lastRV = currentRV

		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for event %d: %s", i, expectedName)
		}
	}
}
