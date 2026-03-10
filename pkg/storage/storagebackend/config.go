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

package storagebackend

import (
	"fmt"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/nvidia/nvsentinel/pkg/storage/storagebackend/options"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

// Config defines configuration for the storage backend.
type Config struct {
	StorageConfig apistorage.Config
	DatabaseDir   string
	SocketPath    string
	KineConfig    endpoint.Config
}

type CompletedConfig struct {
	*Config
}

// NewConfig creates a new Storage Config object with the given options.
func NewConfig(opts options.CompletedOptions) (*Config, error) {
	if opts.Etcd == nil {
		return nil, fmt.Errorf("etcd: required")
	}

	return &Config{
		StorageConfig: opts.Etcd.StorageConfig,
		DatabaseDir:   opts.DatabaseDir,
		SocketPath:    opts.SocketPath,
		KineConfig:    opts.KineConfig,
	}, nil
}

func (c *Config) Complete() (CompletedConfig, error) {
	if c == nil {
		return CompletedConfig{}, nil
	}

	return CompletedConfig{c}, nil
}
