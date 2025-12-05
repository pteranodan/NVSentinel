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

// ReadyReason defines the reasons for the 'Ready' condition.
type ReadyReason string

// These are valid but not exhaustive reasons for the 'Ready' condition.
const (
	// ReadyReasonDriverReady indicates that the device is healthy and ready to accept work.
	// This corresponds to Ready=True.
	ReadyReasonDriverReady ReadyReason = "DriverReady"

	// ReadyReasonStatusUnknown indicates that the status of the device could not be determined.
	ReadyReasonStatusUnknown ReadyReason = "StatusUnknown"

	// ReadyReasonDriverInitFailure indicates that the device driver failed to initialize the device
	// during startup (e.g., Xid 143).
	ReadyReasonDriverInitFailure ReadyReason = "DriverInitFailure"
)

// DegradedReason defines the reason for the 'Degraded' condition.
type DegradedReason string

// These are valid but not exhaustive reasons for the 'Degraded' condition.
const (
	// DegradedReasonNotDegraded indicates that the device is operating within its expected
	// performance range. This corresponds to Degraded=False.
	DegradedReasonNotDegraded DegradedReason = "NotDegraded"

	// DegradedReasonStatusUnknown indicates that it could not be determined whether the device is degraded.
	DegradedReasonStatusUnknown DegradedReason = "StatusUnknown"

	// -- Performance Throttling --

	// DegradedReasonPowerBrakeSlowdown indicates that core clocks are reduced by 50% or more
	// by the system power supply.
	DegradedReasonPowerBrakeSlowdown DegradedReason = "PowerBrakeSlowdown"

	// DegradedReasonHardwareSlowdown indicates that core clocks are reduced by 50% or more
	// due to high temperatures or power draw.
	DegradedReasonHardwareSlowdown DegradedReason = "HardwareSlowdown"

	// DegradedReasonThermalSlowdown indicates a hardware-level slowdown due to high temperatures.
	DegradedReasonThermalSlowdown DegradedReason = "ThermalSlowdown"

	// DegradedReasonSyncBoostSlowdown indicates that clocks are being held at a lower speed
	// to sync with another device in its sync boost group.
	DegradedReasonSyncBoostSlowdown DegradedReason = "SyncBoostSlowdown"

	// -- Memory Failures --

	// DegradedReasonContainedMemoryError indicates that a contained uncorrectable memory error
	// occurred affecting one application (e.g., Xid 94).
	DegradedReasonContainedMemoryError DegradedReason = "ContainedMemoryError"

	// DegradedReasonMemoryRowRemappingPending indicates that a memory page has been blacklisted
	// and is pending a permanent row remapping.
	DegradedReasonMemoryRowRemappingPending DegradedReason = "MemoryRowRemappingPending"
)

// ResetRequiredReason defines the reasons for the 'ResetRequired' condition.
type ResetRequiredReason string

// These are valid but not exhaustive reasons for the 'ResetRequired' condition.
const (
	// ResetRequiredReasonResetNotRequired indicates that the device does not require a reset.
	// This corresponds to ResetRequired=False.
	ResetRequiredReasonResetNotRequired ResetRequiredReason = "ResetNotRequired"

	// ResetRequiredReasonStatusUnknown indicates that it could not be determined whether the device requires a reset.
	ResetRequiredReasonStatusUnknown ResetRequiredReason = "StatusUnknown"

	// -- Device Hangs --

	// ResetRequiredReasonDeviceUnresponsive indicates that the device's main processing units
	// stopped responding or timed out (e.g., Xid 109).
	ResetRequiredReasonDeviceUnresponsive ResetRequiredReason = "DeviceUnresponsive"

	// ResetRequiredReasonGSPUnresponsive indicates that the GPU System Processor (GSP) firmware
	// is unresponsive or timed out (e.g., Xid 119, 120).
	ResetRequiredReasonGSPUnresponsive ResetRequiredReason = "GSPUnresponsive"

	// -- Critical Failures --

	// ResetRequiredReasonDoubleBitECCError indicates that a Double Bit ECC error occurred (e.g., Xid 48).
	ResetRequiredReasonDoubleBitECCError ResetRequiredReason = "DoubleBitECCError"

	// ResetRequiredReasonUnrecoveredECCError indicates that an uncorrectable error occurred,
	// affecting the driver's ability to mark pages for dynamic page offlining (e.g., Xid 140).
	ResetRequiredReasonUnrecoveredECCError ResetRequiredReason = "UnrecoveredECCError"

	// ResetRequiredReasonUncontainedMemoryError indicates that an uncontained memory error
	// occurred, affecting multiple applications (e.g., Xid 95).
	ResetRequiredReasonUncontainedMemoryError ResetRequiredReason = "UncontainedMemoryError"

	// ResetRequiredReasonRowRemappingFailure indicates that the device driver failed to remap a faulty
	// memory row (e.g., Xid 64).
	ResetRequiredReasonRowRemappingFailure ResetRequiredReason = "RowRemappingFailure"

	// ResetRequiredReasonTPCRepairPending indicates that a Texture Processing Cluster (TPC)
	// repair is pending (e.g., Xid 156).
	ResetRequiredReasonTPCRepairPending ResetRequiredReason = "TPCRepairPending"

	// ResetRequiredReasonChannelRepairPending indicates that a High Bandwidth Memory (HBM)
	// channel repair is pending (e.g., Xid 160).
	ResetRequiredReasonChannelRepairPending ResetRequiredReason = "ChannelRepairPending"

	// ResetRequiredReasonNVLinkError indicates that a recoverable NVLink interconnect error
	// was detected (e.g., Xid 74, 155).
	ResetRequiredReasonNVLinkError ResetRequiredReason = "NVLinkError"
)

// HardwareFailureReason defines the reasons for the 'HardwareFailure'
// condition.
type HardwareFailureReason string

// These are valid but not exhaustive reasons for the 'HardwareFailure'
// condition.
const (
	// HardwareFailureReasonNoFailure indicates that no hardware failure has been detected.
	// This corresponds to HardwareFailure=False.
	HardwareFailureReasonNoFailure HardwareFailureReason = "NoFailure"

	// HardwareFailureReasonStatusUnknown indicates that it could not be
	// determined whether the device has a hardware failure.
	HardwareFailureReasonStatusUnknown HardwareFailureReason = "StatusUnknown"

	// -- Unrecoverable Memory Failures --

	// HardwareFailureReasonRowRemappingThresholdExceeded indicates that the DRAM row remapping
	// has exceeded the RMA threshold.
	HardwareFailureReasonRowRemappingThresholdExceeded HardwareFailureReason = "RowRemappingThresholdExceeded"

	// HardwareFailureReasonSRAMUncorrectableErrorThresholdExceeded indicates
	// that the aggregate SRAM uncorrectable error count has exceeded the RMA
	// threshold.
	HardwareFailureReasonSRAMUncorrectableErrorThresholdExceeded HardwareFailureReason = "SRAMUncorrectableErrorThresholdExceeded"

	// -- Connection Failures --

	// HardwareFailureReasonGPUFallenOffBus indicates that the GPU has become inaccessible
	// over its PCIe connection (e.g., Xid 79).
	HardwareFailureReasonGPUFallenOffBus HardwareFailureReason = "GPUFallenOffBus"

	// HardwareFailureReasonNVLinkFailure indicates that a fatal NVLink interconnect error was detected.
	HardwareFailureReasonNVLinkFailure HardwareFailureReason = "NVLinkFailure"
)
