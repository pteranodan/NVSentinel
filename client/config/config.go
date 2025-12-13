package config

import (
	"os"
	"time"

	"github.com/go-logr/logr"
)

const (
	KubernetesDeviceApiTargetEnvVar = "KUBERNETES_DEVICE_API_TARGET"
	DefaultSocketPath               = "unix:///var/run/k8s-device-api/device-api.sock"
	DefaultUserAgent                = "k8s-device-api/v1alpha1"

	DefaultKeepAliveTime    = 30 * time.Second
	DefaultKeepAliveTimeout = 20 * time.Second
	DefaultIdleTimeout      = 4 * time.Hour
)

type Config struct {
	Target           string
	UserAgent        string
	AuthToken        string
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
	IdleTimeout      time.Duration
	Logger           logr.Logger
}

func NewDefaultConfig(target string) (*Config, error) {
	if target == "" {
		target = os.Getenv(KubernetesDeviceApiTargetEnvVar)
	}
	if target == "" {
		target = DefaultSocketPath
	}

	return &Config{
		Target:           target,
		UserAgent:        DefaultUserAgent,
		KeepAliveTime:    DefaultKeepAliveTime,
		KeepAliveTimeout: DefaultKeepAliveTimeout,
		IdleTimeout:      DefaultIdleTimeout,
		Logger:           logr.Discard(),
	}, nil
}
