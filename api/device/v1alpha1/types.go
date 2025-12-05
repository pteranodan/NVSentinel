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

// +k8s:deepcopy-gen=package
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen=true
// GPU represents a single GPU resource.
type GPU struct {
	// Name is the unique name for the GPU.
	// It is primarily used for lookup and creation idempotence.
	// If empty during creation, the server will generate a name.
	// +optional
	Name string `json:"name,omitempty"`

	// +optional
	Spec GPUSpec `json:"spec,omitempty"`
	// +optional
	Status GPUStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen=true
// GPUList is a list of GPUs.
type GPUList struct {
	// +optional
	Items []GPU `json:"items,omitempty"`
}

// +k8s:deepcopy-gen=true
// GPUSpec defines the desired state for a specific GPU.
type GPUSpec struct {
	// ID is the GPU's UUID.
	ID string `json:"id"`
	// NodeName is the name of the node where the GPU is located.
	NodeName string `json:"nodeName"`
}

// +k8s:deepcopy-gen=true
// GPUStatus describes the observed state of a single GPU.
type GPUStatus struct {
	// Conditions represents the observations of a GPU's current state.
	// Known condition types are "Ready", "Degraded", "ResetRequired", and
	// "HardwareFailure". The 'Reason' field in each condition corresponds to
	// specific error patterns (e.g., "DoubleBitECCError", "GPUFallenOffBus").
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// RecommendedActions is a list of suggested remediation steps to resolve the issues reported in Conditions.
	// Examples: "ResetGPU", "RebootNode", "ReportIssue".
	// +optional
	RecommendedActions []string `json:"recommendedActions,omitempty"`
}
