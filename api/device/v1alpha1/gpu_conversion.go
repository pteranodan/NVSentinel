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

//go:build !goverter

package v1alpha1

import (
	pb "github.com/nvidia/nvsentinel/internal/generated/proto/device/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// converter is the singleton instance of the generated Converter implementation.
var converter Converter = &ConverterImpl{}

// FromProto converts a protobuf Gpu message pointer to a GPU object pointer.
func FromProto(in *pb.Gpu) *GPU {
	if in == nil {
		return nil
	}

	val := converter.FromProtobuf(in)

	return &val
}

// ToProto converts a GPU object pointer to a protobuf Gpu message pointer.
func ToProto(in *GPU) *pb.Gpu {
	if in == nil {
		return nil
	}

	return converter.ToProtobuf(*in)
}

// FromProtoList converts a protobuf GpuList message pointer to a GPUList object pointer.
func FromProtoList(in *pb.GpuList) *GPUList {
	if in == nil {
		return nil
	}

	return converter.FromProtobufList(in)
}

// ToProtoList converts a GPUList object pointer to a protobuf GpuList message pointer.
func ToProtoList(in *GPUList) *pb.GpuList {
	if in == nil {
		return nil
	}

	return converter.ToProtobufList(in)
}

// ToProtoGetOptions converts a metav1.GetOptions to a protobuf GetOptions message pointer.
func ToProtoGetOptions(in metav1.GetOptions) *pb.GetOptions {
	return converter.ToProtobufGetOptions(in)
}

// ToProtoUpdateOptions converts a metav1.UpdateOptions to a protobuf UpdateOptions message pointer.
func ToProtoUpdateOptions(in metav1.UpdateOptions) *pb.UpdateOptions {
	return (*pb.UpdateOptions)(converter.ToProtobufUpdateOptions(in))
}

// ToProtoListOptions converts a metav1.ListOptions to a protobuf ListOptions message pointer.
func ToProtoListOptions(in metav1.ListOptions) *pb.ListOptions {
	return converter.ToProtobufListOptions(in)
}

// ToProtoDeleteOptions converts a metav1.DeleteOptions to a protobuf DeleteOptions message pointer.
func ToProtoDeleteOptions(in metav1.DeleteOptions) *pb.DeleteOptions {
	return converter.ToProtobufDeleteOptions(in)
}

// ToProtoPreconditions converts a metav1.Preconditions pointer to a protobuf Preconditions message pointer.
func ToProtoPreconditions(in *metav1.Preconditions) *pb.Preconditions {
	if in == nil {
		return nil
	}

	return converter.ToProtobufPreconditions(in)
}

// ToProtoPatchOptions converts a metav1.PatchOptions to a protobuf PatchOptions message pointer.
func ToProtoPatchOptions(in metav1.PatchOptions) *pb.PatchOptions {
	return converter.ToProtobufPatchOptions(in)
}
