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
	"github.com/nvidia/nvsentinel/commons/pkg/errutil"
	"github.com/nvidia/nvsentinel/commons/pkg/eventutil"
	pb "github.com/nvidia/nvsentinel/data-models/pkg/protos"
	"github.com/nvidia/nvsentinel/platform-connectors/pkg/ringbuffer"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

const connectorName = "device"

// DeviceConnector synchronizes health events from a ring buffer to the NVIDIA Device API.
type DeviceConnector struct {
	clientset  deviceset.Interface
	ringBuffer *ringbuffer.RingBuffer
	stopCh     <-chan struct{}
}

// NewConnector returns a new DeviceConnector initialized with the provided clientset and buffer.
func NewConnector(
	client deviceset.Interface,
	ringBuffer *ringbuffer.RingBuffer,
	stopCh <-chan struct{}) *DeviceConnector {
	return &DeviceConnector{
		clientset:  client,
		ringBuffer: ringBuffer,
		stopCh:     stopCh,
	}
}

// InitializeConnector creates a DeviceConnector by discovering the API target
// from the NVIDIA_DEVICE_API_PATH environment variable.
func InitializeConnector(
	ringBuffer *ringbuffer.RingBuffer,
	stopCh <-chan struct{}) (*DeviceConnector, error) {
	target := os.Getenv("NVIDIA_DEVICE_API_PATH")

	cs, err := deviceset.NewForTarget(target)
	if err != nil {
		return nil, err
	}

	return NewConnector(cs, ringBuffer, stopCh), nil
}

// FetchAndProcessHealthMetric runs the main event loop, dequeuing health metrics and patching GPU status.
func (r *DeviceConnector) FetchAndProcessHealthMetric(ctx context.Context) {
	for {
		select {
		case <-r.stopCh:
			slog.InfoContext(ctx, "Stopping platform connector", "connector", connectorName)
			return
		case <-ctx.Done():
			slog.InfoContext(ctx, "Context cancelled; stopping connector", "connector", connectorName)
			return
		default:
			queued, quit := r.ringBuffer.Dequeue()
			if quit {
				slog.InfoContext(ctx, "Platform connector queue shut down; exiting loop", "connector", connectorName)
				return
			}

			r.processQueuedHealthEvents(ctx, queued)
		}
	}
}

func (r *DeviceConnector) processQueuedHealthEvents(ctx context.Context, queued *ringbuffer.QueuedHealthEvents) {
	if queued == nil || queued.Events == nil || len(queued.Events.GetEvents()) == 0 {
		r.ringBuffer.HealthMetricEleProcessingCompleted(queued)
		return
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

		return
	}

	r.ringBuffer.HealthMetricEleProcessingCompleted(queued)
}

func (r *DeviceConnector) processHealthEvents(ctx context.Context, healthEvents *pb.HealthEvents) error {
	eventsByGPU := r.groupByGPU(ctx, healthEvents)
	gpuCount := len(eventsByGPU)

	var errs error

	for name, events := range eventsByGPU {
		if gpuCount > 1 {
			// We delay the execution by a random duration to prevent
			// thundering herd issues against the API server.
			r.applyJitter(ctx, gpuCount)

			if err := ctx.Err(); err != nil {
				errs = errors.Join(errs, err)
			}
		}

		if err := r.processGPUEvents(ctx, name, events); err != nil {
			errs = errors.Join(errs, err)
		}
	}

	return errs
}

func (r *DeviceConnector) groupByGPU(ctx context.Context, healthEvents *pb.HealthEvents) map[string][]*pb.HealthEvent {
	eventsByGPU := make(map[string][]*pb.HealthEvent)

	for _, event := range healthEvents.Events {
		if event.ProcessingStrategy == pb.ProcessingStrategy_STORE_ONLY {
			r.logSkippedEvent(ctx, event, "store only", true)
			continue
		}

		if event.GetCheckName() == "" {
			r.logSkippedEvent(ctx, event, "empty check name", false)
			continue
		}

		for _, entity := range event.EntitiesImpacted {
			if entity.EntityType == "GPU" {
				name := strings.ToLower(entity.EntityValue)
				if name == "" {
					r.logSkippedEvent(ctx, event, "empty entity value", false)
					continue
				}

				eventsByGPU[name] = append(eventsByGPU[name], event)
			}
		}
	}

	return eventsByGPU
}

func (r *DeviceConnector) logSkippedEvent(ctx context.Context, event *pb.HealthEvent, reason string, storeOnly bool) {
	logArgs := []any{
		"connector", connectorName,
		"node", event.GetNodeName(),
		"agent", event.GetAgent(),
		"checkName", event.GetCheckName(),
		"event_id", event.GetId(),
		"reason", reason,
		"skipped", true,
		"storeOnly", storeOnly,
	}

	if storeOnly {
		slog.InfoContext(ctx, "Skipping health event", logArgs...)
	} else {
		slog.WarnContext(ctx, "Skipping health event", logArgs...)
	}
}

// applyJitter delays execution by a random duration.
// The jitter window scales logarithmically with the count.
func (r *DeviceConnector) applyJitter(ctx context.Context, count int) {
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
	case <-ctx.Done():
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
		latestByCheck[event.GetCheckName()] = event
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

	err = retry.OnError(retry.DefaultBackoff,
		func(err error) bool {
			return apierrors.IsConflict(err) || errutil.IsTemporaryError(err)
		},
		func() error {
			_, err = r.clientset.DeviceV1alpha1().GPUs().Patch(ctx,
				name,
				types.StrategicMergePatchType,
				patchBytes,
				metav1.PatchOptions{},
				"status",
			)

			return err
		},
	)
	if err != nil {
		return fmt.Errorf("failed to patch GPU %q status: %w", name, err)
	}

	return nil
}

// Stop gracefully shuts down the connector by draining the internal queue
// and closing the clientset.
func (r *DeviceConnector) Stop(ctx context.Context) error {
	if r.ringBuffer != nil {
		slog.InfoContext(ctx, "Shutting down platform connector queue", "connector", connectorName)
		r.ringBuffer.ShutDownHealthMetricQueue()
		slog.InfoContext(ctx, "Platform connector queue drained", "connector", connectorName)
	}

	if r.clientset != nil {
		return r.clientset.Close()
	}

	return nil
}
