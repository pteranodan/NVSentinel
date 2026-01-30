package storagebackend

import (
	"context"
	"fmt"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/nvidia/nvsentinel/pkg/storage/storagebackend/options"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

type Config struct {
	KineConfig     endpoint.Config
	KineSocketPath string
	DatabaseDir    string

	StorageConfig apistorage.Config
}

type CompletedConfig struct {
	*Config
}

func NewConfig(ctx context.Context, opts options.CompletedOptions) (*Config, error) {
	config := &Config{
		KineConfig:     opts.KineConfig,
		KineSocketPath: opts.KineSocketPath,
		DatabaseDir:    opts.DatabaseDir,
	}

	if err := opts.ApplyTo(&config.StorageConfig); err != nil {
		return nil, fmt.Errorf("failed to apply storage options: %w", err)
	}

	return config, nil
}

func (c *Config) Complete() (CompletedConfig, error) {
	if c == nil {
		return CompletedConfig{}, nil
	}

	return CompletedConfig{c}, nil
}
