package v1alpha1

import (
	"context"
	"errors"
	"io"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type streamWatcher struct {
	cancel context.CancelFunc
	result chan watch.Event
	stream pb.GpuService_WatchGpusClient
	done   chan struct{}
	logger logr.Logger
}

func newStreamWatcher(stream pb.GpuService_WatchGpusClient, cancel context.CancelFunc, logger logr.Logger) watch.Interface {
	w := &streamWatcher{
		cancel: cancel,
		result: make(chan watch.Event),
		stream: stream,
		done:   make(chan struct{}),
		logger: logger,
	}

	go w.receive()
	return w
}

func (w *streamWatcher) Stop() {
	w.cancel()

	select {
	case <-w.done:
		// Already closed
	default:
		close(w.done)
	}
}

func (w *streamWatcher) ResultChan() <-chan watch.Event {
	return w.result
}

func (w *streamWatcher) receive() {
	defer close(w.result)
	for {
		resp, err := w.stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			if status.Code(err) == codes.Canceled {
				return
			}

			statusErr := &metav1.Status{
				Status:  metav1.StatusFailure,
				Message: err.Error(),
				Code:    500,
				Reason:  metav1.StatusReasonInternalError,
			}

			select {
			case w.result <- watch.Event{
				Type:   watch.Error,
				Object: statusErr,
			}:
			case <-w.done: // Exit immediately if Stop() was called
				return
			}
			return
		}

		var eventType watch.EventType
		switch resp.Type {
		case "ADDED":
			eventType = watch.Added
		case "MODIFIED":
			eventType = watch.Modified
		case "DELETED":
			eventType = watch.Deleted
		case "ERROR":
			eventType = watch.Error
		default:
			w.logger.V(1).Info("Unknown watch event type ignored", "type", resp.Type)
			continue
		}

		obj := devicev1alpha1.FromProto(resp.Object)

		select {
		case w.result <- watch.Event{Type: eventType, Object: obj}:
		case <-w.done:
			return
		}
	}
}
