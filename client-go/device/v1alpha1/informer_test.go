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
	"sync"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"k8s.io/client-go/tools/cache"
)

func TestGPUInformer_LazyInitialization(t *testing.T) {
	client := NewFakeGPUClient()
	inf := NewGPUInformer(client, GPUInformerOptions{})

	g := inf.(*gpuInformer)
	if g.informer != nil {
		t.Fatal("expected informer to be nil before first call")
	}

	idx := g.Informer()
	if idx == nil {
		t.Fatal("expected informer to be non-nil after Informer() call")
	}

	// prevent hanging
	stopCh := make(chan struct{})
	go idx.Run(stopCh)
	cache.WaitForCacheSync(stopCh, idx.HasSynced)
	close(stopCh)

	if !client.ListCalled {
		t.Error("expected List to be called at least once during informer initialization")
	}
	if !client.WatchCalled {
		t.Error("expected Watch to be called at least once during informer initialization")
	}
}

func TestGPUInformer_Caching(t *testing.T) {
	client := NewFakeGPUClient()
	inf := NewGPUInformer(client, GPUInformerOptions{})

	first := inf.Informer()
	second := inf.Informer()
	if first != second {
		t.Error("expected informer to be cached and reused")
	}
}

func TestGPUInformer_InformerWithOptions_Resets(t *testing.T) {
	client := NewFakeGPUClient()
	inf := NewGPUInformer(client, GPUInformerOptions{})

	first := inf.Informer()

	newInf := inf.InformerWithOptions(GPUInformerOptions{
		ResyncPeriod: 5 * time.Minute,
	})
	if newInf == first {
		t.Error("expected InformerWithOptions to return a new informer")
	}

	g := inf.(*gpuInformer)
	if g.resyncPeriod != 5*time.Minute {
		t.Errorf("expected resync period to be updated, got %v", g.resyncPeriod)
	}

	if len(g.indexers) != 0 {
		t.Errorf("expected indexers to be empty by default, got %d", len(g.indexers))
	}
}

func TestGPUInformer_InformerWithOptions_ReplacesCachedInformer(t *testing.T) {
	client := NewFakeGPUClient()
	inf := NewGPUInformer(client, GPUInformerOptions{})
	first := inf.Informer()

	newInf := inf.InformerWithOptions(GPUInformerOptions{ResyncPeriod: 1 * time.Second})
	if newInf == first {
		t.Fatal("InformerWithOptions did not return a new informer")
	}

	subsequent := inf.Informer()
	if subsequent != newInf {
		t.Fatal("Informer() did not return updated informer after InformerWithOptions")
	}
}

func TestGPUInformer_InformerWithOptions_CustomIndexers(t *testing.T) {
	client := NewFakeGPUClient()
	inf := NewGPUInformer(client, GPUInformerOptions{})

	idx := cache.Indexers{"byName": func(obj interface{}) ([]string, error) {
		g := obj.(*devicev1alpha1.GPU)
		return []string{g.Name}, nil
	}}

	newInf := inf.InformerWithOptions(GPUInformerOptions{Indexers: idx})
	g := inf.(*gpuInformer)

	if len(g.indexers) != 1 {
		t.Errorf("expected 1 indexer, got %d", len(g.indexers))
	}
	if newInf == nil {
		t.Error("expected new informer to be non-nil")
	}
}

func TestGPUInformer_ConcurrentInitialization(t *testing.T) {
	client := NewFakeGPUClient()
	inf := NewGPUInformer(client, GPUInformerOptions{})

	var wg sync.WaitGroup
	const goroutines = 10
	results := make([]cache.SharedIndexInformer, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = inf.Informer()
		}(i)
	}

	wg.Wait()

	for i := 1; i < goroutines; i++ {
		if results[i] != results[0] {
			t.Errorf("informer instances differ at index %d", i)
		}
	}
}

