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
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	nvvalidation "github.com/nvidia/nvsentinel/pkg/util/validation"
	"k8s.io/apiserver/pkg/server/options"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
	cliflag "k8s.io/component-base/cli/flag"
)

const IN_MEMORY = "file::memory:"

const (
	defaultEtcdVersion                     = "3.5.13"
	defaultEtcdCompactionInterval          = 5 * time.Minute
	defaultEtcdCompactionBatchSize         = 1000
	defaultEtcdWatchProgressNotifyInterval = 5 * time.Second
	defaultDatabaseMaxOpenConns            = 5
	defaultDatabaseMaxIdleConns            = 5
	defaultDatabaseMaxConnLifetime         = 0
)

type Options struct {
	DatabasePath                    string
	DatabaseMaxOpenConns            int
	DatabaseMaxIdleConns            int
	DatabaseMaxConnLifetime         time.Duration
	EtcdVersion                     string
	EtcdCompactionInterval          time.Duration
	EtcdCompactionBatchSize         int64
	EtcdWatchProgressNotifyInterval time.Duration

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
		DatabasePath:                    IN_MEMORY,
		EtcdVersion:                     defaultEtcdVersion,
		EtcdCompactionInterval:          defaultEtcdCompactionInterval,
		EtcdCompactionBatchSize:         defaultEtcdCompactionBatchSize,
		EtcdWatchProgressNotifyInterval: defaultEtcdWatchProgressNotifyInterval,
		DatabaseMaxOpenConns:            defaultDatabaseMaxOpenConns,
		DatabaseMaxIdleConns:            defaultDatabaseMaxIdleConns,
		DatabaseMaxConnLifetime:         defaultDatabaseMaxConnLifetime,
		Etcd:                            options.NewEtcdOptions(apistorage.NewDefaultConfig("/registry", nil)),
	}
}

func (o *Options) AddFlags(fss *cliflag.NamedFlagSets) {
	if o == nil {
		return
	}

	storageFs := fss.FlagSet("storage")

	storageFs.StringVar(&o.DatabasePath, "database-path", o.DatabasePath,
		"The path to the SQLite database file. Defaults to in-memory (\"file::memory:\"). "+
			"If using a file, the path must be absolute.")
	storageFs.IntVar(&o.DatabaseMaxOpenConns, "database-max-open-connections", o.DatabaseMaxOpenConns,
		"The maximum number of open connections to the backend database. Set to 0 or less for unlimited.")
	storageFs.IntVar(&o.DatabaseMaxIdleConns, "database-max-idle-connections", o.DatabaseMaxIdleConns,
		"The maximum number of idle connections to the backend database. Set to 0 to disable connection pooling.")
	storageFs.DurationVar(&o.DatabaseMaxConnLifetime, "database-connection-max-lifetime", o.DatabaseMaxConnLifetime,
		"The maximum amount of time a database connection may be reused. Set to 0s for unlimited. "+
			"If enabled, must be at least 1s.")

	storageFs.StringVar(&o.EtcdVersion, "etcd-version", o.EtcdVersion,
		"The emulated etcd version. Defaults to 3.5.13, to indicate support for watch progress notifications.")
	storageFs.DurationVar(&o.EtcdCompactionInterval, "etcd-compaction-interval", o.EtcdCompactionInterval,
		"The interval of compaction requests. If 0, compaction is disabled. If enabled, must be at least 1m.")
	storageFs.Int64Var(&o.EtcdCompactionBatchSize, "etcd-compaction-batch-size", o.EtcdCompactionBatchSize,
		"Number of revisions to compact in a single batch. Must be between 1 and 10000.")
	storageFs.DurationVar(&o.EtcdWatchProgressNotifyInterval, "etcd-watch-progress-notify-interval", o.EtcdWatchProgressNotifyInterval,
		"Interval between periodic watch progress notifications. Must be between 5s and 10m.")
}

