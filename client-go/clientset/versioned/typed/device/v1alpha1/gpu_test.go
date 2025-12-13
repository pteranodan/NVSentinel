package v1alpha1

import (
	"context"
	"testing"
	"time"

	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestGPU_Get_Success(t *testing.T) {
	client := NewFakeGpuServiceClient()
	client.SetGetResp(map[string]*pb.Gpu{"gpu-1": {Name: "gpu-1"}})
	gpu := &gpuClient{client: client}
	ctx := context.Background()

	got, err := gpu.Get(ctx, "gpu-1", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "gpu-1" {
		t.Errorf("got %q, want %q", got.Name, "gpu-1")
	}
}

func TestGPU_Get_NotFound(t *testing.T) {
	client := &FakeGpuServiceClient{}
	gpu := &gpuClient{client: client}
	ctx := context.Background()

	_, err := gpu.Get(ctx, "missing", metav1.GetOptions{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGPU_List_Success(t *testing.T) {
	client := NewFakeGpuServiceClient()
	client.SetListResp(&pb.GpuList{Items: []*pb.Gpu{{Name: "gpu-1"}, {Name: "gpu-2"}}})
	gpu := &gpuClient{client: client}

	list, err := gpu.List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(list.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(list.Items))
	}
}

func TestGPU_Watch_Events(t *testing.T) {
	client := NewFakeGpuServiceClient()
	client.SetWatchResp([]*pb.WatchGpusResponse{
		{Type: "ADDED", Object: &pb.Gpu{Name: "gpu-1"}},
		{Type: "MODIFIED", Object: &pb.Gpu{Name: "gpu-1"}},
		{Type: "DELETED", Object: &pb.Gpu{Name: "gpu-1"}},
	})
	gpu := &gpuClient{client: client}

	w, err := gpu.Watch(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got []watch.Event
	for e := range w.ResultChan() {
		got = append(got, e)
	}

	wantTypes := []watch.EventType{watch.Added, watch.Modified, watch.Deleted}
	if len(got) != len(wantTypes) {
		t.Fatalf("unexpected event count: got=%d, want=%d", len(got), len(wantTypes))
	}
	for i, ev := range got {
		if ev.Type != wantTypes[i] {
			t.Errorf("event %d type mismatch: got=%v, want=%v", i, ev.Type, wantTypes[i])
		}
	}
}

func TestGPU_Informer_LazyInitialization(t *testing.T) {
	client := &gpuClient{client: &FakeGpuServiceClient{}}

	if client.informer != nil {
		t.Fatal("expected informer to be nil initially")
	}

	idx := client.Informer()
	if idx == nil {
		t.Fatal("expected informer to be non-nil after Informer()")
	}
}

func TestGPU_Informer_Caching(t *testing.T) {
	client := &gpuClient{client: &FakeGpuServiceClient{}}
	first := client.Informer()
	second := client.Informer()
	if first != second {
		t.Error("expected informer to be cached and reused")
	}
}

func TestGPU_InformerWithOptions_Caching(t *testing.T) {
	client := &FakeGpuServiceClient{}
	gpu := &gpuClient{client: client}

	first := gpu.InformerWithOptions(GPUInformerOptions{
		ResyncPeriod: 1 * time.Second,
	})
	if first == nil {
		t.Fatal("expected first informer to be non-nil")
	}

	second := gpu.InformerWithOptions(GPUInformerOptions{
		ResyncPeriod: 5 * time.Second,
	})
	if second != first {
		t.Error("expected InformerWithOptions to return cached informer, but got a different instance")
	}

	third := gpu.Informer()
	if third != first {
		t.Error("expected Informer() to return the cached informer")
	}
}
