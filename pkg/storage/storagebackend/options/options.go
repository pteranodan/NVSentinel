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

// Package options contains flags and options for initializing a storage backend.
package options

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/k3s-io/kine/pkg/drivers/generic"
	"github.com/k3s-io/kine/pkg/endpoint"
	nvgrpc "github.com/nvidia/nvsentinel/pkg/grpc/options"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/server/options"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
	cliflag "k8s.io/component-base/cli/flag"
)

const (
	StorageTypeMemory = "memory"

	defaultKineSocketPath = "/var/run/nvidia-device-api/kine.sock"

	defaultDatabasePath = "sqlite:///var/lib/nvidia-device-api/state.db"
	defaultEtcdVersion  = "3.5.13"
)

var storageTypes = sets.NewString(
	apistorage.StorageTypeETCD3,
	StorageTypeMemory,
)

// Options define the flags and validation for a storage backend.
type Options struct {
	StorageBackend                  string
	StorageInitializationTimeout    time.Duration
	StorageReadycheckTimeout        time.Duration
	DatabasePath                    string
	DatabaseMaxOpenConns            int
	DatabaseMaxIdleConns            int
	DatabaseMaxConnLifetime         time.Duration
	EtcdVersion                     string
	EtcdWatchProgressNotifyInterval time.Duration
	EtcdCompactionInterval          time.Duration
	EtcdCompactionIntervalJitter    int
	EtcdCompactionTimeout           time.Duration
	EtcdCompactionMinRetain         int64
	EtcdCompactionBatchSize         int64
	EtcdPollBatchSize               int64

	GRPC *nvgrpc.Options

	kineSocketPath string
}

// completedOptions is a private wrapper that enforces a call of Complete() before Run can be invoked.
type completedOptions struct {
	Options

	DatabaseDir string
	SocketPath  string
	KineConfig  endpoint.Config
	Etcd        *options.EtcdOptions
}

type CompletedOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedOptions
}

// NewOptions creates a new Storage Options object with default parameters.
func NewOptions() *Options {
	socketPath := defaultKineSocketPath
	if env := os.Getenv("KINE_SOCKET_PATH"); env != "" {
		socketPath = env
	}

	return &Options{
		StorageBackend:                  apistorage.StorageTypeETCD3,
		StorageInitializationTimeout:    1 * time.Minute,
		StorageReadycheckTimeout:        2 * time.Second,
		DatabasePath:                    defaultDatabasePath,
		DatabaseMaxOpenConns:            2,
		DatabaseMaxIdleConns:            2, // from database/sql
		DatabaseMaxConnLifetime:         3 * time.Minute,
		GRPC:                            nvgrpc.NewOptions(),
		EtcdVersion:                     defaultEtcdVersion,
		EtcdWatchProgressNotifyInterval: 5 * time.Minute,
		EtcdCompactionInterval:          5 * time.Minute,
		EtcdCompactionIntervalJitter:    10,
		EtcdCompactionTimeout:           5 * time.Second,
		EtcdCompactionMinRetain:         1000,
		EtcdCompactionBatchSize:         1000,
		EtcdPollBatchSize:               500,
		kineSocketPath:                  socketPath,
	}
}

func (o *Options) AddFlags(fss *cliflag.NamedFlagSets) {
	if o == nil {
		return
	}

	storageFs := fss.FlagSet("storage")

	storageFs.StringVar(&o.StorageBackend, "storage-backend", o.StorageBackend,
		fmt.Sprintf("The storage backend for persistence. Options: %s (default), %s", apistorage.StorageTypeETCD3, StorageTypeMemory))
	storageFs.DurationVar(&o.StorageInitializationTimeout, "storage-initialization-timeout", o.StorageInitializationTimeout,
		"The maximum amount of time to wait for storage initialization before declaring the server ready.")
	storageFs.DurationVar(&o.StorageReadycheckTimeout, "storage-readycheck-timeout", o.StorageReadycheckTimeout,
		"The timeout to use when checking storage readiness.")

	storageFs.StringVar(&o.DatabasePath, "database-path", o.DatabasePath, "The path to the SQLite database file.")
	storageFs.IntVar(&o.DatabaseMaxOpenConns, "database-max-open-connections", o.DatabaseMaxOpenConns,
		"The maximum number of open connections to the backend database. Set to <= 0 for unlimited. If set, must be at least 2.")
	storageFs.IntVar(&o.DatabaseMaxIdleConns, "database-max-idle-connections", o.DatabaseMaxIdleConns,
		"The maximum number of idle connections to the backend database. Set to 0 to disable connection pooling.")
	storageFs.DurationVar(&o.DatabaseMaxConnLifetime, "database-connection-max-lifetime", o.DatabaseMaxConnLifetime,
		"The maximum amount of time a database connection may be reused. Set to 0s for unlimited. If enabled, must be at least 1s.")

	storageFs.StringVar(&o.EtcdVersion, "etcd-version", o.EtcdVersion, "The emulated etcd version.")
	storageFs.DurationVar(&o.EtcdWatchProgressNotifyInterval, "etcd-watch-progress-notify-interval", o.EtcdWatchProgressNotifyInterval,
		"Interval between periodic watch progress notifications. Must be between 5s and 10m.")
	storageFs.DurationVar(&o.EtcdCompactionInterval, "etcd-compaction-interval", o.EtcdCompactionInterval,
		"The interval of compaction requests. If 0, compaction is disabled. If enabled, must be at least 1m.")
	storageFs.IntVar(&o.EtcdCompactionIntervalJitter, "etcd-compaction-interval-jitter", o.EtcdCompactionIntervalJitter,
		"The percentage of jitter to apply to compaction interval durations. Must be between 0 and 100.")
	storageFs.DurationVar(&o.EtcdCompactionTimeout, "etcd-compaction-timeout", o.EtcdCompactionTimeout,
		"The timeout to use when compacting.")
	storageFs.Int64Var(&o.EtcdCompactionMinRetain, "etcd-compaction-min-retain", o.EtcdCompactionMinRetain,
		"The minimum number of revisions to retain when compacting. Must be between 1 and 10000.")
	storageFs.Int64Var(&o.EtcdCompactionBatchSize, "etcd-compaction-batch-size", o.EtcdCompactionBatchSize,
		"Number of revisions to compact in a single batch. Must be between 1 and 10000.")
	storageFs.Int64Var(&o.EtcdPollBatchSize, "etcd-poll-batch-size", o.EtcdPollBatchSize,
		"Number of revisions to poll in a single batch.")
}

