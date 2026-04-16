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

package device

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	deviceset "github.com/nvidia/device-api/client-go/clientset/device"
	pb "github.com/nvidia/nvsentinel/data-models/pkg/protos"
	"github.com/nvidia/nvsentinel/platform-connectors/pkg/ringbuffer"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const connectorName = "device"

// DeviceConnector synchronizes health events from a ring buffer to the NVIDIA Device API.
type DeviceConnector struct {
	clientset  deviceset.Interface
	ringBuffer *ringbuffer.RingBuffer
	stopCh     <-chan struct{}
	ctx        context.Context
}

// NewConnector returns a new DeviceConnector initialized with the provided clientset and buffer.
func NewConnector(
	ctx context.Context,
	client deviceset.Interface,
	ringBuffer *ringbuffer.RingBuffer,
	stopCh <-chan struct{}) *DeviceConnector {
	return &DeviceConnector{
		clientset:  client,
		ringBuffer: ringBuffer,
		stopCh:     stopCh,
		ctx:        ctx,
	}
}

// InitializeConnector creates a DeviceConnector by discovering the API target
// from the NVIDIA_DEVICE_API_PATH environment variable.
func InitializeConnector(
	ctx context.Context,
	ringBuffer *ringbuffer.RingBuffer,
	stopCh <-chan struct{}) (*DeviceConnector, error) {
	target := os.Getenv("NVIDIA_DEVICE_API_PATH")

	cs, err := deviceset.NewForTarget(target)
	if err != nil {
		return nil, err
	}

	return NewConnector(ctx, cs, ringBuffer, stopCh), nil
}

// FetchAndProcessHealthMetric runs the main event loop, dequeuing health metrics and patching GPU status.
//
// nolint:cyclop,gocognit,nestif // Complexity due to inline timestamp windowing for rich error logging
func (r *DeviceConnector) FetchAndProcessHealthMetric(ctx context.Context) {
	for {
		select {
		case <-r.stopCh:
			slog.InfoContext(r.ctx, "Stopping platform connector", "connector", connectorName)
			return
		default:
			queued, quit := r.ringBuffer.Dequeue()
			if quit {
				slog.InfoContext(ctx, "Platform connector queue shut down; exiting loop", "connector", connectorName)
				return
			}

			if queued == nil || queued.Events == nil || len(queued.Events.GetEvents()) == 0 {
				r.ringBuffer.HealthMetricEleProcessingCompleted(queued)
				continue
			}

			healthEvents := queued.Events

			if err := r.processHealthEvents(ctx, healthEvents); err != nil {
				events := healthEvents.GetEvents()
				count := len(events)

				logFields := []any{
					"connector", connectorName,
					"error", err,
					"event_count", count,
					"version", healthEvents.GetVersion(),
				}
				if count > 0 {
					var minTSPtr, maxTSPtr *timestamppb.Timestamp

					var minTime, maxTime time.Time

					for _, e := range events {
						ts := e.GetGeneratedTimestamp()
						if ts == nil {
							continue
						}

						currentTime := ts.AsTime()

						if minTSPtr == nil || currentTime.Before(minTime) {
							minTime = currentTime
							minTSPtr = ts
						}

						if maxTSPtr == nil || currentTime.After(maxTime) {
							maxTime = currentTime
							maxTSPtr = ts
						}
					}

					if minTSPtr != nil && maxTSPtr != nil {
						logFields = append(logFields,
							"window_start", minTime.Format(time.RFC3339),
							"window_end", maxTime.Format(time.RFC3339),
						)
					}
				}

				slog.ErrorContext(ctx, "Failed to process health events", logFields...)
				r.ringBuffer.HealthMetricEleProcessingFailed(queued)
			} else {
				r.ringBuffer.HealthMetricEleProcessingCompleted(queued)
			}
		}
	}
}

