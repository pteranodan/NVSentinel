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
	"fmt"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestStorageFactory(t *testing.T) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	testCodec := codecs.LegacyCodec(scheme.PrioritizedVersionsAllGroups()...)

	t.Run("PathNormalization", func(t *testing.T) {
		config := storagebackend.Config{Type: "invalid"}
		factory := NewStorageFactory(config)

		paths := []string{"nodes", "/nodes"}
		for _, p := range paths {
			_, err := factory.NewStorage(p, testCodec, nil, nil)
			if err == nil {
				t.Errorf("expected error for path %s, got nil", p)
			}
		}
	})

	t.Run("ConfigImmutability", func(t *testing.T) {
		originalConfig := storagebackend.Config{Type: ""}
		f := NewStorageFactory(originalConfig)

		_, _ = f.NewStorage("test", testCodec, nil, nil)

		if internal, ok := f.(*storageFactory); ok {
			if internal.config.Codec != nil {
				t.Error("Critical Failure: Factory base config was mutated. This will cause race conditions.")
			}
		}
	})

	t.Run("ThreadSafety", func(t *testing.T) {
		f := NewStorageFactory(storagebackend.Config{Type: "invalid"})

		const workers = 20
		var wg sync.WaitGroup
		wg.Add(workers)

		for i := 0; i < workers; i++ {
			go func(id int) {
				defer wg.Done()
				resource := fmt.Sprintf("res-%d", id)
				_, _ = f.NewStorage(resource, testCodec, nil, nil)
			}(i)
		}
		wg.Wait()
	})
}