func (o *Options) Complete() (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{completedOptions: &completedOptions{}}, nil
	}

	completed := completedOptions{
		Options: *o,
	}

	etcd := options.NewEtcdOptions(apistorage.NewDefaultConfig("/registry", nil))
	etcd.StorageConfig.HealthcheckTimeout = o.StorageInitializationTimeout
	etcd.StorageConfig.ReadycheckTimeout = o.StorageReadycheckTimeout

	if o.StorageBackend == StorageTypeMemory {
		etcd.StorageConfig.Type = StorageTypeMemory
		completed.Etcd = etcd

		completed.KineConfig = endpoint.Config{}
		completed.DatabaseDir = ""

		return CompletedOptions{completedOptions: &completed}, nil
	}

	etcd.StorageConfig.Type = apistorage.StorageTypeETCD3

	databaseEndpoint := o.DatabasePath
	if databaseEndpoint == "" || databaseEndpoint == defaultDatabasePath {
		v := url.Values{}
		v.Set("_busy_timeout", "5000")
		v.Set("_cache_size", "-65536") // 64MiB
		v.Set("_journal_mode", "WAL")
		v.Set("_locking_mode", "NORMAL")
		v.Set("_mmap_size", "268435456") // 256MiB
		v.Set("_page_size", "4096")      // 4KiB
		v.Set("_synchronous", "NORMAL")
		v.Set("_temp_store", "MEMORY")
		v.Set("_txlock", "immediate")
		databaseEndpoint = fmt.Sprintf("%s?%s", defaultDatabasePath, v.Encode())
	}

	path := databaseEndpoint
	if strings.Contains(path, "://") {
		if u, err := url.Parse(databaseEndpoint); err == nil && u.Path != "" {
			path = u.Path
		}
	}
	completed.DatabaseDir = filepath.Dir(path)

	maxIdle := o.DatabaseMaxIdleConns
	if maxIdle == 0 {
		// In database/sql, MaxIdleConns 0 defaults to 2; set to negative to disable connection pooling.
		maxIdle = -1
	}

	completed.SocketPath = o.kineSocketPath
	kineListener := "unix://" + o.kineSocketPath

	kineConfig := endpoint.Config{
		Listener: kineListener,
		Endpoint: databaseEndpoint,
		ConnectionPoolConfig: generic.ConnectionPoolConfig{
			MaxIdle:     maxIdle,
			MaxOpen:     o.DatabaseMaxOpenConns,
			MaxLifetime: o.DatabaseMaxConnLifetime,
		},
		NotifyInterval:        o.EtcdWatchProgressNotifyInterval,
		EmulatedETCDVersion:   o.EtcdVersion,
		CompactInterval:       o.EtcdCompactionInterval,
		CompactIntervalJitter: o.EtcdCompactionIntervalJitter,
		CompactTimeout:        o.EtcdCompactionTimeout,
		CompactMinRetain:      o.EtcdCompactionMinRetain,
		CompactBatchSize:      o.EtcdCompactionBatchSize,
		PollBatchSize:         o.EtcdPollBatchSize,
	}
	etcd.StorageConfig.Transport.ServerList = []string{kineConfig.Listener}

	completedGRPC, err := o.GRPC.Complete()
	if err != nil {
		return CompletedOptions{}, fmt.Errorf("failed to complete grpc options: %w", err)
	}

	var serverOpts []grpc.ServerOption
	if err := completedGRPC.ApplyTo(&serverOpts); err != nil {
		return CompletedOptions{}, fmt.Errorf("failed to apply grpc options: %w", err)
	}
	kineConfig.GRPCServer = grpc.NewServer(serverOpts...)

	completed.KineConfig = kineConfig
	completed.Etcd = etcd

	return CompletedOptions{
		completedOptions: &completed,
	}, nil
}

