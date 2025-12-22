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

package nvgrpc

import (
	"fmt"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// clientMetrics represents a collection of metrics to be registered on a
// Prometheus metrics registry for a gRPC client.
var clientMetrics = grpcprom.NewClientMetrics(
	grpcprom.WithClientHandlingTimeHistogram(),
)

// init registers the client metrics with the default Prometheus registry.
func init() {
	prometheus.MustRegister(clientMetrics)
}

// ClientConnFor creates a new gRPC connection using the provided configuration and options.
func ClientConnFor(config *Config, opts ...DialOption) (*grpc.ClientConn, error) {
	cfg := *config // Shallow copy to avoid mutation

	dOpts := &dialOptions{}
	for _, opt := range opts {
		opt(dOpts)
	}
	cfg.logger = dOpts.logger

	cfg.Default()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	logger := cfg.GetLogger()

	grpcOpts := []grpc.DialOption{
		grpc.WithUserAgent(cfg.UserAgent),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                DefaultKeepAliveTime,
			Timeout:             DefaultKeepAliveTimeout,
			PermitWithoutStream: true, // Allow keepalive pings even with no active RPCs.
		}),
	}

	// Build the unary interceptor chain.
	unaryInterceptors := []grpc.UnaryClientInterceptor{
		clientMetrics.UnaryClientInterceptor(),
		NewErrorLoggingUnaryInterceptor(logger),
	}
	unaryInterceptors = append(unaryInterceptors, dOpts.unaryInterceptors...)
	grpcOpts = append(grpcOpts, grpc.WithChainUnaryInterceptor(unaryInterceptors...))

	// Build the stream interceptor chain.
	streamInterceptors := []grpc.StreamClientInterceptor{
		clientMetrics.StreamClientInterceptor(),
		NewErrorLoggingStreamInterceptor(logger),
	}
	streamInterceptors = append(streamInterceptors, dOpts.streamInterceptors...)
	grpcOpts = append(grpcOpts, grpc.WithChainStreamInterceptor(streamInterceptors...))

	conn, err := grpc.NewClient(cfg.Target, grpcOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC client for %s: %w", cfg.Target, err)
	}

	return conn, nil
}
