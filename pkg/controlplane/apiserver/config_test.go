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

package apiserver

import (
	"context"
	"testing"

	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/api"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/options"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/registry"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewConfig(t *testing.T) {
	ctx := context.Background()

	registry.Register(&fakeProvider{})

	opts := options.NewOptions()
	opts.NodeName = "test-apiserver"
	completedOpts, err := opts.Complete(ctx)
	if err != nil {
		t.Fatalf("failed to complete options: %v", err)
	}

	config, err := NewConfig(ctx, completedOpts)
	if err != nil {
		t.Fatalf("NewConfig failed: %v", err)
	}

	t.Run("BasicFields", func(t *testing.T) {
		if config.NodeName != "test-apiserver" {
			t.Errorf("NodeName mismatch: got %s", config.NodeName)
		}
		if config.HealthAddress == "" || config.MetricsAddress == "" {
			t.Error("Default addresses were not populated")
		}
	})

	t.Run("APIGroups", func(t *testing.T) {
		if len(config.APIGroups) == 0 {
			t.Error("APIGroups should not be empty when providers are registered")
		}
	})

	t.Run("ServerOptions", func(t *testing.T) {
		if len(config.ServerOptions) < 2 {
			t.Errorf("Expected at least 2 gRPC interceptors, got %d", len(config.ServerOptions))
		}
	})

	t.Run("StorageFactory", func(t *testing.T) {
		if config.StorageFactory == nil {
			t.Error("StorageFactory was not initialized")
		}
	})
}

func TestComplete(t *testing.T) {
	var c *Config
	completed, err := c.Complete()
	if err != nil || completed.Config != nil {
		t.Errorf("Complete() on nil should return empty wrap, got: %v", completed)
	}

	c = &Config{NodeName: "ready"}
	completed, _ = c.Complete()
	if completed.NodeName != "ready" {
		t.Error("Complete() failed to wrap existing config")
	}
}

type fakeProvider struct{}

func (f *fakeProvider) NewGroupInfo() *api.GroupInfo { return &api.GroupInfo{} }
func (f *fakeProvider) GroupName() string            { return "test.group" }
func (f *fakeProvider) GroupVersion() schema.GroupVersion {
	return schema.GroupVersion{Group: "test.group", Version: "v1"}
}
