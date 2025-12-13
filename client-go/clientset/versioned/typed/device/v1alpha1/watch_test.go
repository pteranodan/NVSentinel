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
	"io"
	"testing"
	"time"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestStreamWatcher_NormalEvents(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recvCh := make(chan *pb.WatchGpusResponse, 3)
	recvCh <- &pb.WatchGpusResponse{Type: "ADDED", Object: &pb.Gpu{Name: "gpu1"}}
	recvCh <- &pb.WatchGpusResponse{Type: "MODIFIED", Object: &pb.Gpu{Name: "gpu1"}}
	recvCh <- &pb.WatchGpusResponse{Type: "DELETED", Object: &pb.Gpu{Name: "gpu1"}}
	close(recvCh)

	stream := NewFakeWatchGpusClient(recvCh, ctx)
	w := newStreamWatcher(stream, cancel, logr.Discard())

	var got []watch.Event
	for e := range w.ResultChan() {
		got = append(got, e)
	}

	wantTypes := []watch.EventType{watch.Added, watch.Modified, watch.Deleted}
	if len(got) != len(wantTypes) {
		t.Fatalf("unexpected event count: got=%d, want=%d", len(got), len(wantTypes))
	}
	for i, ev := range got {
		if ev.Type != wantTypes[i] {
			t.Errorf("event %d type mismatch: got=%v, want=%v", i, ev.Type, wantTypes[i])
		}
		if _, ok := ev.Object.(*devicev1alpha1.GPU); !ok {
			t.Errorf("event %d object is not *GPU, got %T", i, ev.Object)
		}
	}
}

func TestStreamWatcher_GrpcErrors(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		wantReason metav1.StatusReason
		wantCode   int32
	}{
		{"Internal", status.Error(codes.Internal, "internal"), metav1.StatusReasonInternalError, int32(codes.Internal)},
		{"OutOfRange", status.Error(codes.OutOfRange, "out"), metav1.StatusReasonExpired, 410},
		{"DeadlineExceeded", status.Error(codes.DeadlineExceeded, "deadline"), metav1.StatusReasonInternalError, int32(codes.DeadlineExceeded)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream := NewFakeWatchGpusClient(make(chan *pb.WatchGpusResponse), ctx)
			stream.SetErrs([]error{tc.err})
			w := newStreamWatcher(stream, cancel, logr.Discard())

			count := 0
			for e := range w.ResultChan() {
				count++
				if e.Type != watch.Error {
					t.Errorf("expected watch.Error, got %v", e.Type)
				}
				st, ok := e.Object.(*metav1.Status)
				if !ok {
					t.Fatalf("expected *Status, got %T", e.Object)
				}
				if st.Reason != tc.wantReason || st.Code != tc.wantCode {
					t.Errorf("status mismatch: got=%+v, wantReason=%v, wantCode=%d", st, tc.wantReason, tc.wantCode)
				}
			}
			if count != 1 {
				t.Errorf("expected exactly 1 error event, got %d", count)
			}
		})
	}
}

func TestStreamWatcher_CanceledOrEOF(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"Canceled", status.Error(codes.Canceled, "canceled")},
		{"EOF", io.EOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			stream := NewFakeWatchGpusClient(make(chan *pb.WatchGpusResponse), ctx)
			stream.SetErr(tc.err)
			w := newStreamWatcher(stream, cancel, logr.Discard())

			for range w.ResultChan() {
				// drain
			}
		})
	}
}

func TestStreamWatcher_UnknownEventType_Ignored(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recvCh := make(chan *pb.WatchGpusResponse, 2)
	recvCh <- &pb.WatchGpusResponse{Type: "UNKNOWN", Object: &pb.Gpu{Name: "gpu1"}}
	recvCh <- &pb.WatchGpusResponse{Type: "ADDED", Object: &pb.Gpu{Name: "gpu2"}}
	close(recvCh)

	w := newStreamWatcher(NewFakeWatchGpusClient(recvCh, ctx), cancel, logr.Discard())

	var got []watch.Event
	for e := range w.ResultChan() {
		got = append(got, e)
	}

	if len(got) != 1 {
		t.Fatalf("expected only 1 valid event, got %d", len(got))
	}
	if got[0].Type != watch.Added || got[0].Object.(*devicev1alpha1.GPU).Name != "gpu2" {
		t.Errorf("unexpected event: %+v", got[0])
	}
}

func TestStreamWatcher_TrySendError_OnlyOnce(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recvCh := make(chan *pb.WatchGpusResponse)
	defer close(recvCh)

	// Stream will always return Internal errors
	stream := NewFakeWatchGpusClient(recvCh, ctx)
	stream.SetErrs([]error{
		status.Error(codes.Internal, "first error"),
		status.Error(codes.Internal, "second error"),
	})

	w := newStreamWatcher(stream, cancel, logr.Discard())

	count := 0
	for e := range w.ResultChan() {
		if e.Type != watch.Error {
			t.Errorf("expected watch.Error, got %v", e.Type)
		}
		count++
	}

	if count != 1 {
		t.Errorf("expected only 1 error event, got %d", count)
	}
}

func TestStreamWatcher_StopClosesResultChan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	recvCh := make(chan *pb.WatchGpusResponse)
	stream := NewFakeWatchGpusClient(recvCh, ctx)
	w := newStreamWatcher(stream, cancel, logr.Discard())

	w.Stop()

	select {
	case _, ok := <-w.ResultChan():
		if ok {
			t.Error("expected ResultChan to be closed after Stop()")
		}
	case <-time.After(time.Second):
		t.Error("ResultChan not closed after 1s")
	}
}
