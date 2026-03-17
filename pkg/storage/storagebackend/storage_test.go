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

package storagebackend

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/nvidia/nvsentinel/pkg/util/test"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestStorage(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		socketFile := test.NewUnixAddr(t)
		socketURL := "unix://" + socketFile

		s := &Storage{
			SocketPath: socketFile,
			StorageDir: tmpDir,
			KineConfig: endpoint.Config{
				Listener:         socketURL,
				Endpoint:         "sqlite://" + dbPath,
				CompactBatchSize: 100,
			},
			StorageConfig: apistorage.Config{
				HealthcheckTimeout: 5 * time.Second,
				ReadycheckTimeout:  2 * time.Second,
			},
		}

		runCtx, stop := context.WithCancel(context.Background())
		defer stop()

		ps, err := s.PrepareRun()
		if err != nil {
			t.Fatalf("PrepareRun failed: %v", err)
		}

		runErr := make(chan error, 1)
		go func() {
			runErr <- ps.Run(runCtx)
		}()

		// confirm socket is secure
		err = wait.PollUntilContextTimeout(runCtx, 100*time.Millisecond, 5*time.Second, true, func(ctx context.Context) (bool, error) {
			info, err := os.Stat(socketFile)
			if err != nil {
				return false, nil
			}
			return info.Mode().Perm() == 0660, nil
		})
		if err != nil {
			t.Fatalf("Timed out waiting for socket to reach 0660 permissions: %v", err)
		}

		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{socketURL},
			DialTimeout: 100 * time.Millisecond,
			DialOptions: []grpc.DialOption{
				grpc.WithBlock(),
			},
		})
		if err != nil {
			t.Fatalf("Failed to create etcd client: %v", err)
		}
		defer cli.Close()

		// confirm etcd is accessible
		err = wait.PollUntilContextTimeout(runCtx, 100*time.Millisecond, 2*time.Second, true, func(ctx context.Context) (bool, error) {
			_, err = cli.Status(context.Background(), socketURL)
			if err == nil {
				return true, nil
			}

			// If Status fails (e.g., missing dbstat extension), fallback to a Get
			// *Default macOS/Darwin SQLite build does NOT include dbstat extension
			_, err = cli.Get(context.Background(), "/", clientv3.WithLimit(1))
			if err == nil {
				return true, nil
			}

			// keep polling
			return false, nil
		})
		if err != nil {
			t.Fatal("Timed out waiting for etcd to become ready")
		}

		// confirm etcd-shim + SQLite backend is functional (i.e., writable)
		err = wait.PollUntilContextTimeout(runCtx, 100*time.Millisecond, 2*time.Second, true, func(ctx context.Context) (bool, error) {
			_, err = cli.Grant(runCtx, 5)
			if err == nil {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			t.Fatal("Timed out waiting for etcd lease grant")
		}

		stop()
		select {
		case err := <-runErr:
			if err != nil && err != context.Canceled {
				t.Errorf("Storage exited with unexpected error: %v", err)
			}
		case <-time.After(2 * time.Second): // matching Kine (non-configurable) GracefulStopTimeout
			t.Error("Storage did not shut down gracefully")
		}

		// confirm socket was cleaned up
		if _, err := os.Stat(socketFile); !os.IsNotExist(err) {
			t.Error("Socket file was not cleaned up after shutdown")
		}
	})

	t.Run("In-memory", func(t *testing.T) {
		s := &Storage{
			StorageConfig: apistorage.Config{
				Type: storageTypeMemory,
			},
		}

		runCtx, stop := context.WithCancel(context.Background())
		defer stop()

		ps, err := s.PrepareRun()
		if err != nil {
			t.Fatalf("PrepareRun failed: %v", err)
		}

		runErr := make(chan error, 1)
		go func() {
			runErr <- ps.Run(runCtx)
		}()

		// confirm storage is running without errors
		select {
		case err := <-runErr:
			t.Fatalf("Storage exited prematurely with error: %v", err)
		case <-time.After(100 * time.Millisecond):
		}

		stop()

		select {
		case err := <-runErr:
			if err != nil && !errors.Is(err, context.Canceled) {
				t.Errorf("Storage exited with unexpected error: %v", err)
			}
		case <-time.After(1 * time.Second):
			t.Error("Storage failed to shut down gracefully")
		}
	})
}

func TestStorage_WaitForSocket_Timeout(t *testing.T) {
	socketPath := test.NewUnixAddr(t)
	socketURL := "unix://" + socketPath

	s := &Storage{
		SocketPath: socketURL,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := s.waitForSocket(ctx)

	if err == nil {
		t.Fatal("Expected timeout waiting for socket")
	}

	d := net.Dialer{Timeout: 50 * time.Millisecond}
	_, err = d.Dial("unix", socketPath)
	if err == nil {
		t.Error("Socket should not be dialable after timeout failure")
	}
}

func TestStorage_StaleSocket(t *testing.T) {
	tmpDir := t.TempDir()
	staleSocketFile := filepath.Join(tmpDir, "stale.sock")
	socketURL := "unix://" + staleSocketFile

	if err := os.WriteFile(staleSocketFile, []byte("stale"), 0666); err != nil {
		t.Fatalf("Failed to create %s: %v", staleSocketFile, err)
	}

	s := &Storage{
		SocketPath:    socketURL,
		StorageDir:    tmpDir,
		StorageConfig: apistorage.Config{Type: "etcd3"},
	}

	err := s.prepareFilesystem()
	if err != nil {
		t.Fatalf("Expected no errors if stale socket exists, but got: %v", err)
	}

	if _, err := os.Stat(staleSocketFile); err == nil {
		t.Error("Expected stale socket file to be removed")
	}
}

func TestStorage_SocketInUse(t *testing.T) {
	tmpDir := t.TempDir()
	socketFile := test.NewUnixAddr(t)
	socketURL := "unix://" + socketFile

	l, err := net.Listen("unix", socketFile)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	s := &Storage{
		SocketPath:    socketURL,
		StorageDir:    tmpDir,
		StorageConfig: apistorage.Config{Type: "etcd3"},
	}

	err = s.prepareFilesystem()
	if err == nil {
		t.Fatal("Expected error, but got none")
	}

	expectedMsg := "is already in use"
	if !strings.Contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error containing %q, got: %v", expectedMsg, err)
	}
}
