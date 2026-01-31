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

package storagebackend

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/nvidia/nvsentinel/pkg/storage/storagebackend/options"
)

func TestPrepareFilesystem(t *testing.T) {
	tmpDir := t.TempDir()
	dbDir := filepath.Join(tmpDir, "db")
	socketPath := filepath.Join(tmpDir, "socket", "kine.sock")

	s := &StorageManager{
		DatabaseDir:    dbDir,
		KineSocketPath: socketPath,
	}

	t.Run("CreateDirectories", func(t *testing.T) {
		err := s.prepareFilesystem(context.Background())
		if err != nil {
			t.Fatalf("Failed to prepare filesystem: %v", err)
		}

		if _, err := os.Stat(dbDir); os.IsNotExist(err) {
			t.Error("Database directory was not created")
		}
	})

	t.Run("CleanupStaleSocket", func(t *testing.T) {
		if err := os.WriteFile(socketPath, []byte("stale"), 0600); err != nil {
			t.Fatal(err)
		}

		err := s.prepareFilesystem(context.Background())
		if err != nil {
			t.Errorf("Should handle stale socket file, got err: %v", err)
		}

		if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
			t.Error("Stale socket file was not removed")
		}
	})

	t.Run("ErrorIfSocketInUse", func(t *testing.T) {
		os.MkdirAll(filepath.Dir(socketPath), 0750)

		l, err := net.Listen("unix", socketPath)
		if err != nil {
			t.Fatal(err)
		}
		defer l.Close()

		err = s.prepareFilesystem(context.Background())
		if err == nil {
			t.Error("Expected error when socket is in use, got nil")
		}
	})
}

func TestStorageReadyState(t *testing.T) {
	s := &StorageManager{
		readyChan: make(chan struct{}),
	}

	if s.IsReady() {
		t.Error("IsReady() should be false initially")
	}

	// Close channel to simulate Run completion
	close(s.readyChan)

	if !s.IsReady() {
		t.Error("IsReady() should be true after readyChan is closed")
	}

	select {
	case <-s.Ready():
	default:
		t.Error("Ready() channel did not fire")
	}
}

func TestRun(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "state.db")
	socketPath := filepath.Join(tmpDir, "kine.sock")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	opts := options.NewOptions()
	opts.DatabaseDir = tmpDir
	opts.KineSocketPath = socketPath
	opts.KineConfig = endpoint.Config{
		Listener: "unix://" + socketPath,
		Endpoint: fmt.Sprintf("sqlite://%s?_journal=WAL", dbPath),
	}

	completedOpts, _ := opts.Complete()
	config, err := NewConfig(ctx, completedOpts)
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	completedCfg, err := config.Complete()
	if err != nil {
		t.Fatalf("failed to complete config: %v", err)
	}

	sm, err := completedCfg.New()
	if err != nil {
		t.Fatalf("failed to create storage manager: %v", err)
	}

	if err := sm.prepareFilesystem(ctx); err != nil {
		t.Fatalf("failed to prepare filesystem: %v", err)
	}

	t.Run("RunAndCleanup", func(t *testing.T) {
		runCtx, runCancel := context.WithCancel(ctx)

		errCh := make(chan error, 1)
		go func() {
			errCh <- sm.run(runCtx)
		}()

		select {
		case <-sm.Ready():
			if _, err := os.Stat(socketPath); err != nil {
				t.Errorf("socket file should exist while running: %v", err)
			}
		case err := <-errCh:
			t.Fatalf("StorageManager exited prematurely: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("StorageManager timed out waiting to become ready")
		}

		runCancel()

		select {
		case err := <-errCh:
			if err != nil && err != context.Canceled {
				t.Errorf("storage exited with unexpected error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("StorageManager failed to shut down within grace period")
		}

		if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
			t.Errorf("socket file %q was not removed after shutdown", socketPath)
		}
	})
}
