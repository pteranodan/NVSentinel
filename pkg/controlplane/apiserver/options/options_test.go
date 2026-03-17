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

package options_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/options"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
	cliflag "k8s.io/component-base/cli/flag"
)

func TestAddFlags(t *testing.T) {
	o := options.NewOptions()
	fss := &cliflag.NamedFlagSets{}
	o.AddFlags(fss)

	fs := fss.FlagSet("generic")
	args := []string{
		"--hostname-override=test-node",
		"--health-probe-bind-address=:1234",
		"--service-monitor-period=30s",
		"--metrics-bind-address=:5678",
		"--shutdown-grace-period=10s",
	}

	err := fs.Parse(args)
	if err != nil {
		t.Fatalf("Failed to parse generic flags: %v", err)
	}

	if o.NodeName != "test-node" {
		t.Errorf("expected NodeName %s, got %s", "test-node", o.NodeName)
	}

	if o.HealthAddress != ":1234" {
		t.Errorf("expected HealthAddress %s, got %s", ":1234", o.HealthAddress)
	}
	if o.ServiceMonitorPeriod != 30*time.Second {
		t.Errorf("expected ServiceMonitorPeriod %v, got %s", 30*time.Second, o.ServiceMonitorPeriod)
	}

	if o.MetricsAddress != ":5678" {
		t.Errorf("expected MetricsAddress %s, got %s", ":5678", o.MetricsAddress)
	}

	if o.ShutdownGracePeriod != 10*time.Second {
		t.Errorf("expected ShutdownGracePeriod %v, got %v", 10*time.Second, o.ShutdownGracePeriod)
	}

	debugFs := fss.FlagSet("debug")
	debugArgs := []string{
		"--enable-pprof=true",
		"--pprof-bind-address=:9012",
	}

	err = debugFs.Parse(debugArgs)

	if err != nil {
		t.Fatalf("Failed to parse debug flags: %v", err)
	}

	if o.EnablePprof != true {
		t.Errorf("expected EnablePprof true, got %v", o.EnablePprof)
	}
	if o.PprofAddress != ":9012" {
		t.Errorf("expected PprofAddress %s, got %s", ":9012", o.PprofAddress)
	}
}

