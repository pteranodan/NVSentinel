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
	"strings"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/test/integration/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/client-go/tools/cache"
)

func TestWatchTimeout(t *testing.T) {
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

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			gpuName := fmt.Sprintf("%s-timeout-gpu", tc.name)
			gpu := &devicev1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: gpuName}}
			if _, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{}); err != nil {
				t.Fatalf("failed to create GPU: %v", err)
			}

			stopCh := make(chan struct{})
			doneCh := make(chan struct{})
			watchCount := 0
			timeoutSeconds := int64(1)

			var lastWatchError error
			lw := &cache.ListWatch{
				ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
					return clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
				},
				WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
					opts.TimeoutSeconds = &timeoutSeconds
					watchCount++

					w, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, opts)
					if err != nil {
						return nil, err
					}

					if watchCount > 1 {
						select {
						case <-doneCh:
						default:
							close(doneCh)
							close(stopCh)
						}
					}
					return &errorCapturingWatcher{
						Interface: w,
						onClose: func(err error) {
							lastWatchError = err
							select {
							case <-doneCh:
							default:
								close(doneCh)
								close(stopCh)
							}
						},
					}, nil
				},
			}

			_, informer := cache.NewIndexerInformer(lw, &devicev1alpha1.GPU{}, 0, cache.ResourceEventHandlerFuncs{}, cache.Indexers{})
			go informer.Run(stopCh)

			select {
			case <-doneCh:
				if lastWatchError != nil {
					errStr := strings.ToLower(lastWatchError.Error())
					validError := strings.Contains(errStr, "deadline exceeded") ||
						strings.Contains(errStr, "context canceled") ||
						strings.Contains(errStr, "timeout")

					if !validError {
						t.Errorf("expected timeout related error, got: %v", lastWatchError)
					}
				}
			case <-time.After(3 * time.Second):
				t.Errorf("informer failed to restart watch after %s", 3*time.Second)
			}
		})
	}
}

// errorCapturingWatcher wraps a watch.Interface to detect closure
type errorCapturingWatcher struct {
	watch.Interface
	onClose func(error)
}

func (w *errorCapturingWatcher) Stop() {
	w.Interface.Stop()
}
