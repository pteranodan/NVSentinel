package v1alpha1

import (
	"context"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	eventv1alpha1 "github.com/nvidia/nvsentinel/client/typed/events/v1alpha1"

	"google.golang.org/grpc"
)

const defaultRPCTimeout = 10 * time.Second

type GPUEvent = eventv1alpha1.TypedEvent[*devicev1alpha1.GPU]

type GPUEventStream interface {
	Recv() (*GPUEvent, error)
	CloseSend() error
}

type GPUInterface interface {
	Get(ctx context.Context, name string) (*devicev1alpha1.GPU, error)
	List(ctx context.Context) (*devicev1alpha1.GPUList, error)
	Watch(ctx context.Context, resourceVersion string) (GPUEventStream, error)
}

type gpuClient struct {
	grpcClient pb.GpuServiceClient
}

func newGPUClient(conn *grpc.ClientConn) *gpuClient {
	return &gpuClient{
		grpcClient: pb.NewGpuServiceClient(conn),
	}
}

func (c *gpuClient) Get(ctx context.Context, name string) (*devicev1alpha1.GPU, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultRPCTimeout)
		defer cancel()
	}

	req := &pb.GetGpuRequest{
		Name: name,
	}

	resp, err := c.grpcClient.GetGpu(ctx, req)
	if err != nil {
		return nil, err
	}

	return devicev1alpha1.GPUFromProto(resp.Gpu), nil
}

func (c *gpuClient) List(ctx context.Context) (*devicev1alpha1.GPUList, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultRPCTimeout)
		defer cancel()
	}

	req := &pb.ListGpusRequest{}

	resp, err := c.grpcClient.ListGpus(ctx, req)
	if err != nil {
		return nil, err
	}

	return devicev1alpha1.GPUListFromProto(resp.GpuList), nil
}

func (c *gpuClient) Watch(ctx context.Context, resourceVersion string) (GPUEventStream, error) {
	req := &pb.WatchGpusRequest{
		ResourceVersion: resourceVersion,
	}

	stream, err := c.grpcClient.WatchGpus(ctx, req)
	if err != nil {
		return nil, err
	}

	return newGPUStreamAdapter(stream), nil
}

type gpuStreamAdapter struct {
	stream pb.GpuService_WatchGpusClient
}

func newGPUStreamAdapter(stream pb.GpuService_WatchGpusClient) *gpuStreamAdapter {
	return &gpuStreamAdapter{stream: stream}
}

func (a *gpuStreamAdapter) Recv() (*GPUEvent, error) {
	pbEvent, err := a.stream.Recv()
	if err != nil {
		return nil, err
	}

	eventType := eventv1alpha1.CleanEventType(pbEvent.Type)
	obj := devicev1alpha1.GPUFromProto(pbEvent.Object)

	return &GPUEvent{
		Type:   eventType,
		Object: obj,
	}, nil
}

func (a *gpuStreamAdapter) CloseSend() error {
	return a.stream.CloseSend()
}
