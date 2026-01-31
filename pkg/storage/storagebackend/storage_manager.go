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
	"strings"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	"k8s.io/apimachinery/pkg/util/wait"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/klog/v2"
)

type StorageManager struct {
	KineConfig     endpoint.Config
	KineSocketPath string
	DatabaseDir    string

	StorageConfig apistorage.Config
	ETCDConfig    *endpoint.ETCDConfig

	readyChan chan struct{}
}

type preparedStorage struct {
	*StorageManager
}

func (c *CompletedConfig) New() (*StorageManager, error) {
	return &StorageManager{
		KineConfig:     c.KineConfig,
		KineSocketPath: c.KineSocketPath,
		DatabaseDir:    c.DatabaseDir,
		StorageConfig:  c.StorageConfig,
		readyChan:      make(chan struct{}),
	}, nil
}

func (s *StorageManager) PrepareRun(ctx context.Context) (preparedStorage, error) {
	if err := s.prepareFilesystem(ctx); err != nil {
		return preparedStorage{}, err
	}

	return preparedStorage{s}, nil
}

func (s *StorageManager) prepareFilesystem(ctx context.Context) error {
	if err := os.MkdirAll(s.DatabaseDir, 0750); err != nil {
		return fmt.Errorf("failed to create storage data directory: %w", err)
	}

	socketDir := filepath.Dir(s.KineSocketPath)
	if err := os.MkdirAll(socketDir, 0750); err != nil {
		return fmt.Errorf("failed to create kine socket directory: %w", err)
	}

	_, err := os.Stat(s.KineSocketPath)
	if err == nil {
		d := net.Dialer{Timeout: 100 * time.Millisecond}
		conn, dialErr := d.DialContext(ctx, "unix", s.KineSocketPath)
		if dialErr == nil {
			conn.Close()
			return fmt.Errorf("kine socket %q is already in use", s.KineSocketPath)
		}

		klog.V(2).InfoS("Removing stale kine socket file", "path", s.KineSocketPath)
		if err := os.Remove(s.KineSocketPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove stale kine socket %q: %w", s.KineSocketPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat kine socket %q: %w", s.KineSocketPath, err)
	}

	return nil
}

func (s *preparedStorage) Run(ctx context.Context) error {
	return s.StorageManager.run(ctx)
}

func (s *StorageManager) run(ctx context.Context) error {
	socketPath := strings.TrimPrefix(s.KineSocketPath, "unix://")

	klog.V(2).InfoS("Starting Kine storage endpoint")
	etcdConfig, err := endpoint.Listen(ctx, s.KineConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize storage backend: %w", err)
	}
	s.ETCDConfig = &etcdConfig

	defer s.cleanupSocket(socketPath)

	if err := s.waitForSocket(ctx); err != nil {
		return err
	}
	close(s.readyChan)

	<-ctx.Done()
	klog.V(2).InfoS("Storage backend shutting down")

	return nil
}

func (s *StorageManager) cleanupSocket(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		klog.ErrorS(err, "Failed to remove socket file during shutdown", "path", path)
	}
}

func (s *StorageManager) waitForSocket(ctx context.Context) error {
	klog.V(2).InfoS("Waiting for socket to be ready")

	socketPath := strings.TrimPrefix(s.KineSocketPath, "unix://")
	err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		if _, err := os.Stat(socketPath); err != nil {
			return false, nil
		}

		d := net.Dialer{Timeout: 100 * time.Millisecond}
		conn, err := d.DialContext(ctx, "unix", socketPath)
		if err != nil {
			return false, nil
		}
		conn.Close()

		if err := os.Chmod(socketPath, 0660); err != nil {
			klog.V(4).ErrorS(err, "failed to secure socket, retrying", "path", socketPath)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("timed out waiting for socket to become ready: %w", err)
	}

	klog.V(2).InfoS("StorageManager is ready",
		"endpoints", s.ETCDConfig.Endpoints,
		"leaderElect", s.ETCDConfig.LeaderElect,
	)
	return nil
}

func (s *StorageManager) Ready() <-chan struct{} {
	return s.readyChan
}

func (s *StorageManager) IsReady() bool {
	if s == nil || s.readyChan == nil {
		return false
	}

	select {
	case <-s.readyChan:
		return true
	default:
		return false
	}
}

// TestOnlySetReadyChan is an internal hook for unit testing.
// DO NOT USE in production.
func (s *StorageManager) TestOnlySetReadyChan(c chan struct{}) {
	s.readyChan = c
}
