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
	"reflect"
	"sync"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type GPUSharedInformerFactory interface {
	Start(stopCh <-chan struct{})
	WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
	GPU() GPUInformer
}

// GPUInformer provides access to a shared informer and lister for GPUs.
type GPUInformer interface {
	// Informer returns the shared informer, creating it on first use.
	Informer() cache.SharedIndexInformer

	Lister() GPULister
}

// GPUInformerOptions configures a GPU informer.
type GPUInformerOptions struct {
	// ResyncPeriod is the interval for full resyncs. Zero disables periodic syncs.
	ResyncPeriod time.Duration

	// Indexers provides additional indices for the informer cache.
	Indexers cache.Indexers
}

// gpuInformer implements GPUInformer.
type gpuInformer struct {
	client GPU

	mu           sync.Mutex
	informer     cache.SharedIndexInformer
	resyncPeriod time.Duration
	indexers     cache.Indexers
}

var _ GPUInformer = (*gpuInformer)(nil)

// NewGPUInformer returns a GPUInformer with the provided client and options.
func NewGPUInformer(client GPU, opts GPUInformerOptions) GPUInformer {
	if opts.Indexers == nil {
		opts.Indexers = cache.Indexers{}
	}

	return &gpuInformer{
		client:       client,
		resyncPeriod: opts.ResyncPeriod,
		indexers:     opts.Indexers,
	}
}

// Informer returns the shared informer, creating it if needed.
func (i *gpuInformer) Informer() cache.SharedIndexInformer {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.informer != nil {
		return i.informer
	}

	if i.indexers == nil {
		i.indexers = cache.Indexers{}
	}

	i.informer = cache.NewSharedIndexInformer(
		&gpuListerWatcher{client: i.client},
		&devicev1alpha1.GPU{},
		i.resyncPeriod,
		i.indexers,
	)

	return i.informer
}

// InformerWithOptions rebuilds the informer using the given options and
// replaces the cached informer.
func (i *gpuInformer) InformerWithOptions(opts GPUInformerOptions) cache.SharedIndexInformer {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.resyncPeriod = opts.ResyncPeriod

	if opts.Indexers != nil {
		i.indexers = opts.Indexers
	} else {
		i.indexers = cache.Indexers{}
	}

	i.informer = cache.NewSharedIndexInformer(
		&gpuListerWatcher{client: i.client},
		&devicev1alpha1.GPU{},
		i.resyncPeriod,
		i.indexers,
	)

	return i.informer
}

// gpuListerWatcher implements cache.ListerWatcher.
type gpuListerWatcher struct {
	client GPU
}

var _ cache.ListerWatcher = (*gpuListerWatcher)(nil)

// List lists GPU resources. Only ResourceVersion in opts is respected.
func (lw *gpuListerWatcher) List(opts metav1.ListOptions) (runtime.Object, error) {
	return lw.client.List(context.Background(), opts)
}

// Watch starts a watch for GPU resources. Canceling the context stops the watch.
func (lw *gpuListerWatcher) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return lw.client.Watch(context.Background(), opts)
}
