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
)

func TestCompleteAndValidate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*Options)
		wantErr     bool
		errContains string
	}{
		{
			name:    "Valid defaults",
			modify:  func(o *Options) {},
			wantErr: false,
		},
		{
			name: "Empty database path",
			modify: func(o *Options) {
				o.DatabasePath = ""
			},
			wantErr:     true,
			errContains: "database-path: required",
		},
		{
			name: "Relative database path",
			modify: func(o *Options) {
				o.DatabasePath = "relative/path.db"
			},
			wantErr:     true,
			errContains: "must be an absolute path",
		},
		{
			name: "Invalid compaction batch size",
			modify: func(o *Options) {
				o.CompactionBatchSize = 0
			},
			wantErr:     true,
			errContains: "compaction-batch-size: 0 must be greater than 0",
		},
		{
			name: "Negative compaction interval",
			modify: func(o *Options) {
				o.CompactionInterval = -1 * time.Second
			},
			wantErr:     true,
			errContains: "compaction-interval: -1s must be 0s or greater",
		},
		{
			name: "Negative watch notify interval",
			modify: func(o *Options) {
				o.WatchProgressNotifyInterval = -1 * time.Second
			},
			wantErr:     true,
			errContains: "watch-progress-notify-interval: -1s must be 0s or greater",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := NewOptions()
			tt.modify(opts)

			completed, err := opts.Complete()
			if err != nil {
				t.Fatalf("Complete() failed: %v", err)
			}
			errs := completed.Validate()

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

func TestKineConfig(t *testing.T) {
	opts := NewOptions()
	opts.DatabasePath = "/tmp/test.db"
	opts.CompactionInterval = 10 * time.Minute
	opts.CompactionBatchSize = 500
	opts.WatchProgressNotifyInterval = 2 * time.Second

	completed, err := opts.Complete()
	if err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	expectedDSN := "sqlite:///tmp/test.db?_journal=WAL"
	if !strings.HasPrefix(completed.KineConfig.Endpoint, expectedDSN) {
		t.Errorf("DSN mismatch.\nGot: %s\nWant prefix: %s", completed.KineConfig.Endpoint, expectedDSN)
	}

	expectedListener := "unix:///var/run/nvidia-device-api/kine.sock"
	if completed.KineConfig.Listener != expectedListener {
		t.Errorf("Listener mismatch: got %s, want %s", completed.KineConfig.Listener, expectedListener)
	}

	if completed.KineConfig.CompactInterval != 10*time.Minute {
		t.Errorf("Kine CompactInterval mismatch")
	}
	if completed.KineConfig.CompactBatchSize != 500 {
		t.Errorf("Kine CompactBatchSize mismatch")
	}
	if completed.KineConfig.NotifyInterval != 2*time.Second {
		t.Errorf("Kine NotifyInterval mismatch")
	}

	if completed.DatabaseDir != "/tmp" {
		t.Errorf("DatabaseDir mismatch: got %s, want /tmp", completed.DatabaseDir)
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
