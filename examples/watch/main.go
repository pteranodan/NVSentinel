//  Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

// Package main demonstrates event-driven monitoring and advanced gRPC transport
// configuration, including manual connection lifecycle and interceptor management.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/go-logr/stdr"
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/pkg/client-go/clientset/versioned"
	"github.com/nvidia/nvsentinel/pkg/grpc/client"
)

func main() {
	logger := stdr.New(log.New(os.Stdout, "", log.LstdFlags))
	stdr.SetVerbosity(1)

	// Advanced transport management: Configure custom gRPC interceptors and lifecycle.
	//   Note: manual connection management is required when injecting custom telemetry,
	//   authentication, or tracing middleware into the transport layer.

	// Example: injecting request metadata.
	tracingInterceptor := func(
		ctx context.Context,
		method string,
		req,
		reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", "nv-trace-123")
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	// Example: providing additional visibility into the lifecycle of long-lived streams.
	watchMonitorInterceptor := func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		logger.Info("starting long-lived watch stream", "method", method)
		return streamer(ctx, desc, cc, method, opts...)
	}

	// Define options for transport-level control.
	opts := []client.DialOption{
		client.WithLogger(logger),
		client.WithUnaryInterceptor(tracingInterceptor),
		client.WithStreamInterceptor(watchMonitorInterceptor),
	}

	config := client.GetConfigOrDie()

	// Initialize the gRPC connection manually.
	conn, err := client.ClientConnFor(config, opts...)
	if err != nil {
		logger.Error(err, "unable to connect to gRPC target")
		os.Exit(1)
	}
	defer conn.Close()

	// Initialize a clientset using the existing gRPC connection.
	clientset, err := versioned.NewForConfigAndClient(config, conn)
	if err != nil {
		logger.Error(err, "unable to create clientset")
		os.Exit(1)
	}

	// Trap system signals to ensure graceful shutdown of active gRPC streams.
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Perform an initial List to establish baseline resource state.
	list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to list GPUs")
		os.Exit(1)
	}
	logger.Info("retrieved GPU list", "count", len(list.Items))

	// Establish a Watch stream for real-time GPU resource updates.
	watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to establish watch stream")
		os.Exit(1)
	}

	defer watcher.Stop()

	logger.Info("watch stream established, waiting for events...")

	// Process events from the stream.
	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				logger.Info("watch channel closed by server")
				return
			}

			if event.Type == watch.Error {
				if status, ok := event.Object.(*metav1.Status); ok {
					logger.Info("received watch error from server", "reason", status.Reason, "message", status.Message)
				}

				return
			}

			gpu, ok := event.Object.(*devicev1alpha1.GPU)
			if !ok {
				logger.Info("received unknown object type", "type", fmt.Sprintf("%T", event.Object))
				continue
			}

			// Example: evaluate device readiness.
			isReady := meta.IsStatusConditionTrue(gpu.Status.Conditions, "Ready")
			status := "NotReady"

			if isReady {
				status = "Ready"
			}

			logger.Info("gpu status changed",
				"event", event.Type,
				"name", gpu.Name,
				"uuid", gpu.Spec.UUID,
				"status", status,
			)

		case <-ctx.Done():
			logger.Info("received shutdown signal, stopping watch")
			return
		}
	}
}
