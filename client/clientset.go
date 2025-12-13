package client

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/nvidia/nvsentinel/client/config"
	devicev1alpha1 "github.com/nvidia/nvsentinel/client/typed/device/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type Interface interface {
	DeviceV1alpha1() devicev1alpha1.DeviceV1alpha1Interface
}

type Clientset struct {
	conn           *grpc.ClientConn
	config         *config.Config
	deviceV1alpha1 *devicev1alpha1.DeviceV1alpha1Client
}

func (c *Clientset) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Clientset) DeviceV1alpha1() devicev1alpha1.DeviceV1alpha1Interface {
	return c.deviceV1alpha1
}

func NewForConfig(c *config.Config) (*Clientset, error) {
	configShallowCopy := *c

	if configShallowCopy.Target == "" {
		return nil, fmt.Errorf("target must be provided in config (e.g., '%s')", config.DefaultSocketPath)
	}

	logger := c.Logger
	if (logger == logr.Logger{}) {
		logger = logr.Discard()
	}

	var opts []grpc.DialOption

	if configShallowCopy.UserAgent == "" {
		configShallowCopy.UserAgent = config.DefaultUserAgent
	}
	opts = append(opts, grpc.WithUserAgent(configShallowCopy.UserAgent))

	if configShallowCopy.AuthToken != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(bearerToken(configShallowCopy.AuthToken)))
	}

	if configShallowCopy.KeepAliveTime < 0 {
		return nil, fmt.Errorf("keep alive time cannot be negative")
	}
	if configShallowCopy.KeepAliveTime == 0 {
		configShallowCopy.KeepAliveTime = config.DefaultKeepAliveTime
	}

	if configShallowCopy.KeepAliveTimeout < 0 {
		return nil, fmt.Errorf("keep alive timeout cannot be negative")
	}
	if configShallowCopy.KeepAliveTimeout == 0 {
		configShallowCopy.KeepAliveTimeout = config.DefaultKeepAliveTimeout
	}

	opts = append(opts, grpc.WithKeepaliveParams(
		keepalive.ClientParameters{
			Time:                configShallowCopy.KeepAliveTime,
			Timeout:             configShallowCopy.KeepAliveTimeout,
			PermitWithoutStream: true,
		},
	))

	if configShallowCopy.IdleTimeout < 0 {
		return nil, fmt.Errorf("idle timeout cannot be negative")
	}
	if configShallowCopy.IdleTimeout == 0 {
		configShallowCopy.IdleTimeout = config.DefaultIdleTimeout
	}
	opts = append(opts, grpc.WithIdleTimeout(configShallowCopy.IdleTimeout))

	// v1alpha1 uses Unix Domain Sockets (UDS), which rely on filesystem permissions for security.
	// TLS is not currently supported.
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	opts = append(opts,
		grpc.WithChainUnaryInterceptor(latencyUnaryInterceptor(logger)),
		grpc.WithChainStreamInterceptor(latencyStreamInterceptor(logger)),
	)

	conn, err := grpc.NewClient(configShallowCopy.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial target %s: %w", configShallowCopy.Target, err)
	}

	return NewForConfigAndClient(&configShallowCopy, conn)
}

func NewForConfigAndClient(c *config.Config, conn *grpc.ClientConn) (*Clientset, error) {
	cs := &Clientset{
		conn:   conn,
		config: c,
	}

	var err error
	cs.deviceV1alpha1, err = devicev1alpha1.NewForConfigAndClient(c, conn)
	if err != nil {
		return nil, err
	}

	return cs, nil
}

func NewForConfigOrDie(c *config.Config) *Clientset {
	cs, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}

	return cs
}

// bearerToken implements credentials.PerRPCCredentials to inject a static token.
type bearerToken string

func (t bearerToken) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + string(t),
	}, nil
}

func (t bearerToken) RequireTransportSecurity() bool {
	// Must be false to allow sending tokens over Unix Sockets (which gRPC sees as 'insecure')
	return false
}
