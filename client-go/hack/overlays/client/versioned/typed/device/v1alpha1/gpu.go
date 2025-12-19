package v1alpha1

import (
	context "context"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"github.com/nvidia/nvsentinel/client-go/nvgrpc"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	watch "k8s.io/apimachinery/pkg/watch"
)

// GPUsGetter has a method to return a GPUInterface.
// A group's client should implement this interface.
type GPUsGetter interface {
	GPUs() GPUInterface
}

// GPUInterface has methods to work with GPU resources.
type GPUInterface interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*devicev1alpha1.GPU, error)
	List(ctx context.Context, opts metav1.ListOptions) (*devicev1alpha1.GPUList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	GPUExpansion
}

// gpus implements GPUInterface
type gpus struct {
	client pb.GpuServiceClient
	logger logr.Logger
}

// newGPUs returns a GPUs
func newGPUs(c *DeviceV1alpha1Client) *gpus {
	return &gpus{
		client: pb.NewGpuServiceClient(c.ClientConn()),
		logger: c.logger.WithName("gpus"),
	}
}

func (c *gpus) Get(ctx context.Context, name string, opts metav1.GetOptions) (*devicev1alpha1.GPU, error) {
	resp, err := c.client.GetGpu(ctx, &pb.GetGpuRequest{Name: name})
	if err != nil {
		return nil, err
	}

	gpu := devicev1alpha1.FromProto(resp.GetGpu())
	c.logger.V(6).Info("Fetched GPU",
		"name", name,
		"resource-version", gpu.GetResourceVersion(),
	)

	return gpu, nil
}

func (c *gpus) List(ctx context.Context, opts metav1.ListOptions) (*devicev1alpha1.GPUList, error) {
	resp, err := c.client.ListGpus(ctx, &pb.ListGpusRequest{ResourceVersion: opts.ResourceVersion})
	if err != nil {
		return nil, err
	}

	list := devicev1alpha1.FromProtoList(resp.GetGpuList())
	c.logger.V(5).Info("Listed GPUs",
		"count", len(list.Items),
		"resource-version", list.GetResourceVersion(),
	)

	return list, nil
}

func (c *gpus) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	c.logger.V(4).Info("Opening watch stream",
		"resource", "gpus",
		"resource-version", opts.ResourceVersion,
	)

	ctx, cancel := context.WithCancel(ctx)
	stream, err := c.client.WatchGpus(ctx, &pb.WatchGpusRequest{ResourceVersion: opts.ResourceVersion})
	if err != nil {
		cancel()
		return nil, err
	}

	return nvgrpc.NewWatcher(&gpuStreamAdapter{stream: stream}, cancel, c.logger), nil
}
