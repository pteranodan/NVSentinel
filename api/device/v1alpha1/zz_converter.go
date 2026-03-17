// Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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
	pb "github.com/nvidia/nvsentinel/internal/generated/proto/device/v1alpha1"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Converter is the interface used to generate type conversion methods between the
// Kubernetes Resource Model structs and the Protobuf message structs.
//
// goverter:converter
// goverter:output:file ./zz_generated.goverter.go
// goverter:extend FromProtoTypeMeta ToProtoTypeMeta FromProtoListTypeMeta ToProtoListTypeMeta FromProtoItems ToProtoItems FromProtoTimestamp ToProtoTimestamp
type Converter interface {
	// FromProto converts a protobuf Gpu pointer into a GPU pointer.
	//
	// goverter:map . TypeMeta | FromProtoTypeMeta
	FromProto(source *pb.Gpu) *GPU

	// ToProto converts a GPU pointer into a protobuf Gpu pointer.
	//
	// goverter:map TypeMeta TypeMeta | ToProtoTypeMeta
	// goverter:ignore state sizeCache unknownFields
	ToProto(source *GPU) *pb.Gpu

	// FromProtoList converts a protobuf GpuList pointer into a GPUList pointer.
	//
	// goverter:map . TypeMeta | FromProtoListTypeMeta
	// goverter:useZeroValueOnPointerInconsistency
	FromProtoList(source *pb.GpuList) *GPUList

	// ToProtoList converts a GPUList pointer into a protobuf GpuList pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	// goverter:map TypeMeta TypeMeta | ToProtoListTypeMeta
	ToProtoList(source *GPUList) *pb.GpuList

	// FromProtoItems converts a list of protobuf Gpu pointers to a list of GPU objects.
	//
	// goverter:useZeroValueOnPointerInconsistency
	FromProtoItems(source []*pb.Gpu) []GPU

	// ToProtoItems converts a list of GPU objects to a list of protobuf Gpu pointers.
	ToProtoItems(source []GPU) []*pb.Gpu

	// FromProtoObjectMeta converts a protobuf ObjectMeta pointer into a metav1.ObjectMeta object.
	//
	// goverter:map Uid UID
	// goverter:ignore GenerateName DeletionTimestamp DeletionGracePeriodSeconds
	// goverter:ignore OwnerReferences Finalizers ManagedFields SelfLink
	// goverter:useZeroValueOnPointerInconsistency
	FromProtoObjectMeta(source *pb.ObjectMeta) metav1.ObjectMeta

	// ToProtoObjectMeta converts a metav1.ObjectMeta into a protobuf Object pointer.
	//
	// goverter:map UID Uid
	// goverter:ignore state sizeCache unknownFields
	ToProtoObjectMeta(source metav1.ObjectMeta) *pb.ObjectMeta

	// FromProtoListMeta converts a protobuf ListMeta pointer into a metav1.ListMeta object.
	//
	// goverter:ignore SelfLink Continue RemainingItemCount
	// goverter:useZeroValueOnPointerInconsistency
	FromProtoListMeta(source *pb.ListMeta) metav1.ListMeta

	// ToProtoListMeta converts a metav1.ListMeta into a protobuf ListMeta pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	ToProtoListMeta(source metav1.ListMeta) *pb.ListMeta

	// FromProtoSpec converts a protobuf GpuSpec pointer into a GPUSpec object.
	//
	// goverter:map Uuid UUID
	// goverter:useZeroValueOnPointerInconsistency
	FromProtoSpec(source *pb.GpuSpec) GPUSpec

	// ToProtoSpec converts a GPUSpec object into a protobuf GpuSpec pointer.
	//
	// goverter:map UUID Uuid
	// goverter:ignore state sizeCache unknownFields
	ToProtoSpec(source GPUSpec) *pb.GpuSpec

	// FromProtoStatus converts a protobuf GpuStatus pointer into a GPUStatus object.
	// goverter:useZeroValueOnPointerInconsistency
	FromProtoStatus(source *pb.GpuStatus) GPUStatus

	// ToProtoStatus converts a GPUStatus object into a protobuf GpuStatus pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	ToProtoStatus(source GPUStatus) *pb.GpuStatus

	// FromProtoCondition converts a protobuf Condition pointer into a metav1.Condition pointer.
	//
	// goverter:useZeroValueOnPointerInconsistency
	FromProtoCondition(source *pb.Condition) metav1.Condition

	// ToProtoCondition converts a metav1.Condition pointer into a protobuf Condition pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	ToProtoCondition(source metav1.Condition) *pb.Condition

	// ToProtoGetOptions converts a metav1.GetOptions pointer into a protobuf GetOptions pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	ToProtoGetOptions(source *metav1.GetOptions) *pb.GetOptions

	// ToProtoUpdateOptions converts a metav1.UpdateOptions pointer into a protobuf UpdateOptions pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	ToProtoUpdateOptions(source *metav1.UpdateOptions) *pb.UpdateOptions

	// ToProtoListOptions converts a metav1.ListOptions pointer into a protobuf ListOptions pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	ToProtoListOptions(source *metav1.ListOptions) *pb.ListOptions

	// ToProtoDeleteOptions maps a metav1.DeleteOptions pointer into a protobuf DeleteOptions pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	ToProtoDeleteOptions(source *metav1.DeleteOptions) *pb.DeleteOptions

	// ToProtoPreconditions maps a metav1.Preconditions pointer into a protobuf Preconditions pointer.
	//
	// goverter:map UID Uid
	// goverter:ignore state sizeCache unknownFields
	ToProtoPreconditions(source *metav1.Preconditions) *pb.Preconditions

	// ToProtoPatchOptions maps a metav1.PatchOptions pointer in to a protobuf PatchOptions pointer.
	//
	// goverter:ignore state sizeCache unknownFields
	ToProtoPatchOptions(source *metav1.PatchOptions) *pb.PatchOptions
}

