// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package options

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	cliflag "k8s.io/component-base/cli/flag"
)

func TestAddFlags(t *testing.T) {
	o := NewOptions()
	fss := &cliflag.NamedFlagSets{}
	o.AddFlags(fss)

	fs := fss.FlagSet("generic")
	args := []string{
		"--hostname-override=test-node",
		"--health-probe-bind-address=:1234",
		"--metrics-bind-address=:5678",
		"--shutdown-grace-period=10s",
	}

	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	if o.NodeName != "test-node" {
		t.Errorf("expected NodeName %s, got %s", "test-node", o.NodeName)
	}

	if o.HealthAddress != ":1234" {
		t.Errorf("expected HealthAddress %s, got %s", ":1234", o.HealthAddress)
	}

	if o.MetricsAddress != ":5678" {
		t.Errorf("expected MetricsAddress %s, got %s", ":5678", o.MetricsAddress)
	}

	if o.ShutdownGracePeriod != 10*time.Second {
		t.Errorf("expected ShutdownGracePeriod %v, got %v", 10*time.Second, o.ShutdownGracePeriod)
	}
}

func TestComplete(t *testing.T) {
	os.Unsetenv("NODE_NAME")

	t.Run("Default assignments", func(t *testing.T) {
		o := NewOptions()
		o.HealthAddress = ""
		o.MetricsAddress = ""
		o.ShutdownGracePeriod = 0

		completed, err := o.Complete(context.Background())
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		if completed.HealthAddress != ":50051" {
			t.Errorf("expected default health address :50051, got %s", completed.HealthAddress)
		}
		if completed.MetricsAddress != ":9090" {
			t.Errorf("expected default metrics address :9090, got %s", completed.MetricsAddress)
		}
		if completed.ShutdownGracePeriod != 25*time.Second {
			t.Errorf("expected default grace period 25s, got %v", completed.ShutdownGracePeriod)
		}
		if completed.NodeName == "" {
			t.Error("NodeName should have been populated from system hostname")
		}
	})

	t.Run("NodeName normalization", func(t *testing.T) {
		o := NewOptions()
		o.NodeName = "  UPPER-case-Node  "

		completed, _ := o.Complete(context.Background())

		expected := "upper-case-node"
		if completed.NodeName != expected {
			t.Errorf("Normalization failed. Got %q, want %q", completed.NodeName, expected)
		}
	})

	t.Run("Manual override takes precedence over ENV", func(t *testing.T) {
		os.Setenv("NODE_NAME", "env-value")
		defer os.Unsetenv("NODE_NAME")

		o := NewOptions()
		o.NodeName = "manual-override"

		completed, _ := o.Complete(context.Background())
		if completed.NodeName != "manual-override" {
			t.Errorf("Manual override should ignore ENV values. Got %q", completed.NodeName)
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
			name:    "Valid configuration",
			modify:  func(o *Options) { o.NodeName = "valid-node" },
			wantErr: false,
		},
		{
			name: "Invalid NodeName (DNS-1123)",
			modify: func(o *Options) {
				// NodeName is lowercased, but underscores are still illegal
				o.NodeName = "invalid_node_name"
			},
			wantErr:     true,
			errContains: "hostname-override \"invalid_node_name\":",
		},
		{
			name: "Invalid Health Address",
			modify: func(o *Options) {
				o.HealthAddress = "127.0.0.1:99999" // Port out of range
			},
			wantErr:     true,
			errContains: "health-probe-bind-address \"127.0.0.1:99999\":",
		},
		{
			name: "Address Collision",
			modify: func(o *Options) {
				o.HealthAddress = ":8080"
				o.MetricsAddress = ":8080"
			},
			wantErr:     true,
			errContains: "must not be the same (:8080)",
		},
		{
			name: "Negative Grace Period",
			modify: func(o *Options) {
				o.ShutdownGracePeriod = -5 * time.Second
			},
			wantErr:     true,
			errContains: "shutdown-grace-period: -5s must be greater than or equal to 0s",
		},
		{
			name: "Grace Period Exceeds Max",
			modify: func(o *Options) {
				o.ShutdownGracePeriod = 11 * time.Minute
			},
			wantErr:     true,
			errContains: "shutdown-grace-period: 11m0s must be 10m or less",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewOptions()
			tt.modify(o)

			completed, err := o.Complete(context.Background())
			if err != nil {
				t.Fatalf("Complete failed in test setup: %v", err)
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
