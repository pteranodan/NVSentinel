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
	"net"
	"os"
	"path/filepath"
	"testing"
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