// FromProtoTypeMeta generates the standard TypeMeta for the GPU resource.
func FromProtoTypeMeta(_ *pb.Gpu) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "GPU",
		APIVersion: SchemeGroupVersion.String(),
	}
}

// FromProtoListTypeMeta generates the standard TypeMeta for the GPUList resource.
func FromProtoListTypeMeta(_ *pb.GpuList) metav1.TypeMeta {
	return metav1.TypeMeta{
		Kind:       "GPUList",
		APIVersion: SchemeGroupVersion.String(),
	}
}

func ToProtoTypeMeta(_ metav1.TypeMeta) *pb.TypeMeta {
	return &pb.TypeMeta{
		Kind:       "GPU",
		ApiVersion: SchemeGroupVersion.String(),
	}
}

func ToProtoListTypeMeta(_ metav1.TypeMeta) *pb.TypeMeta {
	return &pb.TypeMeta{
		Kind:       "GPUList",
		ApiVersion: SchemeGroupVersion.String(),
	}
}

func FromProtoItems(c Converter, s []*pb.Gpu) []GPU {
	if s == nil {
		return []GPU{}
	}

	items := make([]GPU, len(s))
	for i, item := range s {
		if item == nil {
			items[i] = GPU{}
			continue
		}
		res := c.FromProto(item)
		if res != nil {
			items[i] = *res
		}
	}
	return items
}

func ToProtoItems(c Converter, s []GPU) []*pb.Gpu {
	if s == nil {
		return nil
	}

	items := make([]*pb.Gpu, len(s))
	for i := range s {
		items[i] = c.ToProto(&s[i])
	}
	return items
}

// FromProtoTimestamp converts a protobuf Timestamp message to a metav1.Time.
func FromProtoTimestamp(source *timestamppb.Timestamp) metav1.Time {
	if source == nil {
		return metav1.Time{}
	}

	return metav1.NewTime(source.AsTime())
}

// ToProtoTimestamp converts a metav1.Time to a protobuf Timestamp message.
func ToProtoTimestamp(source metav1.Time) *timestamppb.Timestamp {
	if source.IsZero() {
		return nil
	}

	return timestamppb.New(source.Time)
}
