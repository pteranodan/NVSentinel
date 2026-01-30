package apiserver

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/api"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/metrics"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/options"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/registry"
	"github.com/nvidia/nvsentinel/pkg/storage"
	"github.com/prometheus/client_golang/prometheus"
)

type Config struct {
	NodeName            string
	BindAddress         string
	HealthAddress       string
	MetricsAddress      string
	ShutdownGracePeriod time.Duration

	APIGroups             []*api.APIGroupInfo
	ServerOptions         []grpc.ServerOption
	ServerMetrics         *grpcprom.ServerMetrics
	ServerMetricsRegistry *prometheus.Registry
	StorageConfig         storagebackend.Config
	StorageFactory        storage.StorageFactory
	LogOptions            *logs.Options
}

type CompletedConfig struct {
	*Config
}

func BuildConfig(ctx context.Context, opts options.CompletedOptions) (*Config, error) {
	metrics.DefaultServerMetrics.Register()
	serverMetrics := metrics.DefaultServerMetrics.Collectors
	serverRegistry := metrics.DefaultServerMetrics.Registry

	config := &Config{
		NodeName:              opts.NodeName,
		HealthAddress:         opts.HealthAddress,
		MetricsAddress:        opts.MetricsAddress,
		ShutdownGracePeriod:   opts.ShutdownGracePeriod,
		ServerOptions:         []grpc.ServerOption{},
		ServerMetrics:         serverMetrics,
		ServerMetricsRegistry: serverRegistry,
		LogOptions:            opts.Logs,
	}

	for _, p := range registry.All() {
		config.APIGroups = append(config.APIGroups, p.BuildGroupInfo())
	}

	config.ServerOptions = append(config.ServerOptions,
		grpc.ChainUnaryInterceptor(serverMetrics.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(serverMetrics.StreamServerInterceptor()),
	)

	if err := logsapi.ValidateAndApply(opts.Logs, nil); err != nil {
		return nil, fmt.Errorf("failed to apply logging configuration: %w", err)
	}

	if err := opts.GRPC.ApplyTo(&config.BindAddress, &config.ServerOptions); err != nil {
		return nil, fmt.Errorf("failed to apply grpc options: %w", err)
	}

	if err := opts.Storage.ApplyTo(&config.StorageConfig); err != nil {
		return nil, fmt.Errorf("failed to apply storage options: %w", err)
	}
	config.StorageFactory = storage.NewStorageFactory(config.StorageConfig)

	return config, nil
}

func (c *Config) Complete() (CompletedConfig, error) {
	if c == nil {
		return CompletedConfig{}, nil
	}

	return CompletedConfig{c}, nil
}
