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
	"fmt"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

// GPULister lists and gets GPU resources from a shared informer cache.
type GPULister interface {
	// List returns all GPUs matching the given label selector.
	List(selector labels.Selector) ([]*devicev1alpha1.GPU, error)

	// Get retrieves a GPU by name. Returns NotFound if it does not exist.
	Get(name string) (*devicev1alpha1.GPU, error)
}

// gpuLister implements GPULister.
type gpuLister struct {
	indexer cache.Indexer
}

// NewGPULister returns a GPULister backed by the given informer.
func NewGPULister(informer cache.SharedIndexInformer) GPULister {
	return &gpuLister{
		indexer: informer.GetIndexer(),
	}
}

// List returns all GPUs in the indexer matching the selector.
func (l *gpuLister) List(selector labels.Selector) ([]*devicev1alpha1.GPU, error) {
	var ret []*devicev1alpha1.GPU
	err := cache.ListAll(l.indexer, selector, func(obj interface{}) {
		if gpu, ok := obj.(*devicev1alpha1.GPU); ok {
			ret = append(ret, gpu)
		}
	})
	return ret, err
}

// Get retrieves a GPU by name from the indexer.
func (l *gpuLister) Get(name string) (*devicev1alpha1.GPU, error) {
	obj, exists, err := l.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(schema.GroupResource{
			Group:    devicev1alpha1.GroupName,
			Resource: "gpus",
		}, name)
	}

	gpu, ok := obj.(*devicev1alpha1.GPU)
	if !ok {
		return nil, fmt.Errorf("unexpected type in indexer for %q", name)
	}

	return gpu, nil
}
