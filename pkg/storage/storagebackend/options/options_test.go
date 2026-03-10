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

// TODO: update test cases
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
		"--storage-backend=etcd3",
		"--storage-initialization-timeout=2m",
		"--storage-readycheck-timeout=5s",
		"--database-path=/tmp/custom.db",
		"--database-max-open-connections=8",
		"--database-max-idle-connections=4",
		"--database-connection-max-lifetime=1h",
		"--etcd-version=3.6.5",
		"--etcd-compaction-interval=2m",
		"--etcd-compaction-batch-size=5000",
		"--etcd-watch-progress-notify-interval=30s",
	}

	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	if o.StorageBackend != "etcd3" {
		t.Errorf("expected StorageBackend 'etcd3', got %s", o.StorageBackend)
	}
	if o.StorageInitializationTimeout != 2*time.Minute {
		t.Errorf("expected StorageInitializationTimeout '2m', got %s", o.StorageInitializationTimeout)
	}
	if o.StorageReadycheckTimeout != 5*time.Second {
		t.Errorf("expected StoraageReadycheckTimeout '5s', got %s", o.StorageReadycheckTimeout)
	}

	if o.DatabasePath != "/tmp/custom.db" {
		t.Errorf("expected DatabasePath '/tmp/custom.db', got %s", o.DatabasePath)
	}
	if o.DatabaseMaxOpenConns != 8 {
		t.Errorf("expected DatabaseMaxOpenConns 8, got %d", o.DatabaseMaxOpenConns)
	}
	if o.DatabaseMaxIdleConns != 4 {
		t.Errorf("expected DatabaseMaxIdleConns 4, got %d", o.DatabaseMaxIdleConns)
	}
	if o.DatabaseMaxConnLifetime != time.Hour {
		t.Errorf("expected DatabaseMaxConnLifetime 1h, got %v", o.DatabaseMaxConnLifetime)
	}

	if o.EtcdVersion != "3.6.5" {
		t.Errorf("expected EtcdVersion '3.6.5', got %s", o.EtcdVersion)
	}
	if o.EtcdCompactionInterval != 2*time.Minute {
		t.Errorf("expected EtcdCompactionInterval '2m', got %s", o.EtcdCompactionInterval)
	}
	if o.EtcdCompactionBatchSize != 5000 {
		t.Errorf("expected EtcdCompactionBatchSize '5000', got %d", o.EtcdCompactionBatchSize)
	}
	if o.EtcdWatchProgressNotifyInterval != 30*time.Second {
		t.Errorf("expected EtcdWatchProgressNotifyInterval '30s', got %s", o.EtcdWatchProgressNotifyInterval)
	}
}

func TestComplete(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		opts := options.NewOptions()

		completed, err := opts.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		if completed.StorageBackend != apistorage.StorageTypeETCD3 {
			t.Errorf("expected default storage backend %v, got %v", apistorage.StorageTypeETCD3, completed.StorageBackend)
		}
		if completed.SocketPath != "/var/run/nvidia-device-api/kine.sock" {
			t.Errorf("expected Kine socket path, got %v", completed.SocketPath)
		}
		if completed.DatabaseDir != "/var/lib/nvidia-device-api" {
			t.Errorf("expected DatabaseDir derived from DSN, got %v", completed.DatabaseDir)
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

	t.Run("Custom DatabasePath", func(t *testing.T) {
		opts := options.NewOptions()
		opts.DatabasePath = "sqlite:///custom/path/data.db?_timeout=5000"

		completed, err := opts.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		expectedDir := "/custom/path"
		if completed.DatabaseDir != expectedDir {
			t.Errorf("expected DatabaseDir %q, got %q", expectedDir, completed.DatabaseDir)
		}
		expectedEndpoint := "sqlite:///custom/path/data.db?_timeout=5000"
		if completed.KineConfig.Endpoint != expectedEndpoint {
			t.Errorf("expected Kine endpoint to match custom DatabasePath, got %v", completed.KineConfig.Endpoint)
		}
	})

	t.Run("Disabled Connection Pooling", func(t *testing.T) {
		opts := options.NewOptions()
		opts.DatabaseMaxIdleConns = 0

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
		opts.StorageBackend = options.StorageTypeMemory

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
				o.StorageBackend = options.StorageTypeMemory
				o.DatabasePath = "" // should not trigger "required" error
			},
			wantErr: false,
		},
		{
			name: "Invalid storage backend type",
			modify: func(o *options.Options) {
				o.StorageBackend = "postgres"
			},
			wantErr:     true,
			errContains: "invalid, allowed values",
		},
		{
			name: "Storage initialization timeout too low",
			modify: func(o *options.Options) {
				o.StorageInitializationTimeout = 500 * time.Millisecond
			},
			wantErr:     true,
			errContains: "must be at least 1s",
		},
		{
			name: "Readycheck timeout exceeds initialization timeout",
			modify: func(o *options.Options) {
				o.StorageInitializationTimeout = 5 * time.Second
				o.StorageReadycheckTimeout = 10 * time.Second
			},
			wantErr:     true,
			errContains: "must be less than or equal to",
		},
		{
			name: "Relative database path",
			modify: func(o *options.Options) {
				o.DatabasePath = "relative/path.db"
			},
			wantErr:     true,
			errContains: "must be an absolute path",
		},
		{
			name: "Database path with sqlite scheme is valid",
			modify: func(o *options.Options) {
				o.DatabasePath = "sqlite:///var/lib/nvsentinel/state.db"
			},
			wantErr: false,
		},
		{
			name: "Database Max Connections invalid (SQLite WAL requirement)",
			modify: func(o *options.Options) {
				o.DatabaseMaxOpenConns = 1
			},
			wantErr:     true,
			errContains: "must be less than or equal to 0 (unlimited) or greater than 2",
		},
		{
			name: "Database connection max lifetime negative",
			modify: func(o *options.Options) {
				o.DatabaseMaxConnLifetime = -1 * time.Second
			},
			wantErr:     true,
			errContains: "must be 0 (unlimited) or a positive duration",
		},
		{
			name: "Idle connections cannot exceed Max open connections",
			modify: func(o *options.Options) {
				o.DatabaseMaxOpenConns = 5
				o.DatabaseMaxIdleConns = 10
			},
			wantErr:     true,
			errContains: "must be less than or equal to",
		},
		{
			name: "Compaction batch size too large",
			modify: func(o *options.Options) {
				o.EtcdCompactionBatchSize = 50000
			},
			wantErr:     true,
			errContains: "must be between",
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
			name: "Watch notify interval too low",
			modify: func(o *options.Options) {
				o.EtcdWatchProgressNotifyInterval = 1 * time.Second
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
