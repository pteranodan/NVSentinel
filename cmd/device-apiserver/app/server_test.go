//  Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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

package app_test

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app"
	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app/options"
	"github.com/nvidia/nvsentinel/pkg/util/test"
)

func setupTestOptions(t *testing.T, tmpDir string, kineSocketPathOverride string) *options.ServerRunOptions {
	apiSocketPath := filepath.Join(tmpDir, "api.sock")
	dbPath := filepath.Join(tmpDir, "state.db")
	kineSocketPath := filepath.Join(tmpDir, "kine.sock")
	if kineSocketPathOverride != "" {
		kineSocketPath = kineSocketPathOverride
	}
	t.Setenv("KINE_SOCKET_PATH", kineSocketPath)
	opts := options.NewServerRunOptions()

	opts.NodeName = "test-node"
	opts.BindAddress = "unix://" + apiSocketPath
	opts.HealthAddress = test.MustGetFreeTCPAddress(t)
	opts.ShutdownGracePeriod = 1 * time.Second

	opts.Storage.Endpoint = "sqlite://" + dbPath
	opts.Storage.InitializationTimeout = 5 * time.Second
	opts.Storage.ReadycheckTimeout = 2 * time.Second

	return opts
}

func TestRun(t *testing.T) {
	tmpDir := t.TempDir()
	opts := setupTestOptions(t, tmpDir, "")
	apiSocketPath := strings.TrimPrefix(opts.BindAddress, "unix://")

	completedOpts, err := opts.Complete()
	if err != nil {
		t.Fatalf("Failed to complete options: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(ctx, completedOpts)
	}()

	test.WaitForStatus(t, opts.HealthAddress, "", 10*time.Second, test.IsServing)

	cancel()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Server exited with unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server failed to shut down within grace period")
	}

	if _, err := os.Stat(apiSocketPath); err == nil {
		t.Errorf("API socket file %q still exists after server shutdown", apiSocketPath)
	}
}

func TestRun_SignalHandling(t *testing.T) {
	tmpDir := t.TempDir()
	opts := setupTestOptions(t, tmpDir, "")
	opts.ShutdownGracePeriod = 100 * time.Millisecond

	ctx, stop := context.WithCancel(context.Background())

	completedOpts, err := opts.Complete()
	if err != nil {
		t.Fatalf("Failed to complete options: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Run(ctx, completedOpts)
	}()

	test.WaitForStatus(t, opts.HealthAddress, "", 5*time.Second, test.IsServing)

	stop()
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("Server exited with unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Server failed to shut down within grace period")
	}
}

func TestRun_SocketConflict(t *testing.T) {
	tmpDir := t.TempDir()
	opts := setupTestOptions(t, tmpDir, "")
	apiSocketPath := strings.TrimPrefix(opts.BindAddress, "unix://")

	completedOpts, err := opts.Complete()
	if err != nil {
		t.Fatalf("Failed to complete options: %v", err)
	}

	l, err := net.Listen("unix", apiSocketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	err = app.Run(context.Background(), completedOpts)

	if err == nil {
		t.Fatal("Expected server to fail due to socket conflict")
	}

	if !strings.Contains(err.Error(), "already in use") {
		t.Errorf("Expected 'already in use' error, got: %v", err)
	}
}

func TestRun_StorageInitializationFailure(t *testing.T) {
	t.Run("Database", func(t *testing.T) {
		tmpDir := t.TempDir()

		blockedPath := filepath.Join(tmpDir, "blocked")
		if err := os.WriteFile(blockedPath, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		opts := setupTestOptions(t, tmpDir, "")
		opts.Storage.Endpoint = "sqlite://" + filepath.Join(blockedPath, "state.db")
		opts.Storage.InitializationTimeout = 1 * time.Second
		opts.Storage.ReadycheckTimeout = 1 * time.Second

		completedOpts, err := opts.Complete()
		if err != nil {
			t.Fatalf("Failed to complete options: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err = app.Run(ctx, completedOpts)

		if err == nil {
			t.Fatal("Expected server to fail due to storage (database) initialization error")
		}

		if !strings.Contains(err.Error(), "storage data") || !strings.Contains(err.Error(), "not a directory") {
			t.Errorf("expected 'storage data...not a directory' error, got: %v", err)
		}
	})

	t.Run("Kine", func(t *testing.T) {
		tmpDir := t.TempDir()

		blockedPath := filepath.Join(tmpDir, "blocked")
		if err := os.WriteFile(blockedPath, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		opts := setupTestOptions(t, tmpDir, filepath.Join(blockedPath, "kine.sock"))

		completedOpts, err := opts.Complete()
		if err != nil {
			t.Fatalf("Failed to complete options: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err = app.Run(ctx, completedOpts)

		if err == nil {
			t.Fatal("Expected server to fail due to storage (kine) initialization error")
		}

		if !strings.Contains(err.Error(), "storage socket") || !strings.Contains(err.Error(), "not a directory") {
			t.Errorf("Expected 'storage socket...not a directory' error, got: %v", err)
		}
	})
}
