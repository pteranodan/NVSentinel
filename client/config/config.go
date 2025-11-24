package config

import (
	"errors"
	"time"

	"github.com/go-logr/logr"
)

const (
	defaultClientKeepAliveTime    = 30 * time.Second
	defaultClientKeepAliveTimeout = 5 * time.Second
	defaultIdleTimeout            = 5 * time.Minute
)

type Config struct {
	Target           string
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
	IdleTimeout      time.Duration
	Insecure         bool
	Logger           logr.Logger
}

func NewDefaultConfig(target string) (*Config, error) {
	if target == "" {
		return nil, errors.New("target cannot be empty")
	}

	return &Config{
		Target:           target,
		KeepAliveTime:    defaultClientKeepAliveTime,
		KeepAliveTimeout: defaultClientKeepAliveTimeout,
		IdleTimeout:      defaultIdleTimeout,
		Insecure:         false,
	}, nil
}
