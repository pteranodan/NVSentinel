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

package options_test

import (
	"strings"
	"testing"
	"time"

	"github.com/nvidia/nvsentinel/pkg/storage/storagebackend/options"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
	cliflag "k8s.io/component-base/cli/flag"
)

func TestAddFlags(t *testing.T) {
	o := options.NewOptions()
	fss := &cliflag.NamedFlagSets{}
	o.AddFlags(fss)

	fs := fss.FlagSet("storage")
	args := []string{
		"--storage-type=etcd3",
		"--storage-initialization-timeout=2m",
		"--storage-readycheck-timeout=5s",
		"--storage-endpoint=/tmp/custom.db",
		"--storage-max-open-connections=8",
		"--storage-max-idle-connections=4",
		"--storage-connection-max-lifetime=1h",
		"--etcd-version=3.6.5",
		"--etcd-watch-progress-notify-interval=30s",
		"--etcd-compaction-interval=2m",
		"--etcd-compaction-interval-jitter=5",
		"--etcd-compaction-timeout=3m",
		"--etcd-compaction-min-retain=500",
		"--etcd-compaction-batch-size=500",
		"--etcd-poll-batch-size=500",
	}

	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	if o.Type != "etcd3" {
		t.Errorf("expected Type 'etcd3', got %s", o.Type)
	}
	if o.InitializationTimeout != 2*time.Minute {
		t.Errorf("expected InitializationTimeout '2m', got %s", o.InitializationTimeout)
	}
	if o.ReadycheckTimeout != 5*time.Second {
		t.Errorf("expected StoraageReadycheckTimeout '5s', got %s", o.ReadycheckTimeout)
	}

	if o.Endpoint != "/tmp/custom.db" {
		t.Errorf("expected Endpoint '/tmp/custom.db', got %s", o.Endpoint)
	}
	if o.MaxOpenConns != 8 {
		t.Errorf("expected MaxOpenConns 7, got %d", o.MaxOpenConns)
	}
	if o.MaxIdleConns != 4 {
		t.Errorf("expected MaxIdleConns 4, got %d", o.MaxIdleConns)
	}
	if o.MaxConnLifetime != time.Hour {
		t.Errorf("expected MaxConnLifetime 1h, got %v", o.MaxConnLifetime)
	}

	if o.EtcdVersion != "3.6.5" {
		t.Errorf("expected EtcdVersion '3.6.5', got %s", o.EtcdVersion)
	}
	if o.EtcdWatchProgressNotifyInterval != 30*time.Second {
		t.Errorf("expected EtcdWatchProgressNotifyInterval '30s', got %s", o.EtcdWatchProgressNotifyInterval)
	}
	if o.EtcdCompactionInterval != 2*time.Minute {
		t.Errorf("expected EtcdCompactionInterval '2m', got %s", o.EtcdCompactionInterval)
	}
	if o.EtcdCompactionIntervalJitter != 5 {
		t.Errorf("expected EtcdCompactionIntervalJitter '5', got %d", o.EtcdCompactionIntervalJitter)
	}
	if o.EtcdCompactionTimeout != 3*time.Minute {
		t.Errorf("expected EtcdCompactionTimeout '3m', got %s", o.EtcdCompactionTimeout)
	}
	if o.EtcdCompactionMinRetain != 500 {
		t.Errorf("expected EtcdCompactionMinRetain '500', got %d", o.EtcdCompactionMinRetain)
	}
	if o.EtcdCompactionBatchSize != 500 {
		t.Errorf("expected EtcdCompactionBatchSize '500', got %d", o.EtcdCompactionBatchSize)
	}
	if o.EtcdPollBatchSize != 500 {
		t.Errorf("expected EtcdPollBatchSize '500', got %d", o.EtcdPollBatchSize)
	}
}

