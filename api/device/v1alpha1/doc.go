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

// Package v1alpha1 defines the device health API types and conversion
// utilities for GPU health monitoring in Kubernetes environments.
//
// This package provides Go types that follow Kubernetes API conventions
// (spec/status pattern) and support bidirectional conversion to/from
// Protocol Buffer messages for gRPC communication. It enables Kubernetes
// controllers to receive GPU device health information from custom health
// providers.
//
// Key types:
//   - GPU: Represents a single GPU resource with spec and status
//   - GPUStatus: Contains conditions and recommended remediation actions
//   - GPUConditionType: Defines health states (Ready, Degraded, etc.)
//
// The API uses metav1.Condition for status reporting, making it compatible
// with standard Kubernetes condition patterns and enabling integration with
// tools like node-problem-detector.
//
// +k8s:deepcopy-gen=package
// +k8s:defaulter-gen=package
package v1alpha1
