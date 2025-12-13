package v1alpha1

import (
	"testing"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

func TestGPULister_List(t *testing.T) {
	idx := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	gpu1 := &devicev1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-1", Labels: map[string]string{"env": "dev"}}}
	gpu2 := &devicev1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-2", Labels: map[string]string{"env": "prod"}}}

	idx.Add(gpu1)
	idx.Add(gpu2)

	lister := &gpuLister{indexer: idx}

	all, err := lister.List(labels.Everything())
	if err != nil || len(all) != 2 {
		t.Errorf("expected 2 GPUs, got %d, err=%v", len(all), err)
	}

	devList, err := lister.List(labels.Set{"env": "dev"}.AsSelector())
	if err != nil || len(devList) != 1 || devList[0].Name != "gpu-1" {
		t.Errorf("expected only gpu-1, got %+v, err=%v", devList, err)
	}
}

func TestGPULister_Get(t *testing.T) {
	idx := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	gpu1 := &devicev1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-1"}}
	idx.Add(gpu1)

	lister := &gpuLister{indexer: idx}

	g, err := lister.Get("gpu-1")
	if err != nil || g.Name != "gpu-1" {
		t.Errorf("expected gpu-1, got %+v, err=%v", g, err)
	}

	_, err = lister.Get("gpu-2")
	if !errors.IsNotFound(err) {
		t.Errorf("expected NotFound error, got %v", err)
	}
}

func TestGPULister_Get_WrongType(t *testing.T) {
	idx := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
	idx.Add(&metav1.PartialObjectMetadata{
		ObjectMeta: metav1.ObjectMeta{Name: "dpu-1"},
	})

	lister := &gpuLister{indexer: idx}

	_, err := lister.Get("dpu-1")
	if err == nil || err.Error() != `unexpected type in indexer for "dpu-1"` {
		t.Errorf("expected type error, got %v", err)
	}
}
