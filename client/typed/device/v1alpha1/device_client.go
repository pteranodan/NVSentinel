package v1alpha1

import (
	"github.com/nvidia/nvsentinel/client/config"
	"google.golang.org/grpc"
)

type DeviceV1alpha1Interface interface {
	GPUs() GPUInterface
}

type DeviceV1alpha1Client struct {
	conn   *grpc.ClientConn
	config *config.Config
}

func (c *DeviceV1alpha1Client) GPUs() GPUInterface {
	return newGPUClient(c.conn, c.config.Logger)
}

func NewForConfigAndClient(c *config.Config, conn *grpc.ClientConn) (*DeviceV1alpha1Client, error) {
	return &DeviceV1alpha1Client{
		conn:   conn,
		config: c,
	}, nil
}
