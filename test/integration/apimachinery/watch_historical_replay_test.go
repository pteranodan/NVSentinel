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

func TestWatchHistoricalReplay(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to list GPUs: %v", err)
	}
	snapshotRV := list.ResourceVersion

	gpuMissed := &devicev1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-missed"},
		Spec:       devicev1alpha1.GPUSpec{UUID: "missed-uuid"},
	}
	createdMissed, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpuMissed, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to create GPU %s: %v", gpuMissed.Name, err)
	}

	watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, metav1.ListOptions{
		ResourceVersion: snapshotRV,
	})
	if err != nil {
		t.Fatalf("failed to start watch from historical RV: %v", err)
	}
	defer watcher.Stop()

	found := false
	timeout := time.After(5 * time.Second)
	for !found {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				t.Fatal("watch closed during replay")
			}
			obj := event.Object.(*devicev1alpha1.GPU)
			if obj.Name == createdMissed.Name && event.Type == watch.Added {
				found = true
			}
		case <-timeout:
			t.Fatalf("failed to replay historical event for %s from RV %s", createdMissed.Name, snapshotRV)
		}
	}
}
