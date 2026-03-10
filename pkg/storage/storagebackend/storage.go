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
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	filesystemutils "github.com/nvidia/nvsentinel/pkg/util/filesystem"
	netutils "github.com/nvidia/nvsentinel/pkg/util/net"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/klog/v2"
)

const storageTypeMemory = "memory"

// Storage is a struct that contains a control plane storage backend instance
// that can be run to start the underlying data store that backs the API server.
type Storage struct {
	StorageConfig apistorage.Config
	DatabaseDir   string
	SocketPath    string
	KineConfig    endpoint.Config

	ETCDConfig *endpoint.ETCDConfig
}

type preparedStorage struct {
	// TODO: add comment
	*Storage
}

// New returns a new instance of Storage from the given config.
func (c *CompletedConfig) New() (*Storage, error) {
	return &Storage{
		StorageConfig: c.StorageConfig,
		DatabaseDir:   c.DatabaseDir,
		SocketPath:    c.SocketPath,
		KineConfig:    c.KineConfig,
	}, nil
}

// TODO: add docs
func (s *Storage) PrepareRun() (preparedStorage, error) {
	if s.StorageConfig.Type == storageTypeMemory {
		return preparedStorage{s}, nil
	}

	if err := s.prepareFilesystem(); err != nil {
		return preparedStorage{}, err
	}

	return preparedStorage{s}, nil
}

func (s *Storage) prepareFilesystem() error {
	if err := os.MkdirAll(s.DatabaseDir, 0770); err != nil {
		return fmt.Errorf("failed to create storage data directory: %w", err)
	}
	if err := os.Chmod(s.DatabaseDir, 0770); err != nil {
		return fmt.Errorf("failed to secure storage data directory: %w", err)
	}
	if err := filesystemutils.CheckPermissions(s.DatabaseDir); err != nil {
		return fmt.Errorf("storage data directory %q: %w", s.DatabaseDir, err)
	}

	socketPath := strings.TrimPrefix(s.SocketPath, "unix://")
	socketDir := filepath.Dir(socketPath)

	if err := os.MkdirAll(socketDir, 0770); err != nil {
		return fmt.Errorf("failed to create storage socket directory: %w", err)
	}
	if err := os.Chmod(socketDir, 0770); err != nil {
		return fmt.Errorf("failed to secure storage socket directory: %w", err)
	}
	if err := filesystemutils.CheckPermissions(socketDir); err != nil {
		return fmt.Errorf("storage socket directory %q: %w", socketDir, err)
	}

	_, err := os.Stat(socketPath)
	if err == nil {
		d := net.Dialer{Timeout: 100 * time.Millisecond}
		conn, dialErr := d.DialContext(context.Background(), "unix", socketPath) //nolint:wsl_v5
		if dialErr == nil {
			conn.Close()
			return fmt.Errorf("storage socket %q is already in use", socketPath)
		}

		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove stale storage socket %q: %w", socketPath, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat storage socket %q: %w", socketPath, err)
	}

	return nil
}

// TODO: add docs
func (s *preparedStorage) Run(ctx context.Context) error {
	if s.StorageConfig.Type == storageTypeMemory {
		return s.runInMemory(ctx)
	}

	return s.run(ctx)
}

func (s *Storage) run(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	defer func() {
		if err := netutils.CleanupUDS(s.SocketPath); err != nil {
			klog.V(2).ErrorS(err, "Failed to cleanup storage socket", "path", s.SocketPath)
		}
	}()

	logger.V(2).Info("Starting storage backend", "type", s.StorageConfig.Type)

	etcdConfig, err := endpoint.Listen(ctx, s.KineConfig)
	if err != nil {
		return fmt.Errorf("failed to start storage backend: %w", err)
	}
	s.ETCDConfig = &etcdConfig

	if err := s.waitForSocket(ctx); err != nil {
		return err
	}

	if err := s.waitForEtcd(ctx); err != nil {
		return err
	}

	logger.V(3).Info("Storage backend is ready", "path", s.SocketPath)

	<-ctx.Done()
	logger.Info("Shutting down storage backend")

	return nil
}

func (s *Storage) waitForSocket(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	timeout := s.StorageConfig.HealthcheckTimeout

	logger.V(4).Info("Waiting for storage socket to accept connections", "path", s.SocketPath, "timeout", timeout)

	err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, timeout, true,
		func(ctx context.Context) (bool, error) {
			if _, err := os.Stat(s.SocketPath); err != nil {
				// socket isn't there yet, keep polling
				return false, nil
			}

			d := net.Dialer{Timeout: 100 * time.Millisecond}
			conn, err := d.DialContext(ctx, "unix", s.SocketPath)
			if err != nil {
				// socket isn't accepting yet, keep polling
				return false, nil
			}
			conn.Close()

			if err := os.Chmod(s.SocketPath, 0660); err != nil {
				if os.IsPermission(err) {
					return false, fmt.Errorf("failed to secure storage socket %q: %w", s.SocketPath, err)
				}

				return false, nil
			}

			return true, nil
		},
	)
	if err != nil {
		return fmt.Errorf("timed out waiting %v to connect to storage socket: %w", timeout, err)
	}

	return nil
}

func (s *Storage) waitForEtcd(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	timeout := s.StorageConfig.ReadycheckTimeout

	logger.V(4).Info("Waiting for etcd readinesss", "timeout", timeout)

	cli, err := s.etcdClient()
	if err != nil {
		return fmt.Errorf("failed to create etcd client: %w", err)
	}
	defer cli.Close()

	err = wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, timeout, true,
		func(ctx context.Context) (bool, error) {
			// Avoid blocking the entire polling loop if one request hangs.
			rpcCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()

			_, err := cli.Status(rpcCtx, s.ETCDConfig.Endpoints[0])
			if err == nil {
				return true, nil
			}

			// If Status fails (e.g., missing dbstat extension), fallback to a Get
			_, err = cli.Get(rpcCtx, "/", clientv3.WithLimit(1))
			if err == nil {
				return true, nil
			}

			logger.V(4).Info("etcd not yet ready", "err", err)
			return false, nil // keep polling
		},
	)
	if err != nil {
		return fmt.Errorf("timed out waiting %v for etcd readiness: %w", timeout, err)
	}

	return nil
}

func (s *Storage) etcdClient() (*clientv3.Client, error) {
	tlsConfig, err := s.ETCDConfig.TLSConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get etcd TLS config: %w", err)
	}

	// Use a silent logger for the etcd client to suppress noisy dbstat warnings
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	etcdLogger, err := zapConfig.Build()
	if err != nil {
		etcdLogger = zap.NewNop()
	}

	return clientv3.New(clientv3.Config{
		Endpoints:   s.ETCDConfig.Endpoints,
		DialTimeout: 2 * time.Second,
		TLS:         tlsConfig,
		Logger:      etcdLogger,
		DialOptions: []grpc.DialOption{
			grpc.WithBlock(),
		},
	})
}

func (s *Storage) runInMemory(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	logger.V(2).Info("Starting storage backend", "type", s.StorageConfig.Type, "experimental", true)
	logger.V(2).Info("WARNING: Data will not persist across restarts.")
	logger.V(2).Info("WARNING: Unvalidated storage layer: may not be fully storage.Interface compliant.")
	logger.V(3).Info("Storage backend is ready")

	<-ctx.Done()
	logger.Info("Shutting down storage backend")

	return nil
}
