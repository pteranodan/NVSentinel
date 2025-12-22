package v1alpha1

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/nvidia/nvsentinel/client-go/nvgrpc"
	"google.golang.org/grpc"
)

type DeviceV1alpha1Interface interface {
	ClientConn() grpc.ClientConnInterface
	GPUsGetter
}

// DeviceV1alpha1Client is used to interact with features provided by the device.nvidia.com group.
type DeviceV1alpha1Client struct {
	conn   grpc.ClientConnInterface
	logger logr.Logger
}

func (c *DeviceV1alpha1Client) GPUs() GPUInterface {
	return newGPUs(c)
}

// NewForConfig creates a new DeviceV1alpha1Client for the given config.
// NewForConfig is equivalent to NewForConfigAndClient(c, clientConn),
// where clientConn was generated with nvgrpc.ClientConnFor(c).
func NewForConfig(c *nvgrpc.Config) (*DeviceV1alpha1Client, error) {
	if c == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	config := *c // Shallow copy to avoid mutation
	conn, err := nvgrpc.ClientConnFor(&config)
	if err != nil {
		return nil, err
	}

	return NewForConfigAndClient(&config, conn)
}

// NewForConfigAndClient creates a new DeviceV1alpha1Client for the given config and gRPC client connection.
// Note the grpc client connection provided takes precedence over the configured transport values.
func NewForConfigAndClient(c *nvgrpc.Config, conn grpc.ClientConnInterface) (*DeviceV1alpha1Client, error) {
	if c == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if conn == nil {
		return nil, fmt.Errorf("gRPC connection cannot be nil")
	}

	return &DeviceV1alpha1Client{
		conn:   conn,
		logger: c.GetLogger().WithName("device.v1alpha1"),
	}, nil
}

// NewForConfigOrDie creates a new DeviceV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *nvgrpc.Config) *DeviceV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}

	return client
}

// New creates a new DeviceV1alpha1Client for the given gRPC client connection.
func New(c grpc.ClientConnInterface) *DeviceV1alpha1Client {
	return &DeviceV1alpha1Client{
		conn:   c,
		logger: logr.Discard(),
	}
}

// ClientConn returns a gRPC client connection that is used to communicate
// with API server by this client implementation.
func (c *DeviceV1alpha1Client) ClientConn() grpc.ClientConnInterface {
	if c == nil {
		return nil
	}

	return c.conn
}
