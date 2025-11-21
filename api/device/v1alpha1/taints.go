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

// These are valid, well-known but not exhaustive taint keys for devices.
const (
	// TaintGPUNotReady will be added when the device is not ready and removed when the GPU becomes ready.
	// Key: "gpu.nvidia.com/not-ready"
	TaintGPUNotReady = "gpu.nvidia.com/not-ready"

	// TaintGPUUnreachable will be added when the device becomes unreachable or
	// its status could not be determined, and removed when the GPU becomes reachable.
	// Key: "gpu.nvidia.com/unreachable"
	TaintGPUUnreachable = "gpu.nvidia.com/unreachable"

	// TaintGPUDegraded will be added when the device is operating below its expected performance
	// due to a non-fatal issue, and removed when it returns to normal operation.
	// Key: "gpu.nvidia.com/degraded"
	TaintGPUDegraded = "gpu.nvidia.com/degraded"

	// TaintGPUResetRequired will be added when the device is in a state that requires a reset
	// to become fully operational, and removed after the reset is complete.
	// Key: "gpu.nvidia.com/reset-required"
	TaintGPUResetRequired = "gpu.nvidia.com/reset-required"

	// TaintGPUHardwareFailure will be added when a persistent, unrecoverable hardware fault
	// is detected, and removed if the failure is resolved.
	// Key: "gpu.nvidia.com/hardware-failure"
	TaintGPUHardwareFailure = "gpu.nvidia.com/hardware-failure"
)
