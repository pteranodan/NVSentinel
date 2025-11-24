package v1alpha1

import (
	"google.golang.org/grpc"

	"github.com/nvidia/nvsentinel/client/config"
)

type DeviceV1alpha1Interface interface {
	Connection() *grpc.ClientConn
	GPUs() GPUInterface
}

type DeviceV1alpha1Client struct {
	conn *grpc.ClientConn
}

func (c *DeviceV1alpha1Client) Connection() *grpc.ClientConn {
	return c.conn
}

func (c *DeviceV1alpha1Client) GPUs() GPUInterface {
	return newGPUClient(c.conn)
}

func NewForConfigAndClient(_ *config.Config, conn *grpc.ClientConn) (*DeviceV1alpha1Client, error) {
	return &DeviceV1alpha1Client{conn: conn}, nil
}
