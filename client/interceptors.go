package client

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func latencyUnaryInterceptor(logger logr.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		startTime := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		duration := time.Since(startTime)

		s := status.Convert(err)
		code := s.Code()

		keysAndValues := []interface{}{
			"method", method,
			"status", code.String(),
			"code", int(code),
			"duration", duration.String(),
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
			logger.V(4).Info("RPC success", keysAndValues...)
		}

		return err
	}
}

func latencyStreamInterceptor(logger logr.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		startTime := time.Now()
		stream, err := streamer(ctx, desc, cc, method, opts...)
		duration := time.Since(startTime)

		s := status.Convert(err)
		code := s.Code()

		keysAndValues := []interface{}{
			"method", method,
			"status", code.String(),
			"code", int(code),
			"duration", duration.String(),
		}

		if err != nil {
			switch code {
			case codes.Canceled:
				logger.V(4).Info("Stream connection canceled", keysAndValues...)
			case codes.DeadlineExceeded:
				logger.V(4).Info("Stream connection timed out", keysAndValues...)
			case codes.Aborted:
				logger.V(4).Info("Stream connection aborted", keysAndValues...)
			default:
				logger.Error(err, "Stream connection failed", keysAndValues...)
			}
		} else {
			logger.V(4).Info("Stream started", keysAndValues...)
		}

		return stream, err
	}
}
