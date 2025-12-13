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

package transport

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewLatencyUnaryInterceptor returns a gRPC UnaryClientInterceptor that logs
// the latency and status of each unary RPC call. Errors and notable gRPC
// status codes are logged at verbosity level 4 or as errors.
func NewLatencyUnaryInterceptor(logger logr.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(start)

		if logger.V(4).Enabled() || err != nil {
			s := status.Convert(err)
			code := s.Code()

			keysAndValues := []interface{}{
				"method", method,
				"status", code.String(),
				"code", int(code),
				"duration_ms", duration.Milliseconds(),
			}

			if err != nil {
				switch code {
				case codes.Canceled:
					logger.V(4).Info("RPC canceled", keysAndValues...)
				case codes.DeadlineExceeded:
					logger.V(4).Info("RPC timed out", keysAndValues...)
				case codes.Aborted:
					logger.V(4).Info("RPC aborted", keysAndValues...)
				default:
					logger.Error(err, "RPC failed", keysAndValues...)
				}
			} else {
				logger.V(4).Info("RPC succeeded", keysAndValues...)
			}
		}

		return err
	}
}

// NewLatencyStreamInterceptor returns a gRPC StreamClientInterceptor that logs
// the latency and status of establishing each client stream. Errors and notable
// gRPC status codes are logged at verbosity level 4 or as errors.
func NewLatencyStreamInterceptor(logger logr.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		start := time.Now()
		stream, err := streamer(ctx, desc, cc, method, opts...)
		duration := time.Since(start)

		if logger.V(4).Enabled() || err != nil {
			s := status.Convert(err)
			code := s.Code()

			keysAndValues := []interface{}{
				"method", method,
				"status", code.String(),
				"code", int(code),
				"duration_ms", duration.Milliseconds(),
			}

			if err != nil {
				switch code {
				case codes.Canceled:
					logger.V(4).Info("Stream canceled", keysAndValues...)
				case codes.DeadlineExceeded:
					logger.V(4).Info("Stream timed out", keysAndValues...)
				case codes.Aborted:
					logger.V(4).Info("Stream aborted", keysAndValues...)
				default:
					logger.Error(err, "Stream failed", keysAndValues...)
				}
			} else {
				logger.V(4).Info("Stream started", keysAndValues...)
			}
		}
		return stream, err
	}
}
