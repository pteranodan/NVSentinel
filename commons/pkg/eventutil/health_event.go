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
	"slices"
	"strings"
	"time"

	pb "github.com/nvidia/nvsentinel/data-models/pkg/protos"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SortByAge returns a clone of e sorted in ascending order by GeneratedTimestamp.
// Nil events or events with missing timestamps are placed at the beginning.
func SortByAge(e []*pb.HealthEvent) []*pb.HealthEvent {
	if len(e) == 0 {
		return e
	}

	s := slices.Clone(e)
	if len(s) == 1 {
		return s
	}

	slices.SortFunc(s, compareEvents)

	return s
}

func compareEvents(a, b *pb.HealthEvent) int {
	if a == b {
		return 0
	}

	if a == nil {
		return -1
	}

	if b == nil {
		return 1
	}

	ta, tb := a.GetGeneratedTimestamp(), b.GetGeneratedTimestamp()
	if ta == tb {
		return 0
	}

	if ta == nil {
		return -1
	}

	if tb == nil {
		return 1
	}

	return ta.AsTime().Compare(tb.AsTime())
}

// GetTime returns the GeneratedTimestamp of the event as a time.Time.
// If the event is nil or the timestamp is missing, it returns the provided 'now' time.
func GetTime(e *pb.HealthEvent, now time.Time) time.Time {
	if e == nil {
		return now
	}

	if ts := e.GetGeneratedTimestamp(); ts != nil {
		return ts.AsTime()
	}

	return now
}

// GetConditionStatus returns ConditionTrue if the event is unhealthy,
// ConditionFalse if it is healthy, or ConditionUnknown if the event is nil.
func GetConditionStatus(e *pb.HealthEvent) metav1.ConditionStatus {
	if e == nil {
		return metav1.ConditionUnknown
	}

	status := metav1.ConditionTrue
	if e.IsHealthy {
		status = metav1.ConditionFalse
	}

	return status
}

// GetNodeConditionStatus provides a corev1-compatible version of GetConditionStatus.
func GetNodeConditionStatus(e *pb.HealthEvent) corev1.ConditionStatus {
	return corev1.ConditionStatus(GetConditionStatus(e))
}

// GetReason returns a string in the format "[CheckName]IsHealthy" if the event is healthy,
// "[CheckName]IsNotHealthy" if it is unhealthy, or "CheckNameUnknown" if the event is nil.
func GetReason(e *pb.HealthEvent) string {
	if e == nil {
		return "CheckNameUnknown"
	}

	suffix := "IsNotHealthy"
	if e.IsHealthy {
		suffix = "IsHealthy"
	}

	return e.CheckName + suffix
}

const truncationSuffix = "... [truncated]"

// GetMessage returns a human-readable string from a HealthEvent, truncated to maxLength.
// It prioritizes the Message field, falling back to ErrorCodes or "None". If maxLength
// is non-positive, the string is returned untruncated. For small maxLength values,
// the function attempts to preserve signal by using a shorter suffix or hard truncation.
func GetMessage(e *pb.HealthEvent, maxLength int) string {
	if e == nil {
		return "None"
	}

	msg := e.GetMessage()
	if msg == "" {
		if len(e.GetErrorCode()) > 0 {
			msg = "ErrorCodes: " + strings.Join(e.GetErrorCode(), ", ")
		} else {
			msg = "None"
		}
	}

	if maxLength <= 0 || len(msg) <= maxLength {
		return msg
	}

	if maxLength > len(truncationSuffix) {
		return msg[:maxLength-len(truncationSuffix)] + truncationSuffix
	}

	const shortSuffix = "..."
	if maxLength > len(shortSuffix) {
		return msg[:maxLength-len(shortSuffix)] + shortSuffix
	}

	return msg[:maxLength]
}

// GetRecommendedAction returns the string representation of the event's recommended action.
// It returns nil if the event is nil or if the action is NONE or UNKNOWN.
func GetRecommendedAction(e *pb.HealthEvent) *string {
	if e == nil {
		return nil
	}

	rec := e.GetRecommendedAction()
	if rec == pb.RecommendedAction_NONE || rec == pb.RecommendedAction_UNKNOWN {
		return nil
	}

	actionStr := rec.String()

	return &actionStr
}

// GetWindow returns the earliest and latest timestamps as RFC3339 formatted strings;
// ok is true only if at least one valid timestamp was found.
func GetWindow(events []*pb.HealthEvent) (start, end string, ok bool) {
	if len(events) == 0 {
		return "", "", false
	}

	var startTime, endTime time.Time

	found := false

	for _, e := range events {
		ts := e.GetGeneratedTimestamp()
		if ts == nil {
			continue
		}

		curr := ts.AsTime()
		if !found {
			startTime, endTime = curr, curr
			found = true

			continue
		}

		if curr.Before(startTime) {
			startTime = curr
		}

		if curr.After(endTime) {
			endTime = curr
		}
	}

	if !found {
		return "", "", false
	}

	return startTime.Format(time.RFC3339), endTime.Format(time.RFC3339), found
}
