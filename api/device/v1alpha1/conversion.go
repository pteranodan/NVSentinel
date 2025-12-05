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
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
)

// =============================================================================
// Go -> Proto
// =============================================================================

// ToProto converts the GPU object to a Proto message.
func (in *GPU) ToProto() *pb.Gpu {
	if in == nil {
		return nil
	}

	return &pb.Gpu{
		Name:   in.Name,
		Spec:   in.Spec.ToProto(),
		Status: in.Status.ToProto(),
	}
}

// ToProto converts the GPUList object to a Proto message.
func (in *GPUList) ToProto() *pb.GpuList {
	if in == nil {
		return nil
	}

	var items []*pb.Gpu
	if in.Items != nil {
		items = make([]*pb.Gpu, 0, len(in.Items))
		for i := range in.Items {
			items = append(items, in.Items[i].ToProto())
		}
	}

	return &pb.GpuList{
		Items: items,
	}
}

// ToProto converts the GPUSpec object to a Proto message.
func (in *GPUSpec) ToProto() *pb.GpuSpec {
	if in == nil {
		return nil
	}

	return &pb.GpuSpec{
		Id:       in.ID,
		NodeName: in.NodeName,
	}
}

// ToProto converts the GPUStatus object to a Proto message.
func (in *GPUStatus) ToProto() *pb.GpuStatus {
	if in == nil {
		return nil
	}

	var pbConds []*pb.Condition
	if in.Conditions != nil {
		pbConds = make([]*pb.Condition, 0, len(in.Conditions))
		for _, c := range in.Conditions {
			pbConds = append(pbConds, ConditionToProto(c))
		}
	}

	var pbActions []string
	if in.RecommendedActions != nil {
		pbActions = make([]string, 0, len(in.RecommendedActions))
		for _, a := range in.RecommendedActions {
			pbActions = append(pbActions, string(a))
		}
	}

	return &pb.GpuStatus{
		Conditions:         pbConds,
		RecommendedActions: pbActions,
	}
}

// ConditionToProto converts a k8s Condition to a Proto Condition.
func ConditionToProto(in metav1.Condition) *pb.Condition {
	return &pb.Condition{
		Type:               in.Type,
		Status:             string(in.Status),
		LastTransitionTime: timestamppb.New(in.LastTransitionTime.Time),
		Reason:             in.Reason,
		Message:            in.Message,
	}
}

// =============================================================================
// Proto -> Go
// =============================================================================

// GPUFromProto converts a Proto Gpu message to the Go GPU type.
func GPUFromProto(in *pb.Gpu) *GPU {
	if in == nil {
		return nil
	}

	return &GPU{
		Name:   in.Name,
		Spec:   *SpecFromProto(in.Spec),
		Status: *StatusFromProto(in.Status),
	}
}

// GPUListFromProto converts a Proto GpuList message to the Go GPUList type.
func GPUListFromProto(in *pb.GpuList) *GPUList {
	if in == nil {
		return nil
	}

	var items []GPU
	if in.Items != nil {
		items = make([]GPU, 0, len(in.Items))
		for _, item := range in.Items {
			if item != nil {
				items = append(items, *GPUFromProto(item))
			}
		}
	}

	return &GPUList{
		Items: items,
	}
}

// SpecFromProto converts a Proto GpuSpec message to the Go GPUSpec type.
func SpecFromProto(in *pb.GpuSpec) *GPUSpec {
	if in == nil {
		return &GPUSpec{}
	}

	return &GPUSpec{
		ID:       in.Id,
		NodeName: in.NodeName,
	}
}

// StatusFromProto converts a Proto GpuStatus message to the Go GPUStatus type.
func StatusFromProto(in *pb.GpuStatus) *GPUStatus {
	if in == nil {
		return &GPUStatus{}
	}

	var conds []metav1.Condition
	if in.Conditions != nil {
		conds = make([]metav1.Condition, 0, len(in.Conditions))
		for _, c := range in.Conditions {
			if c != nil {
				conds = append(conds, ConditionFromProto(c))
			}
		}
	}

	var actions []string
	if in.RecommendedActions != nil {
		actions = make([]string, 0, len(in.RecommendedActions))
		for _, a := range in.RecommendedActions {
			actions = append(actions, string(a))
		}
	}

	return &GPUStatus{
		Conditions:         conds,
		RecommendedActions: actions,
	}
}

// ConditionFromProto converts a Proto Condition to a k8s Condition.
func ConditionFromProto(in *pb.Condition) metav1.Condition {
	if in == nil {
		return metav1.Condition{}
	}

	var lastTransition metav1.Time
	if in.LastTransitionTime != nil {
		lastTransition = metav1.NewTime(in.LastTransitionTime.AsTime())
	}

	return metav1.Condition{
		Type:               in.Type,
		Status:             metav1.ConditionStatus(in.Status),
		LastTransitionTime: lastTransition,
		Reason:             in.Reason,
		Message:            in.Message,
	}
}
