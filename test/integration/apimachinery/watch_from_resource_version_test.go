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
	"k8s.io/apimachinery/pkg/watch"
)

func TestWatchFromResourceVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	gpu1 := &devicev1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-watch-rv"},
		Spec:       devicev1alpha1.GPUSpec{UUID: "watch-rv"},
	}
	createdGpu1, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu1, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create GPU %s: %v", gpu1.Name, err)
	}

	opts := metav1.ListOptions{
		ResourceVersion: createdGpu1.ResourceVersion,
	}
	watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, opts)
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}
	defer watcher.Stop()

	gpu2 := &devicev1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-watch-rv-2"},
		Spec:       devicev1alpha1.GPUSpec{UUID: "watch-rv-2"},
	}
	createdGpu2, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu2, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create GPU %s: %v", gpu2.Name, err)
	}

	select {
	case event, ok := <-watcher.ResultChan():
		if !ok {
			t.Fatal("watch channel closed prematurely")
		}
		obj := event.Object.(*devicev1alpha1.GPU)
		if obj.Name != createdGpu2.Name {
			t.Errorf("failed to observe event for %s, got %s", createdGpu2.Name, obj.Name)
		}
		if event.Type != watch.Added {
			t.Errorf("failed to observe ADDED event, got %v", event.Type)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting to observe event from RV (%s)", createdGpu1.ResourceVersion)
	}
}
