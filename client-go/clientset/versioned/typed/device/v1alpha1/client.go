package v1alpha1

import (
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
)

// DeviceV1alpha1Interface describes the interface for the Device v1alpha1 group.
type DeviceV1alpha1Interface interface {
	// GPUs returns a client interface for GPU resources.
	GPUs() GPUInterface
}

// DeviceV1alpha1Client implements DeviceV1alpha1Interface.
type DeviceV1alpha1Client struct {
	conn   *grpc.ClientConn
	logger logr.Logger
}

// GPUs returns a GPU client.
func (c *DeviceV1alpha1Client) GPUs() GPUInterface {
	return newGPUClient(c.conn, c.logger)
}

// NewDeviceV1alpha1Client creates a new v1alpha1 device client.
// If the logger is empty, it defaults to logr.Discard().
func NewDeviceV1alpha1Client(conn *grpc.ClientConn, logger logr.Logger) *DeviceV1alpha1Client {
	if (logger == logr.Logger{}) {
		logger = logr.Discard()
	}

	return &DeviceV1alpha1Client{
		conn:   conn,
		logger: logger,
	}
}
