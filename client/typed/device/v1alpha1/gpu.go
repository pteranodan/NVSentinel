package v1alpha1

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"google.golang.org/grpc"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

const DefaultRPCTimeout = 30 * time.Second

type GPUInterface interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*devicev1alpha1.GPU, error)
	List(ctx context.Context, opts metav1.ListOptions) (*devicev1alpha1.GPUList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type gpuClient struct {
	client pb.GpuServiceClient
	logger logr.Logger
}

func newGPUClient(conn *grpc.ClientConn, logger logr.Logger) *gpuClient {
	if (logger == logr.Logger{}) {
		logger = logr.Discard()
	}

	return &gpuClient{
		client: pb.NewGpuServiceClient(conn),
		logger: logger,
	}
}

func (c *gpuClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*devicev1alpha1.GPU, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultRPCTimeout)
		defer cancel()
	}

	req := &pb.GetGpuRequest{
		Name: name,
	}

	resp, err := c.client.GetGpu(ctx, req)
	if err != nil {
		return nil, err
	}

	return devicev1alpha1.FromProto(resp.Gpu), nil
}

func (c *gpuClient) List(ctx context.Context, opts metav1.ListOptions) (*devicev1alpha1.GPUList, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultRPCTimeout)
		defer cancel()
	}

	req := &pb.ListGpusRequest{}

	resp, err := c.client.ListGpus(ctx, req)
	if err != nil {
		return nil, err
	}

	return devicev1alpha1.FromProtoList(resp.GpuList), nil
}

func (c *gpuClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	ctx, cancel := context.WithCancel(ctx)

	req := &pb.WatchGpusRequest{
		ResourceVersion: opts.ResourceVersion,
	}

	stream, err := c.client.WatchGpus(ctx, req)
	if err != nil {
		cancel()
		return nil, err
	}

	return newStreamWatcher(stream, cancel, c.logger), nil
}