func (o *Options) Complete() (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{}, nil
	}

	if o.KineSocketPath == "" {
		o.KineSocketPath = "/var/run/nvidia-device-api/kine.sock"
	}
	o.KineSocketPath = strings.TrimPrefix(o.KineSocketPath, "unix://")

	if o.KineConfig.Listener == "" {
		o.KineConfig.Listener = "unix://" + o.KineSocketPath
	}

	if o.DatabasePath == "" {
		o.DatabasePath = IN_MEMORY
	}

	if o.DatabasePath == IN_MEMORY {
		o.DatabaseDir = "/tmp"

		v := url.Values{}
		v.Set("cache", "shared")
		v.Set("_journal_mode", "WAL")
		v.Set("_synchronous", "OFF")
		v.Set("_busy_timeout", "5000")
		v.Set("_foreign_keys", "ON")
		v.Set("_temp_store", "MEMORY")
		v.Set("_cache_size", "-4096")
		v.Set("_page_size", "16384")
		v.Set("_txlock", "immediate")

		o.KineConfig.Endpoint = fmt.Sprintf("sqlite://%s?%s", IN_MEMORY, v.Encode())
	}

	if o.DatabaseDir == "" {
		o.DatabaseDir = filepath.Dir(o.DatabasePath)
	}

	if o.KineConfig.Endpoint == "" {
		v := url.Values{}
		v.Set("_journal_mode", "WAL")
		v.Set("_busy_timeout", "5000")
		v.Set("_synchronous", "NORMAL")
		v.Set("_foreign_keys", "ON")
		v.Set("_txlock", "immediate")

		o.KineConfig.Endpoint = fmt.Sprintf("sqlite://%s?%s", o.DatabasePath, v.Encode())
	}

	o.KineConfig.ConnectionPoolConfig.MaxOpen = o.DatabaseMaxOpenConns

	if o.DatabaseMaxIdleConns == 0 {
		// In database/sql, MaxIdleConns 0 defaults to 2; set to negative to disable connection pooling.
		o.KineConfig.ConnectionPoolConfig.MaxIdle = -1
	} else {
		o.KineConfig.ConnectionPoolConfig.MaxIdle = o.DatabaseMaxIdleConns
	}

	o.KineConfig.ConnectionPoolConfig.MaxLifetime = o.DatabaseMaxConnLifetime

	if o.EtcdVersion == "" {
		o.EtcdVersion = defaultEtcdVersion
	}
	o.KineConfig.EmulatedETCDVersion = o.EtcdVersion

	o.KineConfig.CompactInterval = o.EtcdCompactionInterval

	if o.EtcdCompactionBatchSize == 0 {
		o.EtcdCompactionBatchSize = defaultEtcdCompactionBatchSize
	}
	o.KineConfig.CompactBatchSize = o.EtcdCompactionBatchSize

	if o.EtcdWatchProgressNotifyInterval <= 0 {
		o.EtcdWatchProgressNotifyInterval = defaultEtcdWatchProgressNotifyInterval
	}
	o.KineConfig.NotifyInterval = o.EtcdWatchProgressNotifyInterval

	o.Etcd.StorageConfig.HealthcheckTimeout = 10 * time.Second
	o.Etcd.StorageConfig.ReadycheckTimeout = 10 * time.Second

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
		allErrors = append(allErrors, fmt.Errorf("--database-path: required"))
	}
	if o.DatabasePath != IN_MEMORY && !filepath.IsAbs(o.DatabasePath) {
		allErrors = append(allErrors, fmt.Errorf("--database-path %q: must be an absolute path", o.DatabasePath))
	}

	expectedSocketPath := o.KineSocketPath
	if expectedSocketPath == "" {
		expectedSocketPath = "/var/run/nvidia-device-api/kine.sock"
	}

	if !filepath.IsAbs(expectedSocketPath) {
		allErrors = append(allErrors, fmt.Errorf("kine socket path %q: must be an absolute path", expectedSocketPath))
	}

	expectedListener := o.KineConfig.Listener
	if expectedListener == "" {
		expectedListener = "unix://" + expectedSocketPath
	}

	if validationErrors := nvvalidation.IsUnixSocketURI(expectedListener); len(validationErrors) > 0 {
		for _, errDesc := range validationErrors {
			allErrors = append(allErrors, fmt.Errorf("kine listener %q: %s", expectedListener, errDesc))
		}
	}

	if o.DatabaseMaxOpenConns > 10 {
		allErrors = append(allErrors, fmt.Errorf("--database-max-open-connections: %v must be 10 or less", o.DatabaseMaxOpenConns))
	}

	if o.DatabaseMaxIdleConns > o.DatabaseMaxOpenConns && o.DatabaseMaxOpenConns > 0 {
		allErrors = append(allErrors, fmt.Errorf("--database-max-idle-connections (%d) cannot be greater than --database-max-open-connections (%d)",
			o.DatabaseMaxIdleConns, o.DatabaseMaxOpenConns))
	}
	if o.DatabaseMaxIdleConns < 0 {
		allErrors = append(allErrors, fmt.Errorf("--database-max-idle-connections: %v must be 0 or greater", o.DatabaseMaxIdleConns))
	}

	if o.DatabaseMaxConnLifetime < 0 {
		allErrors = append(allErrors, fmt.Errorf("--database-connection-max-lifetime: %v must be 0s or greater", o.DatabaseMaxConnLifetime))
	}
	if o.DatabaseMaxConnLifetime > 0 && o.DatabaseMaxConnLifetime < time.Second {
		allErrors = append(allErrors, fmt.Errorf("--database-connection-max-lifetime: %v must be 0s (unlimited) or at least 1s", o.DatabaseMaxConnLifetime))
	}

	if o.EtcdVersion == "" {
		allErrors = append(allErrors, fmt.Errorf("--etcd-version: required"))
	}

	if o.EtcdCompactionInterval > 0 && o.EtcdCompactionInterval < 1*time.Minute {
		allErrors = append(allErrors, fmt.Errorf("--etcd-compaction-interval: %v must be 1m or greater (or 0 to disable)", o.EtcdCompactionInterval))
	}

	if o.EtcdCompactionBatchSize <= 0 || o.EtcdCompactionBatchSize > 10000 {
		allErrors = append(allErrors, fmt.Errorf("--etcd-compaction-batch-size: %v must be between 1 and 10000", o.EtcdCompactionBatchSize))
	}

	if o.EtcdWatchProgressNotifyInterval < 5*time.Second || o.EtcdWatchProgressNotifyInterval > 10*time.Minute {
		allErrors = append(allErrors, fmt.Errorf("--etcd-watch-progress-notify-interval: %v must be between 5s and 10m", o.EtcdWatchProgressNotifyInterval))
	}

	if o.Etcd != nil {
		allErrors = append(allErrors, o.Etcd.Validate()...)
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
