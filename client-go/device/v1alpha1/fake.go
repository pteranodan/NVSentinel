package v1alpha1

import (
	"context"
	"io"
	"sync"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// --- Fake gRPC client ---

type FakeGpuServiceClient struct {
	getResp   map[string]*pb.Gpu
	getErr    map[string]error
	listResp  *pb.GpuList
	listErr   error
	watchResp []*pb.WatchGpusResponse
	watchErr  error
}

func NewFakeGpuServiceClient() *FakeGpuServiceClient {
	return &FakeGpuServiceClient{}
}

func (f *FakeGpuServiceClient) GetGpu(ctx context.Context, req *pb.GetGpuRequest, _ ...grpc.CallOption) (*pb.GetGpuResponse, error) {
	if Err, ok := f.getErr[req.Name]; ok && Err != nil {
		return nil, Err
	}
	if gpu, ok := f.getResp[req.Name]; ok {
		return &pb.GetGpuResponse{Gpu: gpu}, nil
	}
	return nil, status.Error(codes.NotFound, "not found")
}

func (f *FakeGpuServiceClient) ListGpus(ctx context.Context, req *pb.ListGpusRequest, _ ...grpc.CallOption) (*pb.ListGpusResponse, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return &pb.ListGpusResponse{GpuList: f.listResp}, nil
}

func (f *FakeGpuServiceClient) WatchGpus(ctx context.Context, req *pb.WatchGpusRequest, _ ...grpc.CallOption) (pb.GpuService_WatchGpusClient, error) {
	if f.watchErr != nil {
		return nil, f.watchErr
	}
	ch := make(chan *pb.WatchGpusResponse, len(f.watchResp))
	for _, r := range f.watchResp {
		ch <- r
	}
	close(ch)
	return NewFakeWatchGpusClient(ch, ctx), nil
}

func (f *FakeGpuServiceClient) SetGetResp(getResp map[string]*pb.Gpu) { f.getResp = getResp }
func (f *FakeGpuServiceClient) SetListResp(listResp *pb.GpuList)      { f.listResp = listResp }
func (f *FakeGpuServiceClient) SetWatchResp(watchResp []*pb.WatchGpusResponse) {
	f.watchResp = watchResp
}

// --- Fake GPU client ---

type FakeGPUClient struct {
	mu          sync.Mutex
	gpus        map[string]*devicev1alpha1.GPU
	ListCalled  bool
	WatchCalled bool
}

func NewFakeGPUClient() *FakeGPUClient {
	return &FakeGPUClient{
		gpus: map[string]*devicev1alpha1.GPU{
			"gpu1": {
				TypeMeta: metav1.TypeMeta{
					Kind:       "GPU",
					APIVersion: "device.nvidia.com/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{Name: "gpu-1"},
				Spec:       devicev1alpha1.GPUSpec{UUID: "GPU-1"},
			},
			"gpu2": {
				TypeMeta: metav1.TypeMeta{
					Kind:       "GPU",
					APIVersion: "device.nvidia.com/v1alpha1",
				},
				ObjectMeta: metav1.ObjectMeta{Name: "gpu-2"},
				Spec:       devicev1alpha1.GPUSpec{UUID: "GPU-2"},
			},
		},
	}
}

func (f *FakeGPUClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*devicev1alpha1.GPU, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	g, ok := f.gpus[name]
	if !ok {
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    "device.nvidia.com",
			Resource: "GPU",
		}, name)
	}
	return g, nil
}

func (f *FakeGPUClient) List(ctx context.Context, opts metav1.ListOptions) (*devicev1alpha1.GPUList, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ListCalled = true

	items := make([]devicev1alpha1.GPU, 0, len(f.gpus))
	for _, g := range f.gpus {
		items = append(items, *g)
	}
	return &devicev1alpha1.GPUList{Items: items}, nil
}

// -- Fake Watch ---

type FakeWatch struct {
	ch chan watch.Event
}

func (f *FakeWatch) NewFakeWatch(ch chan watch.Event) *FakeWatch {
	return &FakeWatch{ch: ch}
}

func (f *FakeWatch) Stop() {
	close(f.ch)
}

func (f *FakeWatch) ResultChan() <-chan watch.Event {
	return f.ch
}

func (f *FakeGPUClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.WatchCalled = true

	ch := make(chan watch.Event, len(f.gpus))
	for _, g := range f.gpus {
		ch <- watch.Event{Type: watch.Added, Object: g}
	}
	return &FakeWatch{ch: ch}, nil
}

func (f *FakeGPUClient) Informer() cache.SharedIndexInformer {
	inf := NewGPUInformer(f, GPUInformerOptions{})
	return inf.Informer()
}

func (f *FakeGPUClient) InformerWithOptions(opts GPUInformerOptions) cache.SharedIndexInformer {
	inf := NewGPUInformer(f, opts)
	return inf.InformerWithOptions(opts)
}

// --- Fake Watch client ---

type FakeWatchGpusClient struct {
	recvCh chan *pb.WatchGpusResponse
	errs   []error
	err    error
	ctx    context.Context
}

func NewFakeWatchGpusClient(recvCh chan *pb.WatchGpusResponse, ctx context.Context) *FakeWatchGpusClient {
	if ctx == nil {
		ctx = context.Background()
	}
	return &FakeWatchGpusClient{
		recvCh: recvCh,
		ctx:    ctx,
	}
}

func (f *FakeWatchGpusClient) Recv() (*pb.WatchGpusResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.errs) > 0 {
		e := f.errs[0]
		f.errs = f.errs[1:]
		return nil, e
	}
	resp, ok := <-f.recvCh
	if !ok {
		return nil, io.EOF
	}
	return resp, nil
}

func (f *FakeWatchGpusClient) SetErrs(errs []error) { f.errs = errs }
func (f *FakeWatchGpusClient) Errs() []error        { return f.errs }
func (f *FakeWatchGpusClient) SetErr(err error)     { f.err = err }
func (f *FakeWatchGpusClient) Err() error           { return f.err }

// Necessary to satisfy interface
func (f *FakeWatchGpusClient) CloseSend() error             { close(f.recvCh); return nil }
func (f *FakeWatchGpusClient) Context() context.Context     { return f.ctx }
func (f *FakeWatchGpusClient) Header() (metadata.MD, error) { return nil, nil }
func (f *FakeWatchGpusClient) RecvMsg(m interface{}) error  { return nil }
func (f *FakeWatchGpusClient) SendMsg(m interface{}) error  { return nil }
func (f *FakeWatchGpusClient) Trailer() metadata.MD         { return nil }
