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
	"fmt"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// GPUSource is a source of events (e.g. Create, Update, Delete operations on GPU Objects)
// which should be processed by event.EventHandlers to enqueue reconcile.Requests.
type GPUSource struct {
	// Informer is the specific shared informer for the GPU resource.
	Informer cache.SharedIndexInformer

	// Handler enqueues reconcile.Requests in response to events (e.g. GPU Update).
	Handler handler.EventHandler

	// Predicate filters events before enqueuing the keys.
	Predicates []predicate.Predicate
}

var _ source.Source = (*GPUSource)(nil)

// Start should be called only by the Controller to register an EventHandler
// with the Informer to enqueue reconcile.Requests.
func (s *GPUSource) Start(
	ctx context.Context,
	queue workqueue.TypedRateLimitingInterface[reconcile.Request],
) error {
	if s.Informer == nil {
		return fmt.Errorf("must specify GPUSource.Informer")
	}
	if s.Handler == nil {
		return fmt.Errorf("must specify GPUSource.Handler")
	}

	_, err := s.Informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cObj, ok := obj.(client.Object)
			if !ok {
				return
			}

			evt := event.CreateEvent{Object: cObj}
			for _, predicate := range s.Predicates {
				if !predicate.Create(evt) {
					return
				}
			}

			s.Handler.Create(ctx, evt, queue)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			cOldObj, ok1 := oldObj.(client.Object)
			cNewObj, ok2 := newObj.(client.Object)
			if !ok1 || !ok2 {
				return
			}

			evt := event.UpdateEvent{ObjectOld: cOldObj, ObjectNew: cNewObj}
			for _, predicate := range s.Predicates {
				if !predicate.Update(evt) {
					return
				}
			}

			s.Handler.Update(ctx, evt, queue)
		},
		DeleteFunc: func(obj interface{}) {
			if tombstone, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = tombstone.Obj
			}

			cObj, ok := obj.(client.Object)
			if !ok {
				return
			}

			evt := event.DeleteEvent{Object: cObj}
			for _, predicate := range s.Predicates {
				if !predicate.Delete(evt) {
					return
				}
			}

			s.Handler.Delete(ctx, evt, queue)
		},
	})

	if err != nil {
		return fmt.Errorf("failed to add handler: %w", err)
	}

	// Context (ctx) passed here is the Controller Manager's context,
	// so the informer will stop automatically when the manager stops.
	go func() {
		s.Informer.Run(ctx.Done())
	}()

	// Start must be non-blocking.
	go func() {
		logger := log.FromContext(ctx).WithName("gpu-source")

		if !cache.WaitForCacheSync(ctx.Done(), s.Informer.HasSynced) {
			return
		}
		logger.Info("GPU informer cache synced successfully")
	}()

	return nil
}
