// Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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

package client

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	nvvalidation "github.com/nvidia/nvsentinel/pkg/util/validation"
	"github.com/nvidia/nvsentinel/pkg/version"
)

const (
	// NvidiaDeviceAPIEnvVar is the environment variable that overrides the gRPC target.
	NvidiaDeviceAPIEnvVar = "NVIDIA_DEVICE_API"

	// DefaultNvidiaDeviceAPISocket is the default Unix domain socket path.
	DefaultNvidiaDeviceAPISocket = "unix:///var/run/nvidia-device-api/device-api.sock"

	// DefaultKeepAliveTime is the default frequency of keepalive pings.
	DefaultKeepAliveTime = 5 * time.Minute

	// DefaultKeepAliveTimeout is the default time to wait for a keepalive pong.
	DefaultKeepAliveTimeout = 20 * time.Second
)

// Config holds configuration for the Device API client.
type Config struct {
	// Target is the address of the gRPC server (e.g. "unix:///path/to/socket").
	Target string

	// UserAgent is the string to use for the gRPC User-Agent header.
	UserAgent string

	logger logr.Logger
}

// Default populates unset fields in the Config with default values.
func (c *Config) Default() {
	if c.Target == "" {
		c.Target = os.Getenv(NvidiaDeviceAPIEnvVar)
	}

	if c.Target == "" {
		c.Target = DefaultNvidiaDeviceAPISocket
	}

	if c.UserAgent == "" {
		c.UserAgent = version.UserAgent()
	}

	if c.logger.GetSink() == nil {
		c.logger = logr.Discard()
	}
}

// Validate checks if the Config is valid and returns an error if not.
func (c *Config) Validate() error {
	if c.Target == "" {
		return fmt.Errorf("target: required")
	} else {
		if validationErrors := nvvalidation.IsUnixSocketURI(c.Target); len(validationErrors) > 0 {
			return fmt.Errorf("target %q: %s", c.Target, strings.Join(validationErrors, ", "))
		}
	}

	if c.UserAgent == "" {
		return fmt.Errorf("user-agent: required")
	}

	return nil
}

// GetLogger returns the configured logger. If no logger is set, it returns a discard logger.
func (c *Config) GetLogger() logr.Logger {
	if c.logger.GetSink() == nil {
		return logr.Discard()
	}

	return c.logger
}
