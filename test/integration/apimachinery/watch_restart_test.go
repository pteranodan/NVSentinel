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
	"fmt"
	"reflect"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/pkg/client-go/client/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
)

func TestWatchRestarts(t *testing.T) {
	timeout := 1 * time.Second
	initialUUID := "gpu-initial-uuid"
	const numEvents = 5

	generateEvents := func(t *testing.T, c versioned.Interface, gpu *devicev1alpha1.GPU, expected *[]string, stopChan chan struct{}, stoppedChan chan struct{}) {
		defer close(stoppedChan)

		for counter := 1; counter <= numEvents; counter++ {
			select {
			case <-time.After(25 * time.Millisecond):
				counter++
				newUUID := fmt.Sprintf("UUID-%d", counter)

				current, err := c.DeviceV1alpha1().GPUs().Get(context.TODO(), gpu.Name, metav1.GetOptions{})
				if err != nil {
					t.Errorf("failed to get GPU %s: %v", gpu.Name, err)
					return
				}

				current.Spec.UUID = newUUID
				_, err = c.DeviceV1alpha1().GPUs().Update(context.TODO(), current, metav1.UpdateOptions{})
				if err != nil {
					t.Errorf("failed to update GPU %s: %v", current.Name, err)
					return
				}

				*expected = append(*expected, newUUID)
			}
		}
	}

	newTestGPU := func(name string) *devicev1alpha1.GPU {
		return &devicev1alpha1.GPU{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: devicev1alpha1.GPUSpec{
				UUID: initialUUID,
			},
		}
	}

	tt := []struct {
		name                string
		succeed             bool
		gpu                 *devicev1alpha1.GPU
		getWatcher          func(c versioned.Interface, gpu *devicev1alpha1.GPU) (watch.Interface, error, func())
		normalizeOutputFunc func(expected []string) []string
	}{
		{
			name:    "Watcher fails on connection close",
			succeed: false,
			gpu:     newTestGPU("watcher"),
			getWatcher: func(c versioned.Interface, gpu *devicev1alpha1.GPU) (watch.Interface, error, func()) {
				w, err := c.DeviceV1alpha1().GPUs().Watch(context.TODO(), metav1.ListOptions{
					ResourceVersion: gpu.ResourceVersion,
				})
				return w, err, noop
			},
			normalizeOutputFunc: noopNormalization,
		},
		{
			name:    "RetryWatcher survives closed watches",
			succeed: true,
			gpu:     newTestGPU("retry-watcher"),
			getWatcher: func(c versioned.Interface, gpu *devicev1alpha1.GPU) (watch.Interface, error, func()) {
				lw := &cache.ListWatch{
					WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
						return c.DeviceV1alpha1().GPUs().Watch(context.TODO(), options)
					},
				}
				w, err := watchtools.NewRetryWatcher(gpu.ResourceVersion, lw)
				return w, err, func() { <-w.Done() }
			},
			normalizeOutputFunc: noopNormalization,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			gpu, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, tc.gpu, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("failed to create GPU %s: %v", tc.gpu.Name, err)
			}
			defer clientset.DeviceV1alpha1().GPUs().Delete(ctx, gpu.Name, metav1.DeleteOptions{})

			watcher, err, doneFn := tc.getWatcher(clientset, gpu)
			if err != nil {
				t.Fatalf("failed to create watcher: %v", err)
			}
			defer doneFn()

			var expected []string
			var actual []string
			stopChan := make(chan struct{})
			stoppedChan := make(chan struct{})

			go generateEvents(t, clientset, gpu, &expected, stopChan, stoppedChan)

			watchCtx, cancel := watchtools.ContextWithOptionalTimeout(ctx, timeout)
			defer cancel()

			_, err = watchtools.UntilWithoutRetry(watchCtx, watcher, func(event watch.Event) (bool, error) {
				obj, ok := event.Object.(*devicev1alpha1.GPU)
				if !ok {
					return false, fmt.Errorf("unexpected type: %T", event.Object)
				}
				if obj.Spec.UUID != initialUUID {
					actual = append(actual, obj.Spec.UUID)
				}

				if len(actual) >= numEvents {
					return true, nil
				}
				return false, nil
			})

			close(stopChan)
			<-stoppedChan

			actual = tc.normalizeOutputFunc(actual)

			if err == nil && !tc.succeed {
				t.Fatalf("expected Watch error")
			}

			if !wait.Interrupted(err) && tc.succeed {
				t.Fatalf("unexpected RetryWatch error: %v", err)
			}

			if tc.succeed && !reflect.DeepEqual(expected, actual) {
				t.Fatalf("unexpected Event\ngot: %#v\nwant: %#v", actual, expected)
			}
		})
	}
}

func noopNormalization(actual []string) []string {
	return actual
}

// normalizeInformerOutputFunc removes repetitions often caused by Informer relists
func normalizeInformerOutputFunc(initialVal string) func(actual []string) []string {
	return func(actual []string) []string {
		result := make([]string, 0, len(actual))
		lastVal := initialVal
		for _, v := range actual {
			if v == lastVal {
				continue
			}
			result = append(result, v)
			lastVal = v
		}
		return result
	}
}

func noop() {}