func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	allErrors := []error{}

	if !storageTypes.Has(o.StorageBackend) {
		allErrors = append(allErrors, fmt.Errorf("--storage-type %v: invalid, allowed values: %s.", o.StorageBackend, strings.Join(storageTypes.List(), ", ")))
	}

	if o.StorageInitializationTimeout < 1*time.Second {
		allErrors = append(allErrors, fmt.Errorf("--storage-initialization-timeout %v: must be at least 1s", o.StorageInitializationTimeout))
	}

	if o.StorageReadycheckTimeout < 1*time.Second {
		allErrors = append(allErrors, fmt.Errorf("--storage-readycheck-timeout %v: must be at least 1s", o.StorageReadycheckTimeout))
	}

	if o.StorageReadycheckTimeout > o.StorageInitializationTimeout {
		allErrors = append(allErrors, fmt.Errorf("--storage-readycheck-timeout: %v must be less than or equal to --storage-initialization-timeout %v", o.StorageReadycheckTimeout, o.StorageInitializationTimeout))
	}

	// Exit early for StorageTypeMemory (database options don't apply)
	if o.StorageBackend == StorageTypeMemory {
		return allErrors
	}

	if o.DatabasePath == "" {
		allErrors = append(allErrors, fmt.Errorf("--database-path: required"))
	} else {
		path := o.DatabasePath
		if strings.Contains(path, "://") {
			if u, err := url.Parse(path); err == nil && u.Path != "" {
				path = u.Path
			}
		}

		if !filepath.IsAbs(path) {
			allErrors = append(allErrors, fmt.Errorf("--database-path %q: file path component must be an absolute path", o.DatabasePath))
		}
	}

	// Kine+SQLite requires at least 2 connections.
	if o.DatabaseMaxOpenConns == 1 {
		allErrors = append(allErrors, fmt.Errorf("--database-max-open-connections %d: must be less than or equal to 0 (unlimited) or greater than 2", o.DatabaseMaxOpenConns))
	}

	if o.DatabaseMaxOpenConns > 0 && o.DatabaseMaxIdleConns > o.DatabaseMaxOpenConns {
		allErrors = append(allErrors, fmt.Errorf("--database-max-idle-connections %d: must be less than or equal to --database-max-open-connections %d", o.DatabaseMaxIdleConns, o.DatabaseMaxOpenConns))
	}

	if o.DatabaseMaxConnLifetime < 0 {
		allErrors = append(allErrors, fmt.Errorf("--database-connection-max-lifetime %d: must be 0 (unlimited) or a positive duration", o.DatabaseMaxConnLifetime))
	}

	if o.EtcdWatchProgressNotifyInterval < 5*time.Second || o.EtcdWatchProgressNotifyInterval > 10*time.Minute {
		allErrors = append(allErrors, fmt.Errorf("--etcd-watch-progress-notify-interval %v: must be between 5s and 10m", o.EtcdWatchProgressNotifyInterval))
	}

	if o.EtcdCompactionInterval > 0 && o.EtcdCompactionInterval < 1*time.Minute {
		allErrors = append(allErrors, fmt.Errorf("--etcd-compaction-interval %v: must be 0 (disable) or at least 1m", o.EtcdCompactionInterval))
	}

	if o.EtcdCompactionInterval != 0 {
		if o.EtcdCompactionIntervalJitter < 0 || o.EtcdCompactionIntervalJitter > 100 {
			allErrors = append(allErrors, fmt.Errorf("--etcd-compaction-interval-jitter %v: must be between 0 and 100", o.EtcdCompactionIntervalJitter))
		}

		if o.EtcdCompactionInterval != 0 && o.EtcdCompactionTimeout > o.EtcdCompactionInterval {
			allErrors = append(allErrors, fmt.Errorf("--etcd-compaction-timeout %v: must be less than or equal to --etcd-compaction-interval %d", o.EtcdCompactionTimeout, o.EtcdCompactionInterval))
		}

		if o.EtcdCompactionMinRetain <= 100 || o.EtcdCompactionMinRetain > 10000 {
			allErrors = append(allErrors, fmt.Errorf("--etcd-compaction-min-retain %d: must be between 100 and 10000", o.EtcdCompactionMinRetain))
		}

		// Kine minimum compaction batch size is 100
		if o.EtcdCompactionBatchSize <= 100 || o.EtcdCompactionBatchSize > 10000 {
			allErrors = append(allErrors, fmt.Errorf("--etcd-compaction-batch-size %d: must be between 100 and 10000", o.EtcdCompactionBatchSize))
		}
	}

	if o.EtcdPollBatchSize < 100 || o.EtcdPollBatchSize > 10000 {
		allErrors = append(allErrors, fmt.Errorf("--etcd-poll-batch-size %d: must be between 100 and 10000", o.EtcdPollBatchSize))
	}

	if o.GRPC != nil {
		allErrors = append(allErrors, o.GRPC.Validate()...)
	}

	return allErrors
}