func TestGPUInformer_IndexerFunctionality(t *testing.T) {
	client := NewFakeGPUClient()
	idxInf := NewGPUInformer(client, GPUInformerOptions{
		Indexers: cache.Indexers{
			"byName": func(obj interface{}) ([]string, error) {
				g := obj.(*devicev1alpha1.GPU)
				return []string{g.Name}, nil
			},
		},
	}).Informer()

	stopCh := make(chan struct{})
	defer close(stopCh)
	go idxInf.Run(stopCh)
	cache.WaitForCacheSync(stopCh, idxInf.HasSynced)

	index := idxInf.GetIndexer()
	obj, err := index.ByIndex("byName", "gpu-1")
	if err != nil {
		t.Fatalf("unexpected error querying index: %v", err)
	}
	if len(obj) != 1 {
		t.Fatalf("expected 1 object in index, got %d", len(obj))
	}
	gpu := obj[0].(*devicev1alpha1.GPU)
	if gpu.Name != "gpu-1" {
		t.Errorf("expected gpu-1, got %s", gpu.Name)
	}
}

func TestGPUInformer_MultipleInformerIsolation(t *testing.T) {
	client := NewFakeGPUClient()
	firstInf := NewGPUInformer(client, GPUInformerOptions{ResyncPeriod: 1 * time.Second})
	secondInf := NewGPUInformer(client, GPUInformerOptions{ResyncPeriod: 5 * time.Second})

	if firstInf.Informer() == secondInf.Informer() {
		t.Fatal("expected two GPUInformer instances to have separate SharedIndexInformers")
	}
}

func TestGPUInformer_WatchEventPropagation(t *testing.T) {
	client := NewFakeGPUClient()
	idxInf := NewGPUInformer(client, GPUInformerOptions{}).Informer()

	stopCh := make(chan struct{})
	defer close(stopCh)
	go idxInf.Run(stopCh)
	cache.WaitForCacheSync(stopCh, idxInf.HasSynced)

	w := client.WatchCalled
	if !w {
		t.Fatal("watch was not called")
	}

	indexer := idxInf.GetIndexer()
	items := indexer.List()
	if len(items) != 2 {
		t.Fatalf("expected 2 items in cache, got %d", len(items))
	}
}

func TestGPUInformer_StopChannel(t *testing.T) {
	client := NewFakeGPUClient()
	inf := NewGPUInformer(client, GPUInformerOptions{}).Informer()

	stopCh := make(chan struct{})
	done := make(chan struct{})
	go func() {
		inf.Run(stopCh)
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)
	close(stopCh)

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("informer did not stop after closing stopCh")
	}
}

func TestGPUInformerFactory_LazyInitializationAndCaching(t *testing.T) {
	client := NewFakeGPUClient()
	factory := NewGPUInformerFactory(client, 0, nil)

	inf1 := factory.GPUInformer()
	if inf1 == nil {
		t.Fatal("expected GPUInformer to be non-nil")
	}

	inf2 := factory.GPUInformer()
	if inf1 != inf2 {
		t.Error("expected GPUInformer to be cached and reused")
	}
}

func TestGPUInformerFactory_InformerWithOptions_NewInformer(t *testing.T) {
	client := NewFakeGPUClient()
	factory := NewGPUInformerFactory(client, 0, nil)

	first := factory.GPUInformer()
	newInf := factory.GPUInformerWithOptions(GPUInformerOptions{ResyncPeriod: 5 * time.Minute})

	if newInf == first {
		t.Error("expected GPUInformerWithOptions to return a new informer")
	}
}

func TestGPUInformerFactory_StartAndWaitForCacheSync(t *testing.T) {
	client := NewFakeGPUClient()
	factory := NewGPUInformerFactory(client, 0, nil)

	stopCh := make(chan struct{})
	defer close(stopCh)

	factory.Start(stopCh)

	if !factory.WaitForCacheSync(stopCh) {
		t.Fatal("expected WaitForCacheSync to return true after cache sync")
	}
}

func TestGPUInformerFactory_ConcurrentInformerAccess(t *testing.T) {
	client := NewFakeGPUClient()
	factory := NewGPUInformerFactory(client, 0, nil)

	var wg sync.WaitGroup
	const goroutines = 10
	results := make([]cache.SharedIndexInformer, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = factory.GPUInformer()
		}(i)
	}

	wg.Wait()

	for i := 1; i < goroutines; i++ {
		if results[i] != results[0] {
			t.Errorf("GPUInformer instances differ at index %d", i)
		}
	}
}
