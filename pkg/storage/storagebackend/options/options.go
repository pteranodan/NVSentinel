//  Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package options

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server/options"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

type Options struct {
	DatabasePath                string
	CompactionInterval          time.Duration
	CompactionBatchSize         int64
	WatchProgressNotifyInterval time.Duration

	KineConfig     endpoint.Config
	KineSocketPath string
	DatabaseDir    string
	Etcd           *options.EtcdOptions
}

type completedOptions struct {
	Options
}

type CompletedOptions struct {
	*completedOptions
}

func NewOptions() *Options {
	return &Options{
		DatabasePath:                "/var/lib/nvidia-device-api/state.db",
		CompactionInterval:          5 * time.Minute,
		CompactionBatchSize:         1000,
		WatchProgressNotifyInterval: 5 * time.Second,
		Etcd:                        options.NewEtcdOptions(apistorage.NewDefaultConfig("/registry", nil)),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.StringVar(&o.DatabasePath, "database-path", o.DatabasePath,
		"The path to the SQLite database file.")
	fs.DurationVar(&o.CompactionInterval, "compaction-interval", o.CompactionInterval,
		"The interval of compaction requests. If 0, the compaction request from apiserver is disabled.")
	fs.Int64Var(&o.CompactionBatchSize, "compaction-batch-size", o.CompactionBatchSize,
		"Number of revisions to compact in a single batch.")
	fs.DurationVar(&o.WatchProgressNotifyInterval, "watch-progress-notify-interval", o.WatchProgressNotifyInterval,
		"Interval between periodic watch progress notifications.")
}

func (o *Options) Complete() (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{}, nil
	}

	if o.KineSocketPath == "" {
		o.KineSocketPath = "/var/run/nvidia-device-api/kine.sock"
	}
	if o.KineSocketPath != "" && o.KineConfig.Listener == "" {
		o.KineConfig.Listener = "unix://" + o.KineSocketPath
	}

	if o.DatabasePath != "" {
		o.DatabaseDir = filepath.Dir(o.DatabasePath)
	}

	if o.KineConfig.Endpoint == "" && o.DatabasePath != "" {
		o.KineConfig.Endpoint = fmt.Sprintf("sqlite://%s?_journal=WAL&_timeout=5000&_synchronous=NORMAL&_fk=1", o.DatabasePath)
	}

	o.KineConfig.CompactInterval = o.CompactionInterval
	o.KineConfig.CompactBatchSize = o.CompactionBatchSize
	o.KineConfig.NotifyInterval = o.WatchProgressNotifyInterval

	if len(o.Etcd.StorageConfig.Transport.ServerList) == 0 {
		o.Etcd.StorageConfig.Transport.ServerList = []string{o.KineConfig.Listener}
	}

	completed := completedOptions{
		Options: *o,
	}

	return CompletedOptions{
		completedOptions: &completed,
	}, nil
}

func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	allErrors := []error{}

	if o.DatabasePath == "" {
		allErrors = append(allErrors, fmt.Errorf("database-path: required"))
	} else if !filepath.IsAbs(o.DatabasePath) {
		allErrors = append(allErrors, fmt.Errorf("database-path %q: must be an absolute path", o.DatabasePath))
	}

	if o.DatabaseDir == "" {
		allErrors = append(allErrors, fmt.Errorf("database directory: not initialized"))
	}

	if o.CompactionInterval < 0 {
		allErrors = append(allErrors, fmt.Errorf("compaction-interval: %v must be 0s or greater", o.CompactionInterval))
	}
	if o.CompactionBatchSize <= 0 {
		allErrors = append(allErrors, fmt.Errorf("compaction-batch-size: %v must be greater than 0", o.CompactionBatchSize))
	}

	if o.WatchProgressNotifyInterval < 0 {
		allErrors = append(allErrors, fmt.Errorf("watch-progress-notify-interval: %v must be 0s or greater", o.WatchProgressNotifyInterval))
	}

	if o.Etcd != nil {
		allErrors = append(allErrors, o.Etcd.Validate()...)
	}

	if o.KineSocketPath == "" {
		allErrors = append(allErrors, fmt.Errorf("kine-socket-path: not initialized"))
	} else if !filepath.IsAbs(o.KineSocketPath) {
		allErrors = append(allErrors, fmt.Errorf("kine-socket-path %q: must be an absolute path", o.KineSocketPath))
	}

	if o.KineConfig.Listener == "" {
		allErrors = append(allErrors, fmt.Errorf("kine-listener: not initialized"))
	} else {
		prefix := "unix://"
		if !strings.HasPrefix(o.KineConfig.Listener, prefix) {
			allErrors = append(allErrors, fmt.Errorf("kine-listener %q: must start with %q", o.KineConfig.Listener, prefix))
		}

		actualPath := strings.TrimPrefix(o.KineConfig.Listener, prefix)
		if actualPath != o.KineSocketPath {
			allErrors = append(allErrors, fmt.Errorf("kine-listener path %q: does not match kine-socket-path %q", actualPath, o.KineSocketPath))
		}
	}

	return allErrors
}

func (o *Options) ApplyTo(storageConfig *apistorage.Config) error {
	if o == nil {
		return nil
	}

	*storageConfig = o.Etcd.StorageConfig

	return nil
}
