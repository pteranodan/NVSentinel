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
	"github.com/nvidia/nvsentinel/pkg/client-go/client/versioned"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
)

func TestWatchEvent(t *testing.T) {
	timeout := 10 * time.Second

	newWatcher := func(ctx context.Context, gpu *devicev1alpha1.GPU, resourceVersion string) (watch.Interface, error) {
		return clientset.DeviceV1alpha1().GPUs().Watch(ctx, metav1.ListOptions{
			FieldSelector:   fields.OneTermEqualSelector("metadata.name", gpu.Name).String(),
			ResourceVersion: resourceVersion,
		})
	}

	newTestGPU := func(name string) *devicev1alpha1.GPU {
		return &devicev1alpha1.GPU{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: devicev1alpha1.GPUSpec{
				UUID: "uuid",
			},
		}
	}

	tt := []struct {
		name            string
		gpu             *devicev1alpha1.GPU
		resourceVersion string
	}{
		{
			name: "watch object by name",
			gpu:  newTestGPU("watch-gpu-direct"),
		},
		{
			name:            "watch object with resource version 0",
			gpu:             newTestGPU("watch-gpu-with-rv0"),
			resourceVersion: "0",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			gpu, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, tc.gpu, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create GPU %s: %v", tc.gpu.Name, err)
			}
			defer clientset.DeviceV1alpha1().GPUs().Delete(context.Background(), gpu.Name, metav1.DeleteOptions{})

			watcher, err := newWatcher(ctx, gpu, tc.resourceVersion)
			if err != nil {
				t.Fatalf("failed to create watcher: %v", err)
			}
			defer watcher.Stop()

			generateAndWatchEvent(ctx, t, clientset, gpu, watcher)
		})
	}
}

func generateAndWatchEvent(ctx context.Context, t *testing.T, cs versioned.Interface, gpu *devicev1alpha1.GPU, watcher watch.Interface) {
	timeout := 10 * time.Second
	gpuName := gpu.Name

	_, ok := waitForEvent(watcher, watch.Added, gpu, timeout)
	if !ok {
		t.Fatalf("failed to observe ADDED event for %s", gpuName)
	}

	gpu.Spec.UUID = "updated-uuid"
	gpu, err := cs.DeviceV1alpha1().GPUs().Update(ctx, gpu, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("failed to update GPU %s: %v", gpuName, err)
	}

	_, ok = waitForEvent(watcher, watch.Modified, gpu, timeout)
	if !ok {
		t.Fatalf("failed to observe first MODIFIED event for %s", gpuName)
	}

	gpu.Spec.UUID = "updated-again-uuid"
	gpu, err = cs.DeviceV1alpha1().GPUs().Update(ctx, gpu, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("failed to update GPU %s: %v", gpuName, err)
	}

	_, ok = waitForEvent(watcher, watch.Modified, gpu, timeout)
	if !ok {
		t.Fatalf("failed to observe second MODIFIED event for %s", gpuName)
	}
}

func waitForEvent(w watch.Interface, expectType watch.EventType, expectObject *devicev1alpha1.GPU, duration time.Duration) (watch.Event, bool) {
	stopTimer := time.NewTimer(duration)
	defer stopTimer.Stop()
	for {
		select {
		case actual, ok := <-w.ResultChan():
			if !ok {
				return watch.Event{}, false
			}

			actualObj, ok := actual.Object.(*devicev1alpha1.GPU)
			if !ok {
				continue
			}

			typeMatches := actual.Type == expectType
			objectMatches := expectObject == nil || apiequality.Semantic.DeepEqual(expectObject, actualObj)

			if typeMatches && objectMatches {
				return actual, true
			}

		case <-stopTimer.C:
			return watch.Event{}, false
		}
	}
}
