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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func TestWatchTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gpu := &devicev1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "timeout-gpu"}}
	if _, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{}); err != nil {
		t.Fatalf("failed to create GPU %s: %v", "timeout-gpu", err)
	}
	defer clientset.DeviceV1alpha1().GPUs().Delete(context.Background(), gpu.Name, metav1.DeleteOptions{})

	stopCh := make(chan struct{})
	watchCount := 0
	timeout := int64(1)

	lw := &cache.ListWatch{
		ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
			return clientset.DeviceV1alpha1().GPUs().List(ctx, opts)
		},
		WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
			opts.TimeoutSeconds = &timeout
			watchCount++
			if watchCount > 1 {
				close(stopCh)
			}
			return clientset.DeviceV1alpha1().GPUs().Watch(ctx, opts)
		},
	}

	_, informer := cache.NewIndexerInformer(lw, &devicev1alpha1.GPU{}, 0, cache.ResourceEventHandlerFuncs{}, cache.Indexers{})
	go informer.Run(stopCh)

	select {
	case <-stopCh:
	case <-time.After(3 * time.Second):
		t.Fatal("informer failed to restart watch")
	}
}