func TestComplete(t *testing.T) {
	os.Unsetenv("NODE_NAME")

	t.Run("Default assignments", func(t *testing.T) {
		o := options.NewOptions()
		o.EnablePprof = true

		completed, err := o.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		if completed.HealthAddress != ":50051" {
			t.Errorf("expected default health address :50051, got %s", completed.HealthAddress)
		}
		if completed.MetricsAddress != ":9090" {
			t.Errorf("expected default metrics address :9090, got %s", completed.MetricsAddress)
		}
		if completed.PprofAddress != ":6060" {
			t.Errorf("expected default pprof address :6060, got %s", completed.PprofAddress)
		}

		if completed.ServiceMonitorPeriod != 10*time.Second {
			t.Errorf("expected default service monitor period 10s, got %s", completed.ServiceMonitorPeriod)
		}
		if completed.ShutdownGracePeriod != 25*time.Second {
			t.Errorf("expected default shutdown grace period 25s, got %v", completed.ShutdownGracePeriod)
		}

		if len(completed.Server) == 0 {
			t.Errorf("expected gRPC server options, got none")
		}

		if completed.Storage.Type != apistorage.StorageTypeETCD3 {
			t.Errorf("expected storage type to be %s, got %s", apistorage.StorageTypeETCD3, completed.Storage.Type)
		}
	})

	t.Run("NodeName resolution", func(t *testing.T) {
		o1 := options.NewOptions()
		o1.NodeName = "  UPPER-case-Node  "
		c1, err := o1.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}
		if c1.NodeName != "upper-case-node" {
			t.Errorf("expected normalized node name %q, got %q", "upper-case-node", c1.NodeName)
		}

		os.Setenv("NODE_NAME", "env-node-name")
		defer os.Unsetenv("NODE_NAME")

		o2 := options.NewOptions()
		o2.NodeName = ""
		c2, err := o2.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}
		if c2.NodeName != "env-node-name" {
			t.Errorf("expected node name from ENV %q, got %q", "env-node-name", c2.NodeName)
		}

		o3 := options.NewOptions()
		o3.NodeName = "manual-override"
		c3, err := o3.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}
		if c3.NodeName != "manual-override" {
			t.Errorf("manual override should beat ENV, got %q", c3.NodeName)
		}
	})

	t.Run("NodeName system hostname fallback", func(t *testing.T) {
		os.Unsetenv("NODE_NAME")
		o := options.NewOptions()
		o.NodeName = ""

		completed, err := o.Complete()
		if err != nil {
			t.Fatalf("Complete failed: %v", err)
		}

		expectedHostname, _ := os.Hostname()
		expectedHostname = strings.ToLower(expectedHostname)
		if completed.NodeName != expectedHostname {
			t.Errorf("expected node name to match system hostname %q, got %q", expectedHostname, completed.NodeName)
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
			name:    "Defaults valid",
			modify:  func(o *options.Options) {},
			wantErr: false,
		},
		{
			name:    "Valid NodeName",
			modify:  func(o *options.Options) { o.NodeName = "valid-node" },
			wantErr: false,
		},
		{
			name:        "Invalid NodeName",
			modify:      func(o *options.Options) { o.NodeName = "invalid_node_name" },
			wantErr:     true,
			errContains: "hostname-override \"invalid_node_name\":",
		},
		{
			name:        "Missing BindAddress",
			modify:      func(o *options.Options) { o.BindAddress = "" },
			wantErr:     true,
			errContains: "required",
		},
		{
			name:        "Invalid Unix socket URI",
			modify:      func(o *options.Options) { o.BindAddress = "/tmp/not-a-uri.sock" },
			wantErr:     true,
			errContains: "",
		},
		{
			name:        "Missing HealthAddress",
			modify:      func(o *options.Options) { o.HealthAddress = "" },
			wantErr:     true,
			errContains: "required",
		},
		{
			name:        "Invalid HealthAddress port",
			modify:      func(o *options.Options) { o.HealthAddress = "127.0.0.1:99999" },
			wantErr:     true,
			errContains: "\"127.0.0.1:99999\":",
		},
		{
			name:        "Invalid MetricsAddress port",
			modify:      func(o *options.Options) { o.MetricsAddress = "127.0.0.1:99999" },
			wantErr:     true,
			errContains: "\"127.0.0.1:99999\":",
		},
		{
			name: "Port collision: Health and Metrics",
			modify: func(o *options.Options) {
				o.HealthAddress = ":8080"
				o.MetricsAddress = ":8080"
			},
			wantErr:     true,
			errContains: "must not use the same port",
		},
		{
			name: "Pprof enabled but missing PprofAddress",
			modify: func(o *options.Options) {
				o.EnablePprof = true
				o.PprofAddress = ""
			},
			wantErr:     true,
			errContains: "required",
		},
		{
			name: "Invalid PprofAddress port",
			modify: func(o *options.Options) {
				o.EnablePprof = true
				o.PprofAddress = "127.0.0.1:99999"
			},
			wantErr:     true,
			errContains: "\"127.0.0.1:99999\":",
		},
		{
			name: "Port colission: Health and Pprof",
			modify: func(o *options.Options) {
				o.HealthAddress = ":8080"
				o.EnablePprof = true
				o.PprofAddress = ":8080"
			},
			wantErr:     true,
			errContains: "must not use the same port",
		},
		{
			name:        "ServiceMonitorPeriod negative",
			modify:      func(o *options.Options) { o.ServiceMonitorPeriod = -2 * time.Minute },
			wantErr:     true,
			errContains: "greater than or equal to 0s",
		},
		{
			name:        "ServiceMonitorPeriod out of bounds",
			modify:      func(o *options.Options) { o.ServiceMonitorPeriod = 2 * time.Minute },
			wantErr:     true,
			errContains: "or less",
		},
		{
			name:        "ShutdownGracePeriod negative",
			modify:      func(o *options.Options) { o.ShutdownGracePeriod = -2 * time.Minute },
			wantErr:     true,
			errContains: "greater than or equal to 0s",
		},
		{
			name:        "ShutdownGracePeriod out of bounds",
			modify:      func(o *options.Options) { o.ShutdownGracePeriod = 30 * time.Minute },
			wantErr:     true,
			errContains: "or less",
		},
		{
			name:        "Invalid gRPC option",
			modify:      func(o *options.Options) { o.GRPC.MaxSendMsgSize = 512 },
			wantErr:     true,
			errContains: "--grpc-max-send-msg-size",
		},
		{
			name:        "Invalid storage option",
			modify:      func(o *options.Options) { o.Storage.Type = apistorage.StorageTypeETCD2 },
			wantErr:     true,
			errContains: "--storage-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := options.NewOptions()
			tt.modify(o)

			completed, err := o.Complete()
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
