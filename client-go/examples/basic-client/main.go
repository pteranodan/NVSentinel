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
	"log"
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/go-logr/stdr"
	"github.com/nvidia/nvsentinel/client-go/client/versioned"
	"github.com/nvidia/nvsentinel/client-go/nvgrpc"
)

func main() {
	logger := stdr.New(log.New(os.Stdout, "", log.LstdFlags))

	// NVIDIA_DEVICE_API_TARGET identifies the local gRPC endpoint.
	target := os.Getenv("NVIDIA_DEVICE_API_TARGET")
	if target == "" {
		target = "unix:///tmp/nvidia-device-api.sock"
	}

	// Initialize the versioned clientset using the gRPC transport.
	config := &nvgrpc.Config{Target: target}
	clientset, err := versioned.NewForConfig(config)
	if err != nil {
		logger.Error(err, "unable to create clientset")
		os.Exit(1)
	}

	gpus, err := clientset.DeviceV1alpha1().GPUs().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		logger.Error(err, "failed to list GPUs")
		os.Exit(1)
	}
	logger.Info("discovered GPUs", "count", len(gpus.Items), "target", target)

	for _, gpu := range gpus.Items {
		// Use standard K8s meta helpers to check status conditions safely.
		isReady := meta.IsStatusConditionTrue(gpu.Status.Conditions, "Ready")
		status := "NotReady"
		if isReady {
			status = "Ready"
		}

		logger.Info("gpu status",
			"name", gpu.Name,
			"uuid", gpu.Spec.UUID,
			"status", status,
		)
	}
}
