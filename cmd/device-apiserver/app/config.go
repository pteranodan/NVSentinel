package app

import (
	"context"

	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app/options"
	controlplane "github.com/nvidia/nvsentinel/pkg/controlplane/apiserver"
	"github.com/nvidia/nvsentinel/pkg/storage/storagebackend"
)

type Config struct {
	Options options.CompletedOptions

	Storage *storagebackend.Config
	APIs    *controlplane.Config
}

type completedConfig struct {
	Options options.CompletedOptions

	Storage storagebackend.CompletedConfig
	APIs    controlplane.CompletedConfig
}

type CompletedConfig struct {
	*completedConfig
}

func NewConfig(ctx context.Context, opts options.CompletedOptions) (*Config, error) {
	c := &Config{
		Options: opts,
	}

	storageConfig, err := storagebackend.NewConfig(ctx, opts.CompletedOptions.Storage)
	if err != nil {
		return nil, err
	}
	c.Storage = storageConfig

	controlPlaneConfig, err := controlplane.BuildConfig(ctx, opts.CompletedOptions)
	if err != nil {
		return nil, err
	}
	c.APIs = controlPlaneConfig

	return c, nil
}

func (c *Config) Complete() (CompletedConfig, error) {
	if c == nil || c.Storage == nil || c.APIs == nil {
		return CompletedConfig{}, nil
	}

	completedStorage, err := c.Storage.Complete()
	if err != nil {
		return CompletedConfig{}, err
	}

	completedAPIs, err := c.APIs.Complete()
	if err != nil {
		return CompletedConfig{}, err
	}

	return CompletedConfig{&completedConfig{
		Options: c.Options,

		Storage: completedStorage,
		APIs:    completedAPIs,
	}}, nil
}
