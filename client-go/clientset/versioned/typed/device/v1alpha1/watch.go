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
	"context"
	"errors"
	"io"
	"sync/atomic"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// streamWatcher implements watch.Interface over a gRPC client stream.
type streamWatcher struct {
	cancel    context.CancelFunc // cancels the gRPC stream
	result    chan watch.Event   // channel delivering watch events
	stream    pb.GpuService_WatchGpusClient
	done      chan struct{} // closed when watcher stops
	sentError uint32        // ensures only one watch.Error is sent
	logger    logr.Logger
}

// newStreamWatcher creates a streamWatcher and starts receiving events.
func newStreamWatcher(
	stream pb.GpuService_WatchGpusClient,
	cancel context.CancelFunc,
	logger logr.Logger,
) watch.Interface {
	w := &streamWatcher{
		cancel: cancel,
		result: make(chan watch.Event, 100),
		stream: stream,
		done:   make(chan struct{}),
		logger: logger,
	}

	go w.receive()
	return w
}

// Stop cancels the gRPC stream and closes the watcher.
func (w *streamWatcher) Stop() {
	w.cancel()
	_ = w.stream.CloseSend()

	select {
	case <-w.done:
	default:
		close(w.done)
	}
}

// ResultChan returns the channel delivering watch events.
func (w *streamWatcher) ResultChan() <-chan watch.Event {
	return w.result
}

// receive reads events from the gRPC stream and sends them to result channel.
func (w *streamWatcher) receive() {
	defer close(w.result)

	for {
		resp, err := w.stream.Recv()
		if err != nil {
			code := status.Code(err)

			if errors.Is(err, io.EOF) || code == codes.Canceled {
				return
			}

			var msg string
			if code == codes.DeadlineExceeded {
				msg = "Watch stream deadline exceeded"
			} else {
				s := status.Convert(err)
				msg = s.Message()
			}

			statusErr := &metav1.Status{
				Status:  metav1.StatusFailure,
				Message: msg,
				Code:    int32(code),
				Reason:  metav1.StatusReasonInternalError,
			}
			if code == codes.OutOfRange {
				statusErr.Reason = metav1.StatusReasonExpired
				statusErr.Code = 410
			}

			w.trySendError(statusErr)
			return
		}

		eventType, isError := w.mapEventType(resp.Type)
		if eventType == "" {
			continue
		}
		if isError {
			return
		}

		obj := devicev1alpha1.FromProto(resp.Object)

		select {
		case <-w.done:
			return
		case w.result <- watch.Event{Type: eventType, Object: obj}:
		}
	}
}

// mapEventType converts a gRPC proto event type to watch.EventType.
// Returns the mapped type and a bool indicating whether the watch should stop.
func (w *streamWatcher) mapEventType(protoType string) (watch.EventType, bool) {
	switch protoType {
	case "ADDED":
		return watch.Added, false
	case "MODIFIED":
		return watch.Modified, false
	case "DELETED":
		return watch.Deleted, false
	case "ERROR":
		w.logger.Error(nil, "Watch stream received explicit ERROR event from server payload")
		w.trySendError(&metav1.Status{
			Status:  metav1.StatusFailure,
			Message: "Server sent explicit ERROR event",
			Code:    int32(codes.Internal),
			Reason:  metav1.StatusReasonInternalError,
		})
		return "", true
	default:
		w.logger.V(1).Info("Unknown watch event type ignored", "type", protoType)
		return "", false
	}
}

// trySendError sends a single watch.Error event if not already sent.
func (w *streamWatcher) trySendError(statusErr *metav1.Status) {
	if atomic.CompareAndSwapUint32(&w.sentError, 0, 1) {
		select {
		case <-w.done:
		case w.result <- watch.Event{Type: watch.Error, Object: statusErr}:
		}
	}
}
