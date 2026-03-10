// Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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

package framework

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app"
	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app/options"
	"github.com/nvidia/nvsentinel/pkg/client-go/client/versioned"
	"github.com/nvidia/nvsentinel/pkg/grpc/client"
	"github.com/nvidia/nvsentinel/pkg/util/test"
	"google.golang.org/grpc/grpclog"
)

func init() {
	// silence transport-level gRPC logs
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, io.Discard))
}

type TestServerOptions struct {
	StorageBackend      string
	CompactionInterval  time.Duration
	CompactionMinRetain int64
}

func SetupServer(t *testing.T, storageBackend string) (versioned.Interface, func()) {
	return SetupServerWithOptions(t, TestServerOptions{
		StorageBackend:      storageBackend,
		CompactionInterval:  5 * time.Minute,
		CompactionMinRetain: 1000,
	})
}

func SetupServerWithOptions(t *testing.T, opts TestServerOptions) (versioned.Interface, func()) {
	t.Helper()

	serverCtx, serverCancel := context.WithCancel(context.Background())
	tmpDir := t.TempDir()

	socketDir, err := os.MkdirTemp("/tmp", "nvsz")
	if err != nil {
		t.Fatalf("failed to create short socket dir: %v", err)
	}

	healthAddr := test.MustGetFreeTCPAddress(t)
	apiSocket := filepath.Join(socketDir, "a.sock")
	kineSocket := filepath.Join(socketDir, "k.sock")
	t.Setenv("KINE_SOCKET_PATH", kineSocket)

	svrOpts := options.NewServerRunOptions()
	svrOpts.NodeName = "test-node"
	svrOpts.BindAddress = "unix://" + apiSocket
	svrOpts.HealthAddress = healthAddr
	svrOpts.ShutdownGracePeriod = 1 * time.Second

	if opts.StorageBackend == "memory" {
		svrOpts.Storage.StorageBackend = "memory"
	} else {
		dbFile := filepath.Join(tmpDir, "state.db")
		svrOpts.Storage.DatabasePath = "sqlite://" + dbFile
	}

	completed, err := svrOpts.Complete()
	if err != nil {
		os.RemoveAll(socketDir)
		t.Fatalf("Failed to complete options: %v", err)
	}

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- app.Run(serverCtx, completed)
	}()

	test.WaitForStatus(t, healthAddr, "", 10*time.Second, test.IsServing)

	config := &client.Config{Target: "unix://" + apiSocket}
	cs, err := versioned.NewForConfig(serverCtx, config)
	if err != nil {
		serverCancel()
		os.RemoveAll(socketDir)
		t.Fatalf("Failed to create clientset: %v", err)
	}

	teardown := func() {
		serverCancel()
		select {
		case err := <-serverErrCh:
			if err != nil && err != context.Canceled {
				t.Errorf("Server exited with error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("Server timed out during shutdown")
		}
		os.RemoveAll(socketDir)
	}

	return cs, teardown
}
