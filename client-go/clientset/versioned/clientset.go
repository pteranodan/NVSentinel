// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientset

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/nvidia/nvsentinel/client-go/config"
	v1alpha1 "github.com/nvidia/nvsentinel/client-go/device/v1alpha1"
	"github.com/nvidia/nvsentinel/client-go/internal/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// Interface provides access to clients for each supported API group.
type Interface interface {
	// Device returns a client for the device API group.
	Device() Device
}

// Device provides access to versioned clients for the device API group.
type Device interface {
	// V1alpha1 returns the v1alpha1 client for device resources.
	V1alpha1() v1alpha1.DeviceV1alpha1Client
}

// Clientset holds the gRPC connection and provides clients for all supported API groups.
type Clientset struct {
	conn           *grpc.ClientConn
	config         *config.Config
	deviceV1alpha1 v1alpha1.DeviceV1alpha1Client
}

// Close closes the underlying gRPC connection.
// Safe to call multiple times.
func (c *Clientset) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// Device returns a Device that provides access to versioned device clients.
//
// Example:
//
//	clientset.Device().V1alpha1().GPUs()
func (c *Clientset) Device() Device {
	return &deviceClient{
		deviceV1alpha1: c.deviceV1alpha1,
	}
}

// deviceClient implements the Device.
type deviceClient struct {
	deviceV1alpha1 v1alpha1.DeviceV1alpha1Client
}

// V1alpha1 returns the v1alpha1 device client.
//
// Example:
//
//	clientset.Device().V1alpha1().GPUs()
func (d *deviceClient) V1alpha1() v1alpha1.DeviceV1alpha1Client {
	return d.deviceV1alpha1
}

// NewForConfig creates a Clientset for the device API using the provided config.
func NewForConfig(c *config.Config) (*Clientset, error) {
	configCopy := *c
	configPtr := &configCopy

	if configPtr.Target == "" {
		return nil, fmt.Errorf("target must be provided in config (e.g., '%s')", config.DefaultNvidiaDeviceAPISocket)
	}

	logger := configPtr.Logger
	if (logger == logr.Logger{}) {
		logger = logr.Discard()
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()), // UDS/TLS not supported
		grpc.WithChainUnaryInterceptor(transport.NewLatencyUnaryInterceptor(logger)),
		grpc.WithChainStreamInterceptor(transport.NewLatencyStreamInterceptor(logger)),
	}

	if configPtr.UserAgent == "" {
		configPtr.UserAgent = config.DefaultNvidiaUserAgent
	}
	opts = append(opts, grpc.WithUserAgent(configPtr.UserAgent))

	if configPtr.TokenSource != nil {
		opts = append(opts, grpc.WithPerRPCCredentials(transport.NewTokenPerRPCCredentials(configPtr.TokenSource)))
	}

	keepAliveTime, err := validateDuration("KeepAliveTime", configPtr.KeepAliveTime, config.DefaultKeepAliveTime)
	if err != nil {
		return nil, err
	}
	configPtr.KeepAliveTime = keepAliveTime

	keepAliveTimeout, err := validateDuration("KeepAliveTimeout", configPtr.KeepAliveTimeout, config.DefaultKeepAliveTimeout)
	if err != nil {
		return nil, err
	}
	configPtr.KeepAliveTimeout = keepAliveTimeout

	opts = append(opts, grpc.WithKeepaliveParams(
		keepalive.ClientParameters{
			Time:                configPtr.KeepAliveTime,
			Timeout:             configPtr.KeepAliveTimeout,
			PermitWithoutStream: true,
		},
	))

	idleTimeout, err := validateDuration("IdleTimeout", configPtr.IdleTimeout, config.DefaultIdleTimeout)
	if err != nil {
		return nil, err
	}
	configPtr.IdleTimeout = idleTimeout
	opts = append(opts, grpc.WithIdleTimeout(configPtr.IdleTimeout))

	conn, err := grpc.NewClient(configPtr.Target, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial target %s: %w", configPtr.Target, err)
	}

	return NewForConfigAndClient(configPtr, conn)
}

// NewForConfigAndClient creates a Clientset using an existing gRPC connection.
//
// Use this when you already maintain a grpc.ClientConn. Otherwise, prefer NewForConfig.
func NewForConfigAndClient(c *config.Config, conn *grpc.ClientConn) (*Clientset, error) {
	cs := &Clientset{
		conn:   conn,
		config: c,
	}

	cs.deviceV1alpha1 = v1alpha1.NewDeviceV1alpha1Client(conn, c.Logger)

	return cs, nil
}

// NewForConfigOrDie creates a new Clientset and panics if creation fails.
func NewForConfigOrDie(c *config.Config) *Clientset {
	cs, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}

	return cs
}

// validateDuration ensures that the duration is non-negative.
// Returns the provided default if zero.
func validateDuration(name string, duration time.Duration, defaultDuration time.Duration) (time.Duration, error) {
	if duration < 0 {
		return 0, fmt.Errorf("%s cannot be negative", name)
	}
	if duration == 0 {
		return defaultDuration, nil
	}

	return duration, nil
}
