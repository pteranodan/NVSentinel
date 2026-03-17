//  Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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

// Package main provides a mock NVIDIA Device API server for demonstration purposes.
//
// It initializes an in-memory API server and simulates synthetic GPU resource activity.
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app"
	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app/options"
	"github.com/nvidia/nvsentinel/pkg/client-go/clientset/versioned"
	"github.com/nvidia/nvsentinel/pkg/grpc/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func main() {
	tmpDir, err := os.MkdirTemp("/tmp", "nvsz")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v\n", err)
	}

	apiSocket := filepath.Join(tmpDir, "api.sock")

	opts := options.NewServerRunOptions()
	opts.NodeName = "example-node"
	opts.BindAddress = "unix://" + apiSocket
	opts.Storage.Type = "memory"

	completed, err := opts.Complete()
	if err != nil {
		os.RemoveAll(tmpDir)
		log.Fatalf("Failed to complete server options: %v\n", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("----------------------------------------------------------------")
	fmt.Printf("NVIDIA Device API Server\n")
	fmt.Printf("To connect, run:\n\n")
	fmt.Printf("  export NVIDIA_DEVICE_API_SOCK=%s\n\n", completed.BindAddress)
	fmt.Println("Press Ctrl+C to stop.")
	fmt.Println("----------------------------------------------------------------")

	go func() {
		if err := app.Run(ctx, completed); err != nil {
			log.Fatalf("Server exited with error: %v\n", err)
		}
	}()

	time.Sleep(500 * time.Millisecond)
	config := &client.Config{Target: completed.BindAddress}
	clientset, err := versioned.NewForConfig(ctx, config)
	if err != nil {
		stop()
		os.RemoveAll(tmpDir)
		log.Fatalf("Failed to create clientset: %v\n", err)
	}

	simulateActivity(ctx, clientset)

	<-ctx.Done()

	os.RemoveAll(tmpDir)
}

func simulateActivity(ctx context.Context, cs *versioned.Clientset) {
	for i := 0; i < 8; i++ {
		gpu := &devicev1alpha1.GPU{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("gpu-%d", i), Namespace: "default"},
			Spec:       devicev1alpha1.GPUSpec{UUID: generateUUID()},
		}

		created, err := cs.DeviceV1alpha1().GPUs().Create(ctx, gpu, metav1.CreateOptions{})
		if err != nil {
			log.Fatalf("Failed to create %s: %v", gpu.Name, err)
		}

		created.Status.Conditions = []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "DriverReady",
				Message:            "Driver is posting ready status",
				LastTransitionTime: metav1.Now(),
			},
		}

		_, err = cs.DeviceV1alpha1().GPUs().UpdateStatus(ctx, created, metav1.UpdateOptions{})
		if err != nil {
			log.Fatalf("Failed to initialize status for %s: %v", gpu.Name, err)
		}
	}

	go periodicStatusUpdates(ctx, cs)
}

func periodicStatusUpdates(ctx context.Context, cs versioned.Interface) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			list, err := cs.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})
			if err != nil || len(list.Items) == 0 {
				continue
			}

			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(list.Items))))
			gpu := &list.Items[n.Int64()]

			current := gpu.Status.Conditions[0].Status
			if current == metav1.ConditionTrue {
				gpu.Status.Conditions[0].Status = metav1.ConditionFalse
				gpu.Status.Conditions[0].Reason = "DriverNotReady"
			} else {
				gpu.Status.Conditions[0].Status = metav1.ConditionTrue
				gpu.Status.Conditions[0].Reason = "DriverReady"
			}

			_, err = cs.DeviceV1alpha1().GPUs().UpdateStatus(ctx, gpu, metav1.UpdateOptions{})
			if err != nil {
				log.Printf("WARN: failed to update status %s: %v", gpu.Name, err)
			}
		}
	}
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("GPU-%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
