// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package v1alpha1

import (
	"context"
	"testing"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache/informertest"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestGPUSource_Enqueues(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	f := &informertest.FakeInformers{}
	informer, err := f.FakeInformerFor(ctx, &devicev1alpha1.GPU{})
	if err != nil {
		t.Fatalf("failed to create fake informer: %v", err)
	}

	q := workqueue.NewTypedRateLimitingQueueWithConfig(
		workqueue.DefaultTypedControllerRateLimiter[reconcile.Request](),
		workqueue.TypedRateLimitingQueueConfig[reconcile.Request]{Name: "test"},
	)

	var createdObjs []*devicev1alpha1.GPU
	var updatedObjs []*devicev1alpha1.GPU
	var deletedObjs []*devicev1alpha1.GPU

	gs := &GPUSource{Informer: informer}
	err = gs.Start(ctx, handler.TypedFuncs[*devicev1alpha1.GPU, reconcile.Request]{
		CreateFunc: func(ctx context.Context, e event.TypedCreateEvent[*devicev1alpha1.GPU], q2 workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			createdObjs = append(createdObjs, e.Object)
		},
		UpdateFunc: func(ctx context.Context, e event.TypedUpdateEvent[*devicev1alpha1.GPU], q2 workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			updatedObjs = append(updatedObjs, e.ObjectNew)
		},
		DeleteFunc: func(ctx context.Context, e event.TypedDeleteEvent[*devicev1alpha1.GPU], q2 workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			deletedObjs = append(deletedObjs, e.Object)
		},
	})
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	obj := &devicev1alpha1.GPU{
		ObjectMeta: v1.ObjectMeta{Name: "gpu0", Namespace: "default"},
	}
	informer.Add(obj)
	if len(createdObjs) != 1 || createdObjs[0] != obj {
		t.Fatalf("expected Create event for obj, got %+v", createdObjs)
	}

	updated := obj.DeepCopy()
	updated.Labels = map[string]string{"a": "b"}
	informer.Update(obj, updated)
	if len(updatedObjs) != 1 || updatedObjs[0] != updated {
		t.Fatalf("expected Update event for obj, got %+v", updatedObjs)
	}

	informer.Delete(updated)
	if len(deletedObjs) != 1 || deletedObjs[0] != updated {
		t.Fatalf("expected Delete event for obj, got %+v", deletedObjs)
	}
}
