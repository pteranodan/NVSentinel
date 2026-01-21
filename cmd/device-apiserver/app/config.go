package app

import (
	"context"

	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app/options"
	controlplane "github.com/nvidia/nvsentinel/controlplane/apiserver"
)

type Config struct {
	Options options.CompletedOptions

	APIs *controlplane.Config
}

type completedConfig struct {
	Options options.CompletedOptions

	APIs controlplane.CompletedConfig
}

type CompletedConfig struct {
	*completedConfig
}

func NewConfig(ctx context.Context, opts options.CompletedOptions) (*Config, error) {
	c := &Config{
		Options: opts,
	}

	controlPlaneConfig, err := controlplane.BuildConfig(ctx, opts.CompletedOptions)
	if err != nil {
		return nil, err
	}

	c.APIs = controlPlaneConfig

	return c, nil
}

func (c *Config) Complete() (CompletedConfig, error) {
	return CompletedConfig{&completedConfig{
		Options: c.Options,

		APIs: c.APIs.Complete(),
	}}, nil
}