func (r *DeviceConnector) processHealthEvents(ctx context.Context, healthEvents *pb.HealthEvents) error {
	eventsByGPU := make(map[string][]*pb.HealthEvent)

	for _, event := range healthEvents.Events {
		if event.ProcessingStrategy == pb.ProcessingStrategy_STORE_ONLY {
			slog.InfoContext(ctx, "Skipping health event: store only",
				"connector", connectorName,
				"event_id", event.GetId(),
				"checkName", event.GetCheckName(),
				"agent", event.GetAgent(),
				"skipped", true,
				"storeOnly", true)

			continue
		}

		for _, entity := range event.EntitiesImpacted {
			if entity.EntityType == "GPU" {
				name := strings.ToLower(entity.EntityValue)
				if name == "" {
					slog.WarnContext(ctx, "Skipping health event: empty entity value",
						"connector", connectorName,
						"event_id", event.GetId(),
						"checkName", event.GetCheckName(),
						"agent", event.GetAgent(),
						"entity_type", entity.GetEntityType(),
						"entity_value", entity.GetEntityValue(),
						"skipped", true,
						"storeOnly", false)

					continue
				}

				eventsByGPU[name] = append(eventsByGPU[name], event)
			}
		}
	}

	var firstErr error

	for name, events := range eventsByGPU {
		if err := r.processGPUEvents(ctx, name, events); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

type statusPatch struct {
	Status struct {
		Conditions []metav1.Condition `json:"conditions"`
		// TODO(pteranodan): Change *string once GPUStatus.RecommendedAction is *string.
		RecommendedAction string `json:"recommendedAction,omitempty"`
	} `json:"status"`
}

// nolint:cyclop,gocognit // Pipeline for sorting, deduplicating, and mapping events
func (r *DeviceConnector) processGPUEvents(ctx context.Context, name string, events []*pb.HealthEvent) error {
	if len(events) == 0 {
		return nil
	}

	sortedEvents := slices.Clone(events)
	slices.SortFunc(sortedEvents, func(a, b *pb.HealthEvent) int {
		ta := a.GetGeneratedTimestamp()
		tb := b.GetGeneratedTimestamp()

		if ta == nil && tb == nil {
			return 0
		}

		if ta == nil {
			return -1
		}

		if tb == nil {
			return 1
		}

		return ta.AsTime().Compare(tb.AsTime())
	})

	var latestEvent *pb.HealthEvent

	latestByCheck := make(map[string]*pb.HealthEvent)

	for _, event := range sortedEvents {
		checkName := event.GetCheckName()
		if checkName == "" {
			slog.WarnContext(ctx, "Skipping health event: empty check name",
				"connector", connectorName,
				"event_id", event.GetId(),
				"agent", event.GetAgent(),
				"entity_type", "GPU",
				"entity_value", name,
				"skipped", true,
				"storeOnly", false)

			continue
		}

		latestByCheck[checkName] = event
		latestEvent = event
	}

	var conditionsToPatch []metav1.Condition

	const maxMsgLen = 1024 // 1KB limit

	const truncationSuffix = "... [truncated]"

	for checkName, latest := range latestByCheck {
		status := metav1.ConditionFalse
		reason := fmt.Sprintf("%sIsHealthy", checkName)

		if !latest.IsHealthy {
			status = metav1.ConditionTrue
			reason = fmt.Sprintf("%sIsNotHealthy", checkName)
		}

		var transitionTime metav1.Time
		if eventTimestamp := latest.GetGeneratedTimestamp(); eventTimestamp != nil {
			transitionTime = metav1.NewTime(eventTimestamp.AsTime())
		} else {
			transitionTime = metav1.Now()
		}

		message := latest.GetMessage()
		if message == "" {
			if len(latest.GetErrorCode()) > 0 {
				message = "ErrorCodes: " + strings.Join(latest.GetErrorCode(), ", ")
			} else {
				message = "None"
			}
		}

		if len(message) > maxMsgLen {
			if maxMsgLen > len(truncationSuffix) {
				message = message[:maxMsgLen-len(truncationSuffix)] + truncationSuffix
			} else {
				message = message[:maxMsgLen]
			}
		}

		conditionsToPatch = append(conditionsToPatch, metav1.Condition{
			Type:               checkName,
			Status:             status,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: transitionTime,
		})
	}

	if len(conditionsToPatch) == 0 {
		return nil
	}

	p := statusPatch{}
	p.Status.Conditions = conditionsToPatch

	if latestEvent != nil {
		action := latestEvent.GetRecommendedAction()
		if action != pb.RecommendedAction_NONE && action != pb.RecommendedAction_UNKNOWN {
			p.Status.RecommendedAction = action.String()
		} else {
			// TODO(pteranodan): Remove "None" once RecommendedAction is a *string.
			// Strategic Merge Patch skips "" due to omitempty, leaving stale data.
			p.Status.RecommendedAction = "None"
		}
	}

	bytes, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal status patch for GPU %q: %w", name, err)
	}

	_, err = r.clientset.DeviceV1alpha1().GPUs().Patch(ctx,
		name,
		types.StrategicMergePatchType,
		bytes,
		metav1.PatchOptions{},
		"status",
	)
	if err != nil {
		return fmt.Errorf("failed to patch GPU %q status: %w", name, err)
	}

	return nil
}

// Stop gracefully shuts down the connector by draining the internal queue
// and closing the clientset.
func (r *DeviceConnector) Stop() error {
	if r.ringBuffer != nil {
		slog.InfoContext(r.ctx, "Shutting down platform connector queue", "connector", connectorName)
		r.ringBuffer.ShutDownHealthMetricQueue()
		slog.InfoContext(r.ctx, "Platform connector queue drained", "connector", connectorName)
	}

	if r.clientset != nil {
		return r.clientset.Close()
	}

	return nil
}
