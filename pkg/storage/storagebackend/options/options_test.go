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
	"strings"
	"testing"
	"time"

	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
	cliflag "k8s.io/component-base/cli/flag"
)

func TestAddFlags(t *testing.T) {
	o := NewOptions()
	fss := &cliflag.NamedFlagSets{}
	o.AddFlags(fss)

	fs := fss.FlagSet("storage")
	args := []string{
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
	t.Run("Default assignments", func(t *testing.T) {
		opts := NewOptions()
		opts.DatabasePath = ""

		completed, err := opts.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		if completed.KineSocketPath != "/var/run/nvidia-device-api/kine.sock" {
			t.Errorf("expected default socket path, got %s", completed.KineSocketPath)
		}
		if completed.KineConfig.Listener != "unix:///var/run/nvidia-device-api/kine.sock" {
			t.Errorf("expected default listener URI, got %s", completed.KineConfig.Listener)
		}
		if !strings.Contains(completed.KineConfig.Endpoint, IN_MEMORY) {
			t.Errorf("expected default DSN to contain %s, got %s", IN_MEMORY, completed.KineConfig.Endpoint)
		}
		if completed.KineConfig.ConnectionPoolConfig.MaxOpen != defaultDatabaseMaxOpenConns {
			t.Errorf("expected database max open connections %d, got %d", defaultDatabaseMaxOpenConns, completed.KineConfig.ConnectionPoolConfig.MaxOpen)
		}
		if completed.KineConfig.ConnectionPoolConfig.MaxIdle != defaultDatabaseMaxIdleConns {
			t.Errorf("expected database max idle connections %d, got %d", defaultDatabaseMaxIdleConns, completed.KineConfig.ConnectionPoolConfig.MaxIdle)
		}
		if completed.KineConfig.ConnectionPoolConfig.MaxLifetime != defaultDatabaseMaxConnLifetime {
			t.Errorf("expected database max connection lifetime %d, got %d", defaultDatabaseMaxConnLifetime, completed.KineConfig.ConnectionPoolConfig.MaxLifetime)
		}
		if completed.DatabaseDir != "/tmp" {
			t.Errorf("expected database directory '/tmp', got %s", completed.DatabaseDir)
		}

		if completed.KineConfig.EmulatedETCDVersion != defaultEtcdVersion {
			t.Errorf("expected etcd version %s, got %s", defaultEtcdVersion, completed.KineConfig.EmulatedETCDVersion)
		}
		if completed.KineConfig.CompactInterval != defaultEtcdCompactionInterval {
			t.Errorf("expected etcd compaction interval %d, got %d", defaultEtcdCompactionInterval, completed.KineConfig.CompactInterval)
		}
		if completed.KineConfig.CompactBatchSize != defaultEtcdCompactionBatchSize {
			t.Errorf("expected etcd compaction batch size %d, got %d", defaultEtcdCompactionBatchSize, completed.KineConfig.CompactBatchSize)
		}
		if completed.KineConfig.NotifyInterval != defaultEtcdWatchProgressNotifyInterval {
			t.Errorf("expected etcd watch progress notify interval %d, got %d", defaultEtcdWatchProgressNotifyInterval, completed.KineConfig.NotifyInterval)
		}
	})

	t.Run("Database max idle connections 0 maps to -1", func(t *testing.T) {
		opts := NewOptions()
		opts.DatabaseMaxIdleConns = 0

		completed, _ := opts.Complete()
		if completed.KineConfig.ConnectionPoolConfig.MaxIdle != -1 {
			t.Errorf("expected database max idle connections -1 for input 0, got %d", completed.KineConfig.ConnectionPoolConfig.MaxIdle)
		}
	})

	t.Run("Etcd compaction batch size 0 maps to default", func(t *testing.T) {
		opts := NewOptions()
		opts.EtcdCompactionBatchSize = 0

		completed, _ := opts.Complete()
		if completed.EtcdCompactionBatchSize != defaultEtcdCompactionBatchSize {
			t.Errorf("expected etcd compaction batch size %d for input 0, got %d", defaultEtcdCompactionBatchSize, completed.EtcdCompactionBatchSize)
		}
	})

	t.Run("Etcd watch progress notify interval 0 maps to default", func(t *testing.T) {
		opts := NewOptions()
		opts.EtcdWatchProgressNotifyInterval = 0

		completed, _ := opts.Complete()
		if completed.EtcdWatchProgressNotifyInterval != defaultEtcdWatchProgressNotifyInterval {
			t.Errorf("expected etcd watch progress notify interval %d for input 0, got %d", defaultEtcdWatchProgressNotifyInterval, completed.EtcdWatchProgressNotifyInterval)
		}
	})

	t.Run("Trims unix prefix from SocketPath", func(t *testing.T) {
		opts := NewOptions()
		opts.KineSocketPath = "unix:///tmp/test.sock"

		completed, _ := opts.Complete()
		if completed.KineSocketPath != "/tmp/test.sock" {
			t.Errorf("Complete should trim prefix from SocketPath: got %s", completed.KineSocketPath)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*Options)
		wantErr     bool
		errContains string
	}{
		{
			name: "Valid defaults",
			modify: func(o *Options) {
				o.Complete()
			},
			wantErr: false,
		},
		{
			name: "Database path is required",
			modify: func(o *Options) {
				o.DatabasePath = ""
			},
			wantErr:     true,
			errContains: "required",
		},
		{
			name: "Database path must be absolute",
			modify: func(o *Options) {
				o.DatabasePath = "relative/path.db"
			},
			wantErr:     true,
			errContains: "must be an absolute path",
		},
		{
			name: "Database max open connections floor",
			modify: func(o *Options) {
				o.DatabaseMaxOpenConns = 1
			},
			wantErr:     true,
			errContains: "must be 2 or greater",
		},
		{
			name: "Database max idle conns cannot exceed max open conns",
			modify: func(o *Options) {
				o.DatabaseMaxOpenConns = 2
				o.DatabaseMaxIdleConns = 5
			},
			wantErr:     true,
			errContains: "cannot be greater than --database-max-open-connections",
		},
		{
			name: "Database max idle connections must be postive",
			modify: func(o *Options) {
				o.DatabaseMaxIdleConns = -1
			},
			wantErr:     true,
			errContains: "must be 0 or greater",
		},
		{
			name: "Database max connection lifetime must be positive",
			modify: func(o *Options) {
				o.DatabaseMaxConnLifetime = -1 * time.Minute
			},
			wantErr:     true,
			errContains: "must be 0s or greater",
		},
		{
			name: "Database max connection lifetime must be at least 1s",
			modify: func(o *Options) {
				o.DatabaseMaxConnLifetime = 5 * time.Millisecond
			},
			wantErr:     true,
			errContains: "must be 0s (unlimited) or at least 1s",
		},
		{
			name: "Etcd version is required",
			modify: func(o *Options) {
				o.EtcdVersion = ""
			},
			wantErr:     true,
			errContains: "required",
		},
		{
			name: "Etcd compaction interval must be at least 1m",
			modify: func(o *Options) {
				o.EtcdCompactionInterval = 10 * time.Second
			},
			wantErr:     true,
			errContains: "must be 1m or greater",
		},
		{
			name: "Etcd compaction batch size must be greater than 0",
			modify: func(o *Options) {
				o.EtcdCompactionBatchSize = 0
			},
			wantErr:     true,
			errContains: "must be between 1 and",
		},
		{
			name: "Etcd compaction batch size must be less than 10000",
			modify: func(o *Options) {
				o.EtcdCompactionBatchSize = 15000
			},
			wantErr:     true,
			errContains: "must be between 1 and 10000",
		},
		{
			name: "Etcd watch progress notify interval must be greater than 5s",
			modify: func(o *Options) {
				o.EtcdWatchProgressNotifyInterval = 1 * time.Second
			},
			wantErr:     true,
			errContains: "must be between 5s and",
		},
		{
			name: "Etcd watch progress notify interval must be less than 10m",
			modify: func(o *Options) {
				o.EtcdWatchProgressNotifyInterval = 11 * time.Minute
			},
			wantErr:     true,
			errContains: "must be between 5s and 10m",
		},
		{
			name: "Default socket path is valid",
			modify: func(o *Options) {
				o.KineSocketPath = ""
				// Satisfy the upstream Etcd validator
				if o.Etcd != nil && o.Etcd.StorageConfig.Transport.ServerList == nil {
					o.Etcd.StorageConfig.Transport.ServerList = []string{"unix:///var/run/nvidia-device-api/kine.sock"}
				}
			},
			wantErr: false,
		},
		{
			name: "Relative kine socket path",
			modify: func(o *Options) {
				o.KineSocketPath = "relative/socket.sock"
			},
			wantErr:     true,
			errContains: "must be an absolute path",
		},
		{
			name: "Invalid unix socket URI format",
			modify: func(o *Options) {
				o.KineConfig.Listener = "http://localhost:8080"
			},
			wantErr:     true,
			errContains: "kine listener",
		},
		{
			name: "Malformed unix URI",
			modify: func(o *Options) {
				o.KineConfig.Listener = "unix:tmp/abs/path.sock"
			},
			wantErr:     true,
			errContains: "kine listener",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
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

func TestApplyTo(t *testing.T) {
	opts := NewOptions()
	completed, _ := opts.Complete()

	storageCfg := &apistorage.Config{}
	err := completed.ApplyTo(storageCfg)
	if err != nil {
		t.Fatalf("ApplyTo failed: %v", err)
	}

	if len(storageCfg.Transport.ServerList) == 0 {
		t.Error("ApplyTo failed to populate ServerList")
	}

	if storageCfg.Transport.ServerList[0] != completed.KineConfig.Listener {
		t.Errorf("StorageConfig server mismatch. Got %v, want %v",
			storageCfg.Transport.ServerList[0], completed.KineConfig.Listener)
	}
}
