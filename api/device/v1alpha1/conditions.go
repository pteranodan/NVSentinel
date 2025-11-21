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

// GPUConditionType defines a GPU's condition.
type GPUConditionType string

// These are valid but not exhaustive conditions of a GPU.
// Relevant events contain 'Ready', 'Degraded', 'ResetRequired', and 'HardwareFailure'.
const (
	// GPUReady indicates whether the GPU is healthy and ready to accept work.
	// Status="True" means the GPU is healthy. Status="False" means the GPU is unhealthy
	// and not accepting work. Status="Unknown" means the state of the GPU could not be determined.
	GPUReady GPUConditionType = "Ready"

	// GPUDegraded indicates that the GPU is functional but operating below its expected performance
	// due to a non-fatal issue (e.g., throttled outside the acceptable operating range).
	// Status="True" means the GPU is degraded. Status="False" means it is operating normally.
	GPUDegraded GPUConditionType = "Degraded"

	// GPUResetRequired indicates that the GPU is in a state that requires a reset to become
	// fully operational again.
	// Status="True" means a reset is required. Status="False" means no reset is needed.
	GPUResetRequired GPUConditionType = "ResetRequired"

	// GPUHardwareFailure indicates that the GPU has a persistent, unrecoverable hardware fault.
	// Status="True" means a fatal failure is detected. Status="False" means no hardware failure is detected.
	GPUHardwareFailure GPUConditionType = "HardwareFailure"
)
