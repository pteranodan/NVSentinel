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

func TestIsValid(t *testing.T) {
	const validNodeName = "worker-node-1"
	const validGPUID = "GPU-00000000-0000-0000-0000-000000000001"

	tests := []struct {
		name        string
		gpu         GPU
		expectError bool
		errorMsg    string
	}{
		{
			name: "Is valid if Name set",
			gpu: GPU{
				Name: "custom-user-gpu",
				Spec: GPUSpec{NodeName: validNodeName, ID: validGPUID},
			},
			expectError: false,
		},
		{
			name: "Is valid if required fields are present",
			gpu: GPU{
				Name: "",
				Spec: GPUSpec{NodeName: validNodeName, ID: validGPUID},
			},
			expectError: false,
		},
		{
			name: "Is not valid if NodeName missing",
			gpu: GPU{
				Name: "valid-name",
				Spec: GPUSpec{NodeName: "", ID: validGPUID},
			},
			expectError: true,
			errorMsg:    "spec.nodeName is required",
		},
		{
			name: "Is not valid if NodeName is not valid format",
			gpu: GPU{
				Name: "valid-name",
				Spec: GPUSpec{NodeName: "Node With Space", ID: validGPUID},
			},
			expectError: true,
			errorMsg:    "must be a valid DNS-1123 subdomain",
		},
		{
			name: "Is not valid if GPU ID missing",
			gpu: GPU{
				Name: "valid-name",
				Spec: GPUSpec{NodeName: validNodeName, ID: ""},
			},
			expectError: true,
			errorMsg:    "spec.id is required",
		},
		{
			name: "Is not valid if GPU ID is not valid NVIDIA GPU UUID",
			gpu: GPU{
				Name: "valid-name",
				Spec: GPUSpec{NodeName: validNodeName, ID: "ID-1234-5678-9012-3456-789012345678"},
			},
			expectError: true,
			errorMsg:    "must be a valid NVIDIA GPU UUID",
		},
		{
			name: "Is not valid if Name is not valid format",
			gpu: GPU{
				Name: "Name/With/Slash",
				Spec: GPUSpec{NodeName: validNodeName, ID: validGPUID},
			},
			expectError: true,
			errorMsg:    "must be a valid DNS-1123 subdomain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gpu.IsValid()

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error, but got nil")
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Validation failed.\nExpected error to contain: '%s'\nGot: '%v'", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Fatalf("Expected success, but got error: %v", err)
				}
			}
		})
	}
}
