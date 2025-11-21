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

package v1alpha1

import (
	"strings"
	"testing"
)

const (
	validNodeName = "worker-node-1"
	validGPUID    = "GPU-00000000-0000-0000-0000-000000000001"

	expectedHashID = "89ebd4226844"
	validPrefix    = "some-prefix-"
)

func testGPU() *GPU {
	return &GPU{
		Spec: GPUSpec{
			NodeName: validNodeName,
			ID:       validGPUID,
		},
		GenerateName: validPrefix,
	}
}

func TestComputeCanonicalName(t *testing.T) {
	t.Run("Respects Existing Name", func(t *testing.T) {
		g := testGPU()
		existingName := "existing-name-123"
		g.Name = existingName
		name, _ := g.ComputeCanonicalName()
		if name != existingName {
			t.Errorf("Failed to return existing Name. Want: %v, Got: %v", existingName, name)
		}
	})

	t.Run("Computes Name", func(t *testing.T) {
		g := testGPU()
		expectedName := validPrefix + expectedHashID

		name, err := g.ComputeCanonicalName()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if name != expectedName {
			t.Errorf("Name mismatch. Want: %v, Got: %v", expectedName, name)
		}
		if g.Name != "" {
			t.Errorf("Name was mutated. Want: %v, Got: %v", expectedName, g.Name)
		}
	})

	t.Run("Errors when missing GPU ID", func(t *testing.T) {
		g := testGPU()
		g.Name = ""
		g.Spec.ID = ""

		_, err := g.ComputeCanonicalName()
		if err == nil || !strings.Contains(err.Error(), "required to compute canonical path") {
			t.Errorf("Expected error for missing GPU ID. Got: %v", err)
		}
		if g.Name != "" {
			t.Errorf("Name was set. Want: %v, Got: %v", "", g.Name)
		}
	})

	t.Run("Errors when missing NodeName", func(t *testing.T) {
		g := testGPU()
		g.Name = ""
		g.Spec.NodeName = ""

		_, err := g.ComputeCanonicalName()
		if err == nil || !strings.Contains(err.Error(), "required to compute canonical path") {
			t.Errorf("Expected error for missing NodeName. Got: %v", err)
		}
		if g.Name != "" {
			t.Errorf("Name was set. Want: %v, Got: %v", "", g.Name)
		}
	})
}

func TestEnsureName(t *testing.T) {
	t.Run("Respects Existing Name", func(t *testing.T) {
		g := testGPU()
		existingName := "existing-name-123"
		g.Name = existingName

		finalName, err := g.EnsureName()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if finalName != existingName {
			t.Errorf("Failed to respect existing Name. Want: %v, Got: %v", existingName, finalName)
		}
		if g.Name != existingName {
			t.Errorf("Name was mutated. Want: %v, Got: %v", existingName, g.Name)
		}
	})

	t.Run("Generates Name When Missing", func(t *testing.T) {
		g := testGPU()
		g.Name = ""

		expected := validPrefix + expectedHashID

		finalName, err := g.EnsureName()
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if finalName != expected {
			t.Errorf("Incorrect name generated. Want: %v, Got: %v", expected, finalName)
		}
		if g.Name != expected {
			t.Errorf("Failed to set Name. Want: %v, Got: %v", expected, g.Name)
		}
	})

	t.Run("Errors when missing GPU ID", func(t *testing.T) {
		g := testGPU()
		g.Name = ""
		g.Spec.ID = ""

		_, err := g.EnsureName()
		if err == nil || !strings.Contains(err.Error(), "required to compute canonical path") {
			t.Errorf("Expected error for missing GPU ID. Got: %v", err)
		}
		if g.Name != "" {
			t.Errorf("Name was set. Got: %v", g.Name)
		}
	})

	t.Run("Errors when missing NodeName", func(t *testing.T) {
		g := testGPU()
		g.Name = ""
		g.Spec.NodeName = ""

		_, err := g.EnsureName()
		if err == nil || !strings.Contains(err.Error(), "required to compute canonical path") {
			t.Errorf("Expected error for missing NodeName. Got: %v", err)
		}
		if g.Name != "" {
			t.Errorf("Name was set. Got: %v", g.Name)
		}
	})
}
