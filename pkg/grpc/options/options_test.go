// Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// TODO: update test cases
package options_test

import (
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	grpcopts "github.com/nvidia/nvsentinel/pkg/grpc/options"
	"google.golang.org/grpc"
	cliflag "k8s.io/component-base/cli/flag"
)

func TestAddFlags(t *testing.T) {
	o := grpcopts.NewOptions()
	fss := &cliflag.NamedFlagSets{}
	o.AddFlags(fss)

	fs := fss.FlagSet("grpc")
	args := []string{
		"--grpc-max-streams-per-connection=500",
		"--grpc-max-recv-msg-size=1048576",
		"--grpc-max-send-msg-size=2097152",
		"--grpc-write-buffer-size=65536",
		"--grpc-shared-write-buffer=true",
		"--grpc-read-buffer-size=65536",
		"--grpc-initial-window-size=131072",
		"--grpc-initial-connection-window-size=262144",
		"--grpc-max-connection-age=20m",
		"--grpc-max-connection-age-grace=30s",
		"--grpc-max-connection-idle=10m",
		"--grpc-keepalive-time=31s",
		"--grpc-keepalive-timeout=8s",
		"--grpc-min-ping-interval=6s",
	}
	fs.Parse(args)

	expected := &grpcopts.Options{
		MaxConcurrentStreams:        500,
		MaxRecvMsgSize:              1048576,
		MaxSendMsgSize:              2097152,
		WriteBufferSize:             65536,
		SharedWriteBuffer:           true,
		ReadBufferSize:              65536,
		InitialWindowSize:           131072,
		InitialConnectionWindowSize: 262144,
		MaxConnectionAge:            20 * time.Minute,
		MaxConnectionAgeGrace:       30 * time.Second,
		MaxConnectionIdle:           10 * time.Minute,
		KeepAliveTime:               31 * time.Second,
		KeepAliveTimeout:            8 * time.Second,
		MinPingInterval:             6 * time.Second,
		PermitWithoutStream:         false,
	}

	if !reflect.DeepEqual(expected, o) {
		t.Errorf("Difference detected:\n%s", cmp.Diff(expected, o))
	}
}

func TestComplete(t *testing.T) {
	t.Run("Default assignments", func(t *testing.T) {
		o := &grpcopts.Options{}
		completed, err := o.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		if completed.MaxConcurrentStreams != 100 {
			t.Errorf("expected default streams 100, got %d", completed.MaxConcurrentStreams)
		}

		if completed.MaxRecvMsgSize != 4194304 {
			t.Errorf("expected default recv size 4MiB, got %d", completed.MaxRecvMsgSize)
		}
		if completed.MaxSendMsgSize != math.MaxInt32 {
			t.Errorf("expected default send size MaxInt32, got %d", completed.MaxSendMsgSize)
		}
		if completed.WriteBufferSize != 32768 {
			t.Errorf("expected default write buffer 32KiB, got %d", completed.WriteBufferSize)
		}
		if completed.ReadBufferSize != 32768 {
			t.Errorf("expected default read buffer 32KiB, got %d", completed.ReadBufferSize)
		}

		if completed.InitialWindowSize != 65535 {
			t.Errorf("expected default window 65535, got %d", completed.InitialWindowSize)
		}
		if completed.InitialConnectionWindowSize != 65535 {
			t.Errorf("expected default connection window 65535, got %d", completed.InitialConnectionWindowSize)
		}

		if completed.KeepAliveTime != 1*time.Minute {
			t.Errorf("expected default keepalive time 1m, got %v", completed.KeepAliveTime)
		}
		if completed.KeepAliveTimeout != 10*time.Second {
			t.Errorf("expected default keepalive timeout 10s, got %v", completed.KeepAliveTimeout)
		}
		if completed.MinPingInterval != 5*time.Second {
			t.Errorf("expected default min ping 5s, got %v", completed.MinPingInterval)
		}
	})

	t.Run("Preserve user overrides", func(t *testing.T) {
		o := grpcopts.NewOptions()
		o.MaxConcurrentStreams = 500

		completed, _ := o.Complete()
		if completed.MaxConcurrentStreams != 500 {
			t.Errorf("User override was lost: got %d", completed.MaxConcurrentStreams)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*grpcopts.Options)
		wantErr     bool
		errContains string
	}{
		{
			name:    "Valid defaults",
			modify:  func(o *grpcopts.Options) {},
			wantErr: false,
		},
		{
			name: "MaxConcurrentStreams out of range",
			modify: func(o *grpcopts.Options) {
				o.MaxConcurrentStreams = 20000
			},
			wantErr:     true,
			errContains: "must be between 1 and 10000",
		},
		{
			name: "MaxRecvMsgSize below minimum",
			modify: func(o *grpcopts.Options) {
				o.MaxRecvMsgSize = 512
			},
			wantErr:     true,
			errContains: "must be at least 1024",
		},
		{
			name: "Initial Window Size too small",
			modify: func(o *grpcopts.Options) {
				o.InitialWindowSize = 100
			},
			wantErr:     true,
			errContains: "must be between 65535",
		},
		{
			name: "Keepalive Timeout too high relative to Keepalive Time",
			modify: func(o *grpcopts.Options) {
				o.KeepAliveTime = 30 * time.Second
				o.KeepAliveTimeout = 40 * time.Second
			},
			wantErr:     true,
			errContains: "must be less than --grpc-keepalive-time",
		},
		{
			name: "Keepalive Time too aggressive",
			modify: func(o *grpcopts.Options) {
				o.KeepAliveTime = 5 * time.Second
			},
			wantErr:     true,
			errContains: "must be 10s or greater",
		},
		{
			name: "Keepalive Timeout out of range",
			modify: func(o *grpcopts.Options) {
				o.KeepAliveTimeout = 10 * time.Minute
			},
			wantErr:     true,
			errContains: "must be between 1s and 5m",
		},
		{
			name: "Min Ping Interval violation",
			modify: func(o *grpcopts.Options) {
				o.MinPingInterval = 1 * time.Second
			},
			wantErr:     true,
			errContains: "must be at least 5s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := grpcopts.NewOptions()
			tt.modify(o)

			completed, err := o.Complete()
			if err != nil {
				t.Fatalf("Complete failed during setup: %v", err)
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

func TestApplyTo(t *testing.T) {
	o := grpcopts.NewOptions()
	completed, _ := o.Complete()

	var serverOpts []grpc.ServerOption
	err := completed.ApplyTo(&serverOpts)
	if err != nil {
		t.Fatalf("ApplyTo failed: %v", err)
	}

	expectedCount := 10
	if len(serverOpts) != expectedCount {
		t.Errorf("Expected %d server options, got %d", expectedCount, len(serverOpts))
	}
}
