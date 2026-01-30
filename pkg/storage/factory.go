//  Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package storage

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
	"k8s.io/klog/v2"
)

type StorageFactory interface {
	NewStorage(resource string, codec runtime.Codec, newFunc, newListFunc func() runtime.Object) (storage.Interface, error)
}

type storageFactory struct {
	config storagebackend.Config
}

func NewStorageFactory(config storagebackend.Config) StorageFactory {
	return &storageFactory{
		config: config,
	}
}

func (f *storageFactory) NewStorage(
	resource string,
	codec runtime.Codec,
	newFunc func() runtime.Object,
	newListFunc func() runtime.Object,
) (storage.Interface, error) {
	storageConfig := f.config
	storageConfig.Codec = codec

	resourceConfig := storagebackend.ConfigForResource{
		Config: storageConfig,
	}

	if !strings.HasPrefix(resource, "/") {
		resource = "/" + resource
	}

	storage, _, err := factory.Create(
		resourceConfig,
		newFunc,
		newListFunc,
		resource,
	)

	if err != nil {
		klog.ErrorS(err, "failed to create storage backend", "resource", resource)
	} else {
		klog.V(3).InfoS("Initialized storage backend", "resource", resource)
	}

	return storage, err
}
