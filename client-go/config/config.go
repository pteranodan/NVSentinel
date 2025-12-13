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

package config

import (
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/nvidia/nvsentinel/client/internal/transport"
)

const (
	// NvidiaDeviceAPITargetEnvVar is the environment variable for overriding the gRPC target.
	NvidiaDeviceAPITargetEnvVar = "NVIDIA_DEVICE_API_TARGET"

	// DefaultNvidiaDeviceAPISocket is the default Unix domain socket path for the gRPC server.
	DefaultNvidiaDeviceAPISocket = "unix:///var/run/nvidia-device-api/device-api.sock"

	// DefaultNvidiaUserAgent is the default User-Agent for gRPC calls.
	DefaultNvidiaUserAgent = "nvidia-device-api/v1alpha1"

	// DefaultKeepAliveTime is the default interval between gRPC keepalive pings.
	DefaultKeepAliveTime = 30 * time.Second

	// DefaultKeepAliveTimeout is the default timeout for gRPC keepalive pings.
	DefaultKeepAliveTimeout = 20 * time.Second

	// DefaultIdleTimeout is the default maximum idle duration before closing a gRPC connection.
	DefaultIdleTimeout = 4 * time.Hour
)

// Config holds configuration for connecting to the gRPC server.
type Config struct {
	// Target is the gRPC server address (unix socket or host:port).
	Target string

	// UserAgent is the gRPC client User-Agent. Defaults to DefaultNvidiaUserAgent.
	UserAgent string

	// Logger is used for logging RPC latencies and errors. Defaults to logr.Discard().
	Logger logr.Logger

	// TokenSource provides bearer tokens for gRPC requests.
	TokenSource transport.TokenSource

	// KeepAliveTime is the interval between gRPC keepalive pings.
	KeepAliveTime time.Duration

	// KeepAliveTimeout is the timeout for gRPC keepalive pings.
	KeepAliveTimeout time.Duration

	// IdleTimeout is the maximum time a connection can remain idle before closing.
	IdleTimeout time.Duration
}

// NewDefaultConfig returns a Config populated with default values.
//
// If target is empty, it uses the environment variable NVIDIA_DEVICE_API_TARGET
// or the default Unix domain socket path.
//
// Example usage:
//
//	cfg, err := config.NewDefaultConfig("")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	cfg.TokenSource = transport.NewTokenSource(func(ctx context.Context) (string, error) {
//	    return fetchTokenFromVault(ctx)
//	})
//	cfg.Logger = logr.New(logr.StdLogger{})
func NewDefaultConfig(target string) (*Config, error) {
	if target == "" {
		target = os.Getenv(NvidiaDeviceAPITargetEnvVar)
	}
	if target == "" {
		target = DefaultNvidiaDeviceAPISocket
	}

	return &Config{
		Target:           target,
		UserAgent:        DefaultNvidiaUserAgent,
		Logger:           logr.Discard(),
		KeepAliveTime:    DefaultKeepAliveTime,
		KeepAliveTimeout: DefaultKeepAliveTimeout,
		IdleTimeout:      DefaultIdleTimeout,
	}, nil
}
