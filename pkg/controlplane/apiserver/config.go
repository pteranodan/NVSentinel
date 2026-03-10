//  Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package apiserver

import (
	"fmt"
	"strings"
	"time"

	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/api"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/metrics"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/options"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/registry"
	"github.com/nvidia/nvsentinel/pkg/version"
	"google.golang.org/grpc"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
)

// Config defines the configuration for the server.
type Config struct {
	NodeName             string
	BindAddress          string
	HealthAddress        string
	ServiceMonitorPeriod time.Duration
	MetricsAddress       string
	ShutdownGracePeriod  time.Duration
	PprofAddress         string

	ServerOptions    []grpc.ServerOption
	ServerMetrics    *metrics.ServerMetrics
	ServiceProviders []api.ServiceProvider
	Storage          storagebackend.Config
	LogOptions       *logs.Options
}

type CompletedConfig struct {
	*Config
}

// NewConfig creates a new Server Config object with the given options.
func NewConfig(opts options.CompletedOptions) (*Config, error) {
	if opts.Storage.Etcd == nil {
		return nil, fmt.Errorf("storage.etcd: required")
	}

	serverMetrics := metrics.DefaultServerMetrics.WithBuildInfo(version.Get())
	serverMetrics.Register()

	pprofAddr := ""
	if opts.EnablePprof {
		pprofAddr = opts.PprofAddress
	}

	config := &Config{
		NodeName:             opts.NodeName,
		BindAddress:          opts.BindAddress,
		HealthAddress:        opts.HealthAddress,
		ServiceMonitorPeriod: opts.ServiceMonitorPeriod,
		MetricsAddress:       opts.MetricsAddress,
		ShutdownGracePeriod:  opts.ShutdownGracePeriod,
		PprofAddress:         pprofAddr,
		ServerOptions:        opts.Server,
		ServerMetrics:        serverMetrics,
		Storage:              opts.Storage.Etcd.StorageConfig,
		LogOptions:           opts.Logs,
	}

	config.ServiceProviders = append(config.ServiceProviders, registry.List()...)
	if len(config.ServiceProviders) == 0 {
		return nil, fmt.Errorf("service discovery: no providers registered")
	}

	config.ServerOptions = append(config.ServerOptions,
		grpc.ChainUnaryInterceptor(serverMetrics.Collectors.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(serverMetrics.Collectors.StreamServerInterceptor()),
	)

	if err := logsapi.ValidateAndApply(opts.Logs, nil); err != nil {
		if !strings.Contains(err.Error(), "already applied") {
			return nil, fmt.Errorf("failed to apply logging configuration: %w", err)
		}
	}

	return config, nil
}

func (c *Config) Complete() (CompletedConfig, error) {
	if c == nil {
		return CompletedConfig{}, nil
	}

	return CompletedConfig{c}, nil
}
