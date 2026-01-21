package apiserver

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	"k8s.io/klog/v2"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/controlplane/apiserver/options"
)

type Config struct {
	NodeName      string
	BindAddress   string
	ServerOptions []grpc.ServerOption
	StorageConfig storagebackend.Config
	LogOptions    *logs.Options
}

type CompletedConfig struct {
	*Config
}

func BuildConfig(ctx context.Context, opts options.CompletedOptions) (*Config, error) {
	config := &Config{
		NodeName: opts.NodeName,
		// TODO(user): add internal-only defaults
		ServerOptions: []grpc.ServerOption{},
	}

	opts.Metrics.Apply()
	// TODO(dhuenecke): add metrics to service account?

	if err := logsapi.ValidateAndApply(opts.Logs, nil); err != nil {
		return nil, fmt.Errorf("failed to apply logging configuration: %w", err)
	}

	if err := opts.GRPC.ApplyTo(&config.BindAddress, &config.ServerOptions); err != nil {
		return nil, fmt.Errorf("failed to apply grpc options: %w", err)
	}

	if err := opts.Storage.ApplyTo(ctx.Done()); err != nil {
		return nil, fmt.Errorf("failed to apply storage options: %w", err)
	}
	config.StorageConfig = opts.Storage.Etcd.StorageConfig

	return config, nil
}

func (c *Config) Complete() CompletedConfig {
	if c == nil {
		return CompletedConfig{}
	}

	// TODO(dhuenecke): add late-stage defaulting

	return CompletedConfig{c}
}

func (c *CompletedConfig) NewStorage(resource string, newFunc, newListFunc func() runtime.Object) (storage.Interface, error) {
	storageConfig := c.StorageConfig

	storageConfig.Codec = codecs.LegacyCodec(devicev1alpha1.SchemeGroupVersion)
	resourceConfig := storagebackend.ConfigForResource{
		Config: storageConfig,
	}

	if !strings.HasPrefix(resource, "/") {
		resource = "/" + resource
	}

	klog.V(4).InfoS("Creating storage backend",
		"node", c.NodeName,
		"resource", resource,
		"backendType", storageConfig.Type,
		"prefix", storageConfig.Prefix,
	)

	storage, _, err := factory.Create(
		resourceConfig,
		newFunc,
		newListFunc,
		resource,
	)

	if err != nil {
		klog.ErrorS(err, "failed to create storage backend", "node", c.NodeName, "resource", resource)
	} else {
		klog.V(3).InfoS("Initialized storage backend", "node", c.NodeName, "resource", resource)
	}

	return storage, err
}
