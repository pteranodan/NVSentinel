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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGPURoundTrip(t *testing.T) {
	fixedTime := metav1.NewTime(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

	original := &GPU{
		Name: "test-gpu-resource",
		Spec: GPUSpec{
			ID:       "gpu-uuid-1234-5678",
			NodeName: "node-1",
		},
		Status: GPUStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionFalse,
					LastTransitionTime: fixedTime,
					Reason:             string(ReadyReasonDriverInitFailure),
					Message:            "Driver failed to load",
				},
			},
			RecommendedActions: []string{
				"ResetGPU",
				"ReportIssue",
			},
		},
	}

	// Go -> Proto
	protoMsg := original.ToProto()
	// Proto -> Go
	converted := GPUFromProto(protoMsg)

	if diff := cmp.Diff(original, converted); diff != "" {
		t.Errorf("GPU RoundTrip conversion mismatch (-want +got):\n%s", diff)
	}
}

func TestGPUListRoundTrip(t *testing.T) {
	fixedTime := metav1.NewTime(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC))

	original := &GPUList{
		Items: []GPU{
			{
				Name: "gpu-1",
				Spec: GPUSpec{
					ID:       "uuid-1",
					NodeName: "node-A",
				},
				Status: GPUStatus{
					Conditions: []metav1.Condition{
						{
							Type:               "Ready",
							Status:             metav1.ConditionFalse,
							LastTransitionTime: fixedTime,
							Reason:             string(ReadyReasonDriverInitFailure),
							Message:            "Driver failed to load",
						},
					},
					RecommendedActions: []string{
						"ResetGPU",
					},
				},
			},
			{
				Name: "gpu-2",
				Spec: GPUSpec{
					ID:       "uuid-2",
					NodeName: "node-B",
				},
				Status: GPUStatus{
					Conditions: []metav1.Condition{
						{
							Type:               "Ready",
							Status:             metav1.ConditionTrue,
							LastTransitionTime: fixedTime,
							Reason:             string(ReadyReasonDriverReady),
							Message:            "GPU is healthy",
						},
					},
					RecommendedActions: []string{},
				},
			},
		},
	}

	// Go -> Proto
	protoMsg := original.ToProto()
	// Proto -> Go
	converted := GPUListFromProto(protoMsg)

	if diff := cmp.Diff(original, converted); diff != "" {
		t.Errorf("GPUList RoundTrip conversion mismatch (-want +got):\n%s", diff)
	}
}

func TestNilValues(t *testing.T) {
	t.Run("ToProto handles nil", func(t *testing.T) {
		var gpu *GPU = nil
		if res := gpu.ToProto(); res != nil {
			t.Error("Expected nil result from nil GPU input")
		}

		var gpuList *GPUList = nil
		if res := gpuList.ToProto(); res != nil {
			t.Error("Expected nil result from nil GPUList input")
		}

		var spec *GPUSpec = nil
		if res := spec.ToProto(); res != nil {
			t.Error("Expected nil result from nil GPUSpec input")
		}

		var status *GPUStatus = nil
		if res := status.ToProto(); res != nil {
			t.Error("Expected nil result from nil GPUStatus input")
		}
	})

	t.Run("FromProto handles nil", func(t *testing.T) {
		if res := GPUFromProto(nil); res != nil {
			t.Error("Expected nil result from nil Proto input")
		}

		if res := GPUListFromProto(nil); res != nil {
			t.Error("Expected nil result from nil List Proto input")
		}

		if res := SpecFromProto(nil); res == nil {
			t.Error("Expected non-nil empty struct from nil Spec input")
		}

		if res := StatusFromProto(nil); res == nil {
			t.Error("Expected non-nil empty struct from nil Status input")
		}
	})
}
