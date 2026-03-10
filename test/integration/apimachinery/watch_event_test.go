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
	"github.com/nvidia/nvsentinel/pkg/client-go/client/versioned"
	"github.com/nvidia/nvsentinel/test/integration/framework"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestWatchEvent(t *testing.T) {
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

			timeout := 15 * time.Second

			newWatcher := func(ctx context.Context, gpu *devicev1alpha1.GPU, resourceVersion string) (watch.Interface, error) {
				return clientset.DeviceV1alpha1().GPUs().Watch(ctx, metav1.ListOptions{
					ResourceVersion: resourceVersion,
				})
			}

			testCases := []struct {
				scenarioName    string
				gpuName         string
				resourceVersion string
			}{
				{scenarioName: "watch object with resource version 0", gpuName: fmt.Sprintf("%s-rv0", tc.name), resourceVersion: "0"},
			}

			for _, sc := range testCases {
				t.Run(sc.scenarioName, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), timeout)
					defer cancel()

					testGPU := &devicev1alpha1.GPU{
						ObjectMeta: metav1.ObjectMeta{Name: sc.gpuName},
						Spec:       devicev1alpha1.GPUSpec{UUID: "initial-uuid"},
					}

					gpu, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, testGPU, metav1.CreateOptions{})
					if err != nil {
						t.Fatalf("failed to create GPU %s: %v", sc.gpuName, err)
					}
					defer clientset.DeviceV1alpha1().GPUs().Delete(context.Background(), gpu.Name, metav1.DeleteOptions{})

					watcher, err := newWatcher(ctx, gpu, sc.resourceVersion)
					if err != nil {
						t.Fatalf("failed to create watcher: %v", err)
					}
					defer watcher.Stop()

					generateAndWatchEvent(ctx, t, tc, clientset, gpu, watcher, tc.storageBackend)
				})
			}
		})
	}
}

func generateAndWatchEvent(ctx context.Context, t *testing.T, tc struct {
	name           string
	storageBackend string
}, cs versioned.Interface, gpu *devicev1alpha1.GPU, watcher watch.Interface, backend string) {
	timeout := 10 * time.Second
	gpuName := gpu.Name

	_, ok := waitForEvent(watcher, watch.Added, gpu, timeout, backend, t)
	if !ok {
		if tc.storageBackend == "memory" {
			framework.SkipWithWarning(t, fmt.Sprintf("failed to observe ADDED event for GPU %s", gpuName))
		}
		t.Fatalf("failed to observe ADDED event for GPU %s", gpuName)
	}

	gpu.Spec.UUID = "updated-uuid"
	gpu, err := cs.DeviceV1alpha1().GPUs().Update(ctx, gpu, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("failed to update GPU %s: %v", gpuName, err)
	}

	_, ok = waitForEvent(watcher, watch.Modified, gpu, timeout, backend, t)
	if !ok {
		t.Fatalf("failed to observe first MODIFIED event for %s", gpuName)
	}

	gpu.Spec.UUID = "updated-again-uuid"
	gpu, err = cs.DeviceV1alpha1().GPUs().Update(ctx, gpu, metav1.UpdateOptions{})
	if err != nil {
		t.Fatalf("failed to update GPU %s: %v", gpuName, err)
	}

	_, ok = waitForEvent(watcher, watch.Modified, gpu, timeout, backend, t)
	if !ok {
		t.Fatalf("failed to observe second MODIFIED event for %s", gpuName)
	}
}

func waitForEvent(w watch.Interface, expectType watch.EventType, expectObject *devicev1alpha1.GPU, duration time.Duration, backend string, t *testing.T) (watch.Event, bool) {
	stopTimer := time.NewTimer(duration)
	defer stopTimer.Stop()
	for {
		select {
		case actual, ok := <-w.ResultChan():
			if !ok {
				return watch.Event{}, false
			}

			if actual.Type == watch.Error {
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
