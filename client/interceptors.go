package client

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func latencyUnaryInterceptor(logger logr.Logger) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{},
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		startTime := time.Now()

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(startTime).Round(time.Millisecond)
		status := status.Convert(err)

		keysAndValues := []interface{}{
			"method", method,
			"status", status.Code().String(),
			"code", int(status.Code()),
			"duration", duration.String(),
		}

		if err != nil {
			logger.Error(err, "RPC failed", keysAndValues...)
		} else {
			logger.V(1).Info("RPC success", keysAndValues...)
		}

		return err
	}
}

func latencyStreamInterceptor(logger logr.Logger) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {

		start := time.Now()

		stream, err := streamer(ctx, desc, cc, method, opts...)

		duration := time.Since(start).Round(time.Millisecond)
		status := status.Convert(err)

		keysAndValues := []interface{}{
			"method", method,
			"status", status.Code().String(),
			"code", int(status.Code()),
			"duration", duration.String(),
		}

		if err != nil {
			logger.Error(err, "Stream failed", keysAndValues...)
		} else {
			logger.V(1).Info("Stream started", keysAndValues...)
		}

		return stream, err
	}
}
