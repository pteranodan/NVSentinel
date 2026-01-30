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
	ctx := context.Background()

	t.Run("Default values are applied when empty", func(t *testing.T) {
		opts := NewOptions()
		opts.NodeName = ""
		opts.HealthAddress = ""

		completed, err := opts.Complete(ctx)
		if err != nil {
			t.Fatalf("Failed to complete options: %v", err)
		}

		if completed.HealthAddress != ":50051" {
			t.Errorf("Expected default health address :50051, got %s", completed.HealthAddress)
		}
		if completed.NodeName == "" {
			t.Error("NodeName should have been populated via Hostname or Env")
		}
	})

	t.Run("Manual overrides are preserved", func(t *testing.T) {
		opts := NewOptions()
		opts.NodeName = "Custom-Node"
		opts.HealthAddress = "127.0.0.1:8080"

		completed, err := opts.Complete(ctx)
		if err != nil {
			t.Fatalf("Failed to complete: %v", err)
		}

		if completed.NodeName != "custom-node" {
			t.Errorf("Expected lowercased node name, got %s", completed.NodeName)
		}
	})
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name        string
		modify      func(*Options)
		expectError bool
	}{
		{
			name:        "Valid options",
			modify:      func(o *Options) {},
			expectError: false,
		},
		{
			name: "Invalid DNS NodeName",
			modify: func(o *Options) {
				o.NodeName = "Invalid_Name_With_Underscores"
			},
			expectError: true,
		},
		{
			name: "Invalid HealthAddress port",
			modify: func(o *Options) {
				o.HealthAddress = "localhost:999999"
			},
			expectError: true,
		},
		{
			name: "Negative ShutdownGracePeriod",
			modify: func(o *Options) {
				o.ShutdownGracePeriod = -1 * time.Second
			},
			expectError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := NewOptions()
			tc.modify(opts)

			completed, _ := opts.Complete(context.Background())
			errs := completed.Validate()

			if tc.expectError && len(errs) == 0 {
				t.Errorf("Expected errors but got none")
			}
			if !tc.expectError && len(errs) > 0 {
				t.Errorf("Expected no errors but got: %v", errs)
			}
		})
	}
}
