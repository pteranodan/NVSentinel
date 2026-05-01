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
	"errors"
	"fmt"
	"log/slog"
	"math/bits"
	rand "math/rand/v2"
	"os"
	"strings"
	"time"

	"github.com/nvidia/device-api/api/device/v1alpha1"
	deviceset "github.com/nvidia/device-api/client-go/clientset/device"
	"github.com/nvidia/nvsentinel/commons/pkg/eventutil"
	pb "github.com/nvidia/nvsentinel/data-models/pkg/protos"
	"github.com/nvidia/nvsentinel/platform-connectors/pkg/ringbuffer"
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
				logFields := []any{
					"connector", connectorName,
					"error", err,
					"count", len(events),
					"version", healthEvents.GetVersion(),
				}

				if start, end, ok := eventutil.GetWindow(events); ok {
					logFields = append(logFields, "window_start", start, "window_end", end)
				}

				slog.ErrorContext(ctx, "Failed to process health events", logFields...)

				slog.DebugContext(ctx, "Health event payload",
					"version", healthEvents.GetVersion(),
					"events", events,
				)

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
				"node", event.GetNodeName(),
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
						"node", event.GetNodeName(),
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

	gpuCount := len(eventsByGPU)

	var errs error

	for name, events := range eventsByGPU {
		if gpuCount > 1 {
			// We delay the execution by a random duration to prevent
			// thundering herd issues against the API server.
			r.applyJitter(gpuCount)
		}

		if err := r.processGPUEvents(ctx, name, events); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

// applyJitter delays execution by a random duration.
// The jitter window scales logarithmically with the count.
func (r *DeviceConnector) applyJitter(count int) {
	if count <= 1 {
		return
	}

	// By using bits.Len(count) as the shift, the jitter window doubles as the
	// request count doubles, which keeps the total request density stable.
	shift := bits.Len(uint(count))
	if shift > 10 {
		shift = 10 // Cap at ~1ms to keep responsiveness.
	}

	mask := uint32((1 << shift) - 1)

	// We use a bitwise AND with a mask for a zero-allocation random duration
	// in the range of [0, 2^shift - 1] microseconds.
	// #nosec G404 // crypto/rand is unnecessary for jitter
	delay := time.Duration(rand.Uint32()&mask) * time.Microsecond

	// Ensure the delay is interruptible to prevent hanging during connector shutdown.
	select {
	case <-time.After(delay):
	case <-r.stopCh:
	case <-r.ctx.Done():
	}
}

func (r *DeviceConnector) processGPUEvents(ctx context.Context, name string, events []*pb.HealthEvent) error {
	if len(events) == 0 {
		return nil
	}

	latestByCheck := make(map[string]*pb.HealthEvent)

	var latest *pb.HealthEvent

	sortedEvents := eventutil.SortByAge(events)
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
		latest = event
	}

	var conditionsToPatch []metav1.Condition

	now := time.Now()

	for checkName, event := range latestByCheck {
		status := eventutil.GetConditionStatus(event)

		conditionsToPatch = append(conditionsToPatch, metav1.Condition{
			Type:               checkName,
			Status:             status,
			Reason:             eventutil.GetReason(event),
			Message:            eventutil.GetMessage(event, 1024), // 1KB limit
			LastTransitionTime: metav1.NewTime(eventutil.GetTime(event, now)),
		})
	}

	if len(conditionsToPatch) == 0 {
		return nil
	}

	status := v1alpha1.GPUStatus{
		Conditions:        conditionsToPatch,
		RecommendedAction: eventutil.GetRecommendedAction(latest),
	}

	patch := struct {
		Status v1alpha1.GPUStatus `json:"status"`
	}{Status: status}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("failed to marshal status patch for GPU %q: %w", name, err)
	}

	_, err = r.clientset.DeviceV1alpha1().GPUs().Patch(ctx,
		name,
		types.StrategicMergePatchType,
		patchBytes,
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
