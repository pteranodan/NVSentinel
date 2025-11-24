package client

import (
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/nvidia/nvsentinel/client/config"
	"github.com/nvidia/nvsentinel/client/typed/device/v1alpha1"
	devicev1alpha1 "github.com/nvidia/nvsentinel/client/typed/device/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"k8s.io/apimachinery/pkg/api/meta"
)

type Interface interface {
	DeviceV1alpha1() devicev1alpha1.DeviceV1alpha1Interface
}

type Clientset struct {
	conn           *grpc.ClientConn
	config         *config.Config
	deviceV1alpha1 *devicev1alpha1.DeviceV1alpha1Client
}

func main() {

	cfg, _ := config.NewDefaultConfig("localhost:500051")

	clientset, err := NewForConfig(cfg)
	if err != nil {
		return err
	}

	gpu, _ := clientset.DeviceV1alpha1().GPUs().Get("gpu-23342342")

	ready, _ := meta.IsStatusConditionTrue(gpu.Status.Conditions, v1alpha1.GPUReady)
	if !ready {
		//update resourceSlice obj
	}
}

func (c *Clientset) DeviceV1alpha1() devicev1alpha1.DeviceV1alpha1Interface {
	return c.deviceV1alpha1
}

func (c *Clientset) Connection() *grpc.ClientConn {
	return c.conn
}

func (c *Clientset) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Clientset) Config() *config.Config {
	return c.config
}

func NewForConfig(c *config.Config) (*Clientset, error) {
	configShallowCopy := *c

	if configShallowCopy.Target == "" {
		return nil, errors.New("target must be provided in config")
	}

	if configShallowCopy.KeepAliveTime <= 0 {
		return nil, fmt.Errorf("keep alive time is required to be greater than 0")
	}

	if configShallowCopy.KeepAliveTimeout <= 0 {
		return nil, fmt.Errorf("keep alive timeout is required to be greater than 0")
	}

	if configShallowCopy.IdleTimeout <= 0 {
		return nil, fmt.Errorf("idle timeout is required to be greater than 0")
	}

	var creds credentials.TransportCredentials
	if c.Insecure {
		creds = insecure.NewCredentials()
	} else {
		creds = credentials.NewTLS(nil)
	}

	logger := c.Logger
	if (logger == logr.Logger{}) {
		logger = logr.Discard()
	}

	dialOpts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(latencyUnaryInterceptor(logger)),
		grpc.WithChainStreamInterceptor(latencyStreamInterceptor(logger)),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                configShallowCopy.KeepAliveTime,
			Timeout:             configShallowCopy.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),
		grpc.WithIdleTimeout(configShallowCopy.IdleTimeout),
		grpc.WithTransportCredentials(creds),
	}

	conn, err := grpc.NewClient(configShallowCopy.Target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial target %s: %w", configShallowCopy.Target, err)
	}

	return NewForConfigAndClient(&configShallowCopy, conn)
}

func NewForConfigAndClient(c *config.Config, conn *grpc.ClientConn) (*Clientset, error) {
	configShallowCopy := *c

	var cs Clientset
	var err error

	cs.config = &configShallowCopy
	cs.conn = conn

	cs.deviceV1alpha1, err = devicev1alpha1.NewForConfigAndClient(&configShallowCopy, conn)
	if err != nil {
		return nil, err
	}

	return &cs, nil
}

func NewForConfigOrDie(c *config.Config) *Clientset {
	cs, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return cs
}
