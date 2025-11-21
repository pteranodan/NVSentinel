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

// RecommendedAction defines recommended actions for a resource.
type RecommendedAction string

// These are valid but not exhaustive recommended actions for resources.
const (
	// RecommendedActionRestartApp indicates that the affected applications should be restarted.
	RecommendedActionRestartApp RecommendedAction = "RestartApp"

	// RecommendedActionResetGPU indicates that the GPU needs to be reset.
	// Refer to the 'nvidia-smi' command-line utility documentation for details.
	RecommendedActionResetGPU RecommendedAction = "ResetGPU"

	// RecommendedActionRestartVM indicates that the virtual machine must be restarted.
	RecommendedActionRestartVM RecommendedAction = "RestartVM"

	// RecommendedActionRebootNode indicates that the host node must be rebooted.
	RecommendedActionRebootNode RecommendedAction = "RebootNode"

	// RecommendedActionResetFabric indicates that the system's interconnect fabric must be reset.
	// Refer to the NVIDIA Fabric Manager Guide for details.
	RecommendedActionResetFabric RecommendedAction = "ResetFabric"

	// RecommendedActionRunExtendedUtilityDiagnostics indicates that the NVIDIA CPU Extended Utility Diagnostics (CPU EUD)
	// should be run to detect potential system problems.
	RecommendedActionRunExtendedUtilityDiagnostics RecommendedAction = "RunExtendedUtilityDiagnostics"

	// RecommendedActionRunFieldDiagnostics indicates that the NVIDIA Field Diagnostics should be run
	// to detect potential system problems.
	RecommendedActionRunFieldDiagnostics RecommendedAction = "RunFieldDiagnostics"

	// RecommendedActionReportIssue indicates that the issue should be reported to the system vendor for analysis.
	RecommendedActionReportIssue RecommendedAction = "ReportIssue"
)
