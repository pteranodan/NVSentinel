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

package eventutil

import (
	"strings"
	"testing"
	"time"

	pb "github.com/nvidia/nvsentinel/data-models/pkg/protos"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSortByAge(t *testing.T) {
	now := time.Now()
	e1 := &pb.HealthEvent{GeneratedTimestamp: timestamppb.New(now.Add(time.Hour))}
	e2 := &pb.HealthEvent{GeneratedTimestamp: timestamppb.New(now)}
	// Nil events or events with missing timestamps are placed at the beginning.
	e3 := &pb.HealthEvent{GeneratedTimestamp: nil}
	input := []*pb.HealthEvent{e1, e2, e3, nil}
	sorted := SortByAge(input)

	if len(sorted) != 4 || sorted[2] != e2 || sorted[3] != e1 {
		t.Errorf("SortByAge failed to order events correctly")
	}
}

func TestGetConditionStatus(t *testing.T) {
	tests := []struct {
		event  *pb.HealthEvent
		expect metav1.ConditionStatus
	}{
		{nil, metav1.ConditionUnknown},
		{&pb.HealthEvent{IsHealthy: true}, metav1.ConditionFalse},
		{&pb.HealthEvent{IsHealthy: false}, metav1.ConditionTrue},
	}

	for _, tt := range tests {
		if got := GetConditionStatus(tt.event); got != tt.expect {
			t.Errorf("GetConditionStatus(%v) = %v; want %v", tt.event, got, tt.expect)
		}
	}
}

func TestGetMessage(t *testing.T) {
	const limit = 1024
	const truncationSuffix = "... [truncated]" // len: 15
	const shortSuffix = "..."                  // len: 3

	tests := []struct {
		name      string
		event     *pb.HealthEvent
		maxLength int
		expected  string
	}{
		{
			name:      "Nil event",
			event:     nil,
			maxLength: limit,
			expected:  "None",
		},
		{
			name:      "Empty event",
			event:     &pb.HealthEvent{},
			maxLength: limit,
			expected:  "None",
		},
		{
			name: "Fallback to ErrorCodes",
			event: &pb.HealthEvent{
				ErrorCode: []string{"404", "500"},
			},
			maxLength: limit,
			expected:  "ErrorCodes: 404, 500",
		},
		{
			name: "Exact match",
			event: &pb.HealthEvent{
				Message: strings.Repeat("A", 10),
			},
			maxLength: 10,
			expected:  strings.Repeat("A", 10),
		},
		{
			name: "Standard truncation",
			event: &pb.HealthEvent{
				Message: "GPU Temperature High",
			},
			maxLength: 18,
			expected:  "GPU" + truncationSuffix,
		},
		{
			name: "Short truncation",
			event: &pb.HealthEvent{
				Message: "Critical",
			},
			maxLength: 7,
			expected:  "Crit" + shortSuffix,
		},
		{
			name: "Hard truncate",
			event: &pb.HealthEvent{
				Message: "Fatal",
			},
			maxLength: 1,
			expected:  "F",
		},
		{
			name: "Unlimited maxLength",
			event: &pb.HealthEvent{
				Message: "No limits",
			},
			maxLength: 0,
			expected:  "No limits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMessage(tt.event, tt.maxLength)
			if got != tt.expected {
				t.Errorf("GetMessage() = %q, want %q", got, tt.expected)
			}

			if tt.maxLength > 0 && len(got) > tt.maxLength {
				t.Errorf("GetMessage() length %d exceeds maxLength %d", len(got), tt.maxLength)
			}
		})
	}
}

func TestGetReason(t *testing.T) {
	tests := []struct {
		event  *pb.HealthEvent
		expect string
	}{
		{nil, "CheckNameUnknown"},
		{&pb.HealthEvent{CheckName: "GpuXidError", IsHealthy: true}, "GpuXidErrorIsHealthy"},
		{&pb.HealthEvent{CheckName: "GpuXidError", IsHealthy: false}, "GpuXidErrorIsNotHealthy"},
	}

	for _, tt := range tests {
		if got := GetReason(tt.event); got != tt.expect {
			t.Errorf("GetReason(%v) = %v; want %v", tt.event, got, tt.expect)
		}
	}
}

func TestGetRecommendedAction(t *testing.T) {
	// NONE/UNKNOWN s/b filtered out
	if got := GetRecommendedAction(&pb.HealthEvent{RecommendedAction: pb.RecommendedAction_NONE}); got != nil {
		t.Error("Expected nil for NONE")
	}

	action := pb.RecommendedAction_RESTART_VM
	got := GetRecommendedAction(&pb.HealthEvent{RecommendedAction: action})
	if got == nil || *got != "RESTART_VM" {
		t.Errorf("Expected RESTART_VM, got %v", got)
	}
}

func TestGetTime(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	eventTime := now.Add(time.Hour)

	tests := []struct {
		name     string
		event    *pb.HealthEvent
		expected time.Time
	}{
		{
			name:     "Nil event returns provided now",
			event:    nil,
			expected: now,
		},
		{
			name:     "Missing timestamp returns provided now",
			event:    &pb.HealthEvent{GeneratedTimestamp: nil},
			expected: now,
		},
		{
			name: "Valid timestamp returns actual event time",
			event: &pb.HealthEvent{
				GeneratedTimestamp: timestamppb.New(eventTime),
			},
			expected: eventTime,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTime(tt.event, now)
			if !got.Equal(tt.expected) {
				t.Errorf("GetTime() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetWindow(t *testing.T) {
	t1 := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	events := []*pb.HealthEvent{
		{GeneratedTimestamp: timestamppb.New(t1)},
		{GeneratedTimestamp: timestamppb.New(t2)},
		{GeneratedTimestamp: nil},
	}

	start, end, ok := GetWindow(events)
	if !ok || start != t1.Format(time.RFC3339) || end != t2.Format(time.RFC3339) {
		t.Errorf("GetWindow failed: start=%s, end=%s, ok=%v", start, end, ok)
	}

	if _, _, ok := GetWindow(nil); ok {
		t.Error("GetWindow should return ok=false for empty input")
	}
}
