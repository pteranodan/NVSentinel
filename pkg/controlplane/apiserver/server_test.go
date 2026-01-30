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

package apiserver

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/options"
	"github.com/nvidia/nvsentinel/pkg/storage/storagebackend"
	"github.com/nvidia/nvsentinel/pkg/util/testutils"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestDeviceAPIServer(t *testing.T) {
	socketPath := testutils.NewUnixAddr(t, "test.sock")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := options.NewOptions()
	opts.GRPC.BindAddress = "unix://" + socketPath
	opts.HealthAddress = "127.0.0.1:0"
	opts.MetricsAddress = "127.0.0.1:0"

	completedOpts, _ := opts.Complete(ctx)
	config, err := NewConfig(ctx, completedOpts)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	completedCfg, err := config.Complete()
	if err != nil {
		t.Fatalf("failed to complete config: %v", err)
	}

	sm := &storagebackend.StorageManager{}

	srv, err := completedCfg.New(sm)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	t.Run("PrepareRegistration", func(t *testing.T) {
		prepared, err := srv.PrepareRun()
		if err != nil {
			t.Fatalf("PrepareRun failed: %v", err)
		}

		if prepared.DeviceAPIServer != srv {
			t.Error("prepared server wrapper does not point to the original server instance")
		}

		if srv.HealthServer == nil {
			t.Fatal("HealthServer was not initialized")
		}

		resp, err := srv.HealthServer.Check(context.Background(), &healthpb.HealthCheckRequest{Service: ""})
		if err != nil {
			t.Fatalf("health check failed: %v", err)
		}
		if resp.Status != healthpb.HealthCheckResponse_NOT_SERVING {
			t.Errorf("expected initial status NOT_SERVING, got %v", resp.Status)
		}
	})

	t.Run("CreateListener", func(t *testing.T) {
		socketPath := testutils.NewUnixAddr(t, "lis.sock")

		lis, err := srv.createUDSListener(context.Background(), socketPath)
		if err != nil {
			t.Fatalf("failed to create listener: %v", err)
		}
		defer lis.Close()

		if _, err := os.Stat(socketPath); err != nil {
			t.Errorf("socket file %q was not created: %v", socketPath, err)
		}
	})

	t.Run("RunAndShutdown", func(t *testing.T) {
		localSocket := testutils.NewUnixAddr(t, "run.sock")
		srv.BindAddress = "unix://" + localSocket
		srv.HealthAddress = testutils.GetFreeTCPAddress(t)

		ready := make(chan struct{})
		srv.StorageManager.TestOnlySetReadyChan(ready)

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.run(ctx)
		}()

		close(ready)

		testutils.WaitForRunning(t, srv.HealthAddress, 2*time.Second)

		cancel()

		select {
		case err := <-errCh:
			if err != nil && err != context.Canceled {
				t.Errorf("server exited with unexpected error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("server failed to shut down within grace period")
		}

		if _, err := os.Stat(localSocket); !os.IsNotExist(err) {
			t.Error("socket file was not removed after shutdown")
		}
	})
}