func TestComplete(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		opts := options.NewOptions()

		completed, err := opts.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		if completed.Type != apistorage.StorageTypeETCD3 {
			t.Errorf("expected default storage backend %v, got %v", apistorage.StorageTypeETCD3, completed.Type)
		}
		if completed.SocketPath != "/var/run/nvidia-device-api/kine.sock" {
			t.Errorf("expected Kine socket path, got %v", completed.SocketPath)
		}
		if completed.StorageDir != "/var/lib/nvidia-device-api" {
			t.Errorf("expected StorageDir derived from DSN, got %v", completed.StorageDir)
		}

		kineConfig := completed.KineConfig
		if kineConfig.Listener != "unix:///var/run/nvidia-device-api/kine.sock" {
			t.Errorf("expected Kine listener URI prefix, got %v", kineConfig.Listener)
		}
		if !strings.Contains(kineConfig.Endpoint, "sqlite:///var/lib/nvidia-device-api/state.db") {
			t.Errorf("expected Kine Endpoint to contain 'sqlite:///var/lib/nvidia-device-api/state.db', got %v", kineConfig.Endpoint)
		}
		if kineConfig.ConnectionPoolConfig.MaxIdle != 2 {
			t.Errorf("expected Kine conn pool MaxIdle 2, got %d", kineConfig.ConnectionPoolConfig.MaxIdle)
		}
		if kineConfig.ConnectionPoolConfig.MaxOpen != 2 {
			t.Errorf("expected Kine conn pool MaxOpen 5, got %d", kineConfig.ConnectionPoolConfig.MaxOpen)
		}
		if kineConfig.ConnectionPoolConfig.MaxLifetime != 3*time.Minute {
			t.Errorf("expected Kine conn pool MaxLifetime 3m, got %d", kineConfig.ConnectionPoolConfig.MaxLifetime)
		}
		if kineConfig.NotifyInterval != 5*time.Minute {
			t.Errorf("expected Kine NotifyInterval 5m, got %v", kineConfig.NotifyInterval)
		}
		if kineConfig.EmulatedETCDVersion != "3.5.13" {
			t.Errorf("expected Kine EmulatedETCDVersion 3.5.13, got %v", kineConfig.EmulatedETCDVersion)
		}
		if kineConfig.CompactInterval != 5*time.Minute {
			t.Errorf("expected Kine CompactInterval 5m, got %v", kineConfig.CompactInterval)
		}
		if kineConfig.CompactBatchSize != 1000 {
			t.Errorf("expected Kine CompactBatchSize 1000, got %d", kineConfig.CompactBatchSize)
		}

		if kineConfig.GRPCServer == nil {
			t.Error("expected Kine GRPCServer to be non-nil")
		}

		etcdOpts := completed.Etcd.StorageConfig
		if etcdOpts.Type != apistorage.StorageTypeETCD3 {
			t.Errorf("expected etcd storage type 'etcd3', got %s", etcdOpts.Type)
		}
		if etcdOpts.HealthcheckTimeout != 1*time.Minute {
			t.Errorf("expected etcd healthcheck timeout 1m, got %v", etcdOpts.HealthcheckTimeout)
		}
		if etcdOpts.ReadycheckTimeout != 2*time.Second {
			t.Errorf("expected etcd readycheck timeout 2s, got %v", etcdOpts.ReadycheckTimeout)
		}
		if etcdOpts.Transport.ServerList[0] != kineConfig.Listener {
			t.Errorf("expected etcd server to point to Kine listener, got %v", etcdOpts.Transport.ServerList[0])
		}
	})

	t.Run("Custom Endpoint", func(t *testing.T) {
		opts := options.NewOptions()
		opts.Endpoint = "sqlite:///custom/path/data.db?_timeout=5000"

		completed, err := opts.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		expectedDir := "/custom/path"
		if completed.StorageDir != expectedDir {
			t.Errorf("expected StorageDir %q, got %q", expectedDir, completed.StorageDir)
		}
		expectedEndpoint := "sqlite:///custom/path/data.db?_timeout=5000"
		if completed.KineConfig.Endpoint != expectedEndpoint {
			t.Errorf("expected Kine endpoint to match custom Endpoint, got %v", completed.KineConfig.Endpoint)
		}
	})

	t.Run("Disabled Connection Pooling", func(t *testing.T) {
		opts := options.NewOptions()
		opts.MaxIdleConns = 0

		completed, err := opts.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		// In database/sql, MaxIdleConns 0 defaults to 2. To actually disable connection pooling,
		// it should be set to negative
		if completed.KineConfig.ConnectionPoolConfig.MaxIdle != -1 {
			t.Errorf("expected Kine MaxIdle -1 (disabled), got %d", completed.KineConfig.ConnectionPoolConfig.MaxIdle)
		}
	})

	t.Run("InMemory", func(t *testing.T) {
		opts := options.NewOptions()
		opts.Type = options.StorageTypeMemory

		completed, err := opts.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		if completed.Etcd.StorageConfig.Type != options.StorageTypeMemory {
			t.Errorf("expected memory type in etcd config, got %v", completed.Etcd.StorageConfig.Type)
		}
		if completed.KineConfig.Endpoint != "" {
			t.Errorf("expected empty Kine endpoint for memory mode, got %v", completed.KineConfig.Endpoint)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*options.Options)
		wantErr     bool
		errContains string
	}{
		{
			name:    "Valid defaults",
			modify:  func(o *options.Options) {},
			wantErr: false,
		},
		{
			name: "In-memory storage type ignores SQLite paths",
			modify: func(o *options.Options) {
				o.Type = options.StorageTypeMemory
				o.Endpoint = "" // should not trigger "required" error
			},
			wantErr: false,
		},
		{
			name: "Invalid storage backend type",
			modify: func(o *options.Options) {
				o.Type = "postgres"
			},
			wantErr:     true,
			errContains: "invalid, allowed values",
		},
		{
			name: "Storage endpoint required for etcd-shim backends",
			modify: func(o *options.Options) {
				o.Type = apistorage.StorageTypeETCD3
				o.Endpoint = ""
			},
			wantErr:     true,
			errContains: "--storage-endpoint: required",
		},
		{
			name: "Storage endpoint valid with query params",
			modify: func(o *options.Options) {
				o.Endpoint = "sqlite:///path/to/db.state?cache=shared&mode=ro"
			},
			wantErr: false,
		},
		{
			name: "Storage endpoint contains host",
			modify: func(o *options.Options) {
				o.Endpoint = "sqlite://somehost/path/to/db.state"
			},
			wantErr:     true,
			errContains: "host \"somehost\" must be empty",
		},
		{
			name: "Storage endpoint missing scheme",
			modify: func(o *options.Options) {
				o.Endpoint = "/path/to/db.state"
			},
			wantErr:     true,
			errContains: "must start with \"sqlite://\"",
		},
		{
			name: "Storage endpoint invalid scheme",
			modify: func(o *options.Options) {
				o.Endpoint = "sqlit:///path/to/db.state"
			},
			wantErr:     true,
			errContains: "must start with \"sqlite://\"",
		},
		{
			name: "Storage endpoint relative path",
			modify: func(o *options.Options) {
				o.Endpoint = "sqlite://relative/path.db"
			},
			wantErr:     true,
			errContains: "host \"relative\" must be empty",
		},
		{
			name: "Storage initialization timeout too low",
			modify: func(o *options.Options) {
				o.InitializationTimeout = 500 * time.Millisecond
			},
			wantErr:     true,
			errContains: "must be at least 1s",
		},
		{
			name: "Readycheck timeout exceeds initialization timeout",
			modify: func(o *options.Options) {
				o.InitializationTimeout = 5 * time.Second
				o.ReadycheckTimeout = 10 * time.Second
			},
			wantErr:     true,
			errContains: "must be less than or equal to",
		},
		{
			name: "Storage max connections invalid",
			modify: func(o *options.Options) {
				o.MaxOpenConns = 1
			},
			wantErr:     true,
			errContains: "must be less than or equal to 0 (unlimited) or greater than 2",
		},
		{
			name: "Storage connection max lifetime negative",
			modify: func(o *options.Options) {
				o.MaxConnLifetime = -1 * time.Second
			},
			wantErr:     true,
			errContains: "must be 0 (unlimited) or a positive duration",
		},
		{
			name: "Idle connections cannot exceed Max open connections",
			modify: func(o *options.Options) {
				o.MaxOpenConns = 5
				o.MaxIdleConns = 10
			},
			wantErr:     true,
			errContains: "must be less than or equal to",
		},
		{
			name: "Compaction interval disabled",
			modify: func(o *options.Options) {
				o.EtcdCompactionInterval = 0
			},
			wantErr: false,
		},
		{
			name: "Compaction interval too low",
			modify: func(o *options.Options) {
				o.EtcdCompactionInterval = 30 * time.Second
			},
			wantErr:     true,
			errContains: "must be 0 (disable) or at least",
		},
		{
			name: "Compaction interval jitter negative",
			modify: func(o *options.Options) {
				o.EtcdCompactionInterval = 5 * time.Minute
				o.EtcdCompactionIntervalJitter = -10
			},
			wantErr:     true,
			errContains: "must be between 0 and 100",
		},
		{
			name: "Compaction interval jitter out of range",
			modify: func(o *options.Options) {
				o.EtcdCompactionInterval = 5 * time.Minute
				o.EtcdCompactionIntervalJitter = 150
			},
			wantErr:     true,
			errContains: "must be between 0 and 100",
		},
		{
			name: "Compaction min retain too small",
			modify: func(o *options.Options) {
				o.EtcdCompactionInterval = 5 * time.Minute
				o.EtcdCompactionMinRetain = 50
			},
			wantErr:     true,
			errContains: "must be between",
		},
		{
			name: "Compaction min retain out of range",
			modify: func(o *options.Options) {
				o.EtcdCompactionInterval = 5 * time.Minute
				o.EtcdCompactionMinRetain = 50000
			},
			wantErr:     true,
			errContains: "must be between",
		},
		{
			name: "Compaction batch size too small",
			modify: func(o *options.Options) {
				o.EtcdCompactionInterval = 5 * time.Minute
				o.EtcdCompactionBatchSize = 10
			},
			wantErr:     true,
			errContains: "must be between",
		},
		{
			name: "Compaction batch size out of range",
			modify: func(o *options.Options) {
				o.EtcdCompactionBatchSize = 50000
			},
			wantErr:     true,
			errContains: "must be between",
		},
		{
			name: "Compaction timeout exceeds interval",
			modify: func(o *options.Options) {
				o.EtcdCompactionInterval = 5 * time.Minute
				o.EtcdCompactionTimeout = 10 * time.Minute
			},
			wantErr:     true,
			errContains: "must be less than or equal to --etcd-compaction-interval",
		},
		{
			name: "Watch notify interval too low",
			modify: func(o *options.Options) {
				o.EtcdWatchProgressNotifyInterval = 1 * time.Second
			},
			wantErr:     true,
			errContains: "must be between",
		},
		{
			name: "Watch notify interval too high",
			modify: func(o *options.Options) {
				o.EtcdWatchProgressNotifyInterval = 20 * time.Minute
			},
			wantErr:     true,
			errContains: "must be between",
		},
		{
			name: "Poll batch size too small",
			modify: func(o *options.Options) {
				o.EtcdPollBatchSize = 20
			},
			wantErr:     true,
			errContains: "must be between",
		},
		{
			name: "Poll batch size out of range",
			modify: func(o *options.Options) {
				o.EtcdPollBatchSize = 20000
			},
			wantErr:     true,
			errContains: "must be between",
		},
		{
			name: "gRPC validation failure",
			modify: func(o *options.Options) {
				o.GRPC.KeepAliveTime = 2 * time.Second
			},
			wantErr:     true,
			errContains: "--grpc-keepalive-time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := options.NewOptions()
			tt.modify(opts)

			errs := opts.Validate()

			if (len(errs) > 0) != tt.wantErr {
				t.Errorf("Validate() errors = %v, wantErr %v", errs, tt.wantErr)
			}

			if tt.wantErr && len(errs) > 0 {
				found := false
				for _, e := range errs {
					if strings.Contains(e.Error(), tt.errContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("None of the errors %v contain %q", errs, tt.errContains)
				}
			}
		})
	}
}
