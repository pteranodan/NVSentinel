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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/stdr"
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/client-go/client/versioned"
	"github.com/nvidia/nvsentinel/client-go/nvgrpc"
)

func main() {
	// Initialize a standard logger for transport-level visibility.
	logger := stdr.New(log.New(os.Stdout, "", log.LstdFlags))
	stdr.SetVerbosity(1)

	// NVIDIA_DEVICE_API_TARGET identifies the local gRPC endpoint.
	target := os.Getenv("NVIDIA_DEVICE_API_TARGET")
	if target == "" {
		target = "unix:///tmp/nvidia-device-api.sock"
	}

	// tracingInterceptor injects metadata (x-request-id) into outgoing requests.
	// This enables request tracking across the gRPC boundary.
	tracingInterceptor := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = metadata.AppendToOutgoingContext(ctx, "x-request-id", "nv-trace-123")
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	// watchMonitorInterceptor logs the start of long-lived Watch streams.
	watchMonitorInterceptor := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		logger.Info("starting long-lived watch stream", "method", method)
		return streamer(ctx, desc, cc, method, opts...)
	}

	// Configure manual DialOptions for transport-level control.
	opts := []nvgrpc.DialOption{
		nvgrpc.WithLogger(logger),
		nvgrpc.WithUnaryInterceptor(tracingInterceptor),
		nvgrpc.WithStreamInterceptor(watchMonitorInterceptor),
	}

	// Initialize the underlying gRPC connection manually.
	config := &nvgrpc.Config{Target: target}
	conn, err := nvgrpc.ClientConnFor(config, opts...)
	if err != nil {
		logger.Error(err, "unable to connect to gRPC target")
		os.Exit(1)
	}
	defer conn.Close()

	// Initialize the Clientset using the existing connection.
	// This is required when specific gRPC lifecycle or interceptor management is needed.
	clientset, err := versioned.NewForConfigAndClient(config, conn)
	if err != nil {
		logger.Error(err, "unable to create clientset")
		os.Exit(1)
	}

	// Use a 30-second context to demonstrate timeout handling.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List GPUs. This triggers the Unary interceptor.
	list, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to list GPUs")
	} else {
		logger.Info("retrieved GPU list", "count", len(list.Items))
	}

	// Watch GPUs. This triggers the Stream interceptor.
	watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to establish watch stream")
	} else {
		defer watcher.Stop()
		logger.Info("watch stream established, waiting for events...")

		for {
			select {
			case event, ok := <-watcher.ResultChan():
				if !ok {
					logger.Info("watch channel closed by server")
					return
				}

				gpu, ok := event.Object.(*devicev1alpha1.GPU)
				if !ok {
					logger.Info("received unknown object type", "type", fmt.Sprintf("%T", event.Object))
					continue
				}

				isReady := meta.IsStatusConditionTrue(gpu.Status.Conditions, "Ready")
				status := "NotReady"
				if isReady {
					status = "Ready"
				}

				logger.Info("gpu status changed",
					"name", gpu.Name,
					"uuid", gpu.Spec.UUID,
					"status", status,
				)

			case <-ctx.Done():
				logger.Info("context timeout reached, stopping watch")
				return
			}
		}
	}
}
