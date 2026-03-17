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
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/nvidia/nvsentinel/pkg/storage/storagebackend"
	"github.com/nvidia/nvsentinel/pkg/util/test"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/wait"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestServer(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")
		kineSocket := test.NewUnixAddr(t)
		apiSocket := test.NewUnixAddr(t)
		healthAddr := test.MustGetFreeTCPAddress(t)
		metricsAddr := test.MustGetFreeTCPAddress(t)
		pprofAddr := test.MustGetFreeTCPAddress(t)

		storage := &storagebackend.Storage{
			SocketPath: "unix://" + kineSocket,
			StorageDir: tmpDir,
			KineConfig: endpoint.Config{
				Listener:         "unix://" + kineSocket,
				Endpoint:         "sqlite://" + dbPath,
				CompactBatchSize: 100,
			},
			StorageConfig: apistorage.Config{
				Type: apistorage.StorageTypeETCD3,
				Transport: apistorage.TransportConfig{
					ServerList: []string{"unix://" + kineSocket},
				},
				HealthcheckTimeout: 5 * time.Second,
				ReadycheckTimeout:  2 * time.Second,
			},
		}

		s := &Server{
			BindAddress:          "unix://" + apiSocket,
			HealthAddress:        healthAddr,
			ServiceMonitorPeriod: 100 * time.Millisecond,
			MetricsAddress:       metricsAddr,
			ShutdownGracePeriod:  2 * time.Second,
			DeviceServer:         grpc.NewServer(),
			AdminServer:          grpc.NewServer(),
			PprofAddress:         pprofAddr,
			StorageConfig:        storage.StorageConfig,
		}

		ctx, stop := context.WithCancel(context.Background())
		defer stop()

		ps, err := storage.PrepareRun()
		if err != nil {
			t.Fatalf("Failed to prepare storage: %v", err)
		}
		go ps.Run(ctx)

		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   storage.StorageConfig.Transport.ServerList,
			DialTimeout: 2 * time.Second,
			DialOptions: []grpc.DialOption{
				grpc.WithBlock(),
			},
		})
		if err != nil {
			t.Fatalf("Failed to create etcd client: %v", err)
		}
		defer cli.Close()

		// confirm etcd-shim + SQLite backend is functional (i.e., writable)
		err = wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 2*time.Second, true, func(ctx context.Context) (bool, error) {
			_, err = cli.Grant(ctx, 5)
			if err == nil {
				return true, nil
			}

			return false, nil
		})
		if err != nil {
			t.Fatal("Timed out waiting for etcd readiness")
		}

		// Then start API server
		prepared, err := s.PrepareRun()
		if err != nil {
			t.Fatalf("PrepareRun failed: %v", err)
		}

		serverErr := make(chan error, 1)
		go func() {
			serverErr <- prepared.Run(ctx)
		}()

		test.WaitForStatus(t, healthAddr, "", 5*time.Second, test.IsServing)

		stop()

		select {
		case err := <-serverErr:
			if err != nil && err != context.Canceled && !errors.Is(err, grpc.ErrServerStopped) {
				t.Errorf("server exited with error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Error("server failed to shut down within grace period")
		}

		if _, err := os.Stat(apiSocket); !os.IsNotExist(err) {
			t.Error("socket file was not cleaned up")
		}
	})

	t.Run("InMemory", func(t *testing.T) {
		apiSocket := test.NewUnixAddr(t)
		healthAddr := test.MustGetFreeTCPAddress(t)
		metricsAddr := test.MustGetFreeTCPAddress(t)
		pprofAddr := test.MustGetFreeTCPAddress(t)

		storage := &storagebackend.Storage{
			StorageConfig: apistorage.Config{
				Type: "memory",
			},
		}

		s := &Server{
			BindAddress:          "unix://" + apiSocket,
			HealthAddress:        healthAddr,
			ServiceMonitorPeriod: 100 * time.Millisecond,
			MetricsAddress:       metricsAddr,
			ShutdownGracePeriod:  2 * time.Second,
			DeviceServer:         grpc.NewServer(),
			AdminServer:          grpc.NewServer(),
			PprofAddress:         pprofAddr,
			StorageConfig:        storage.StorageConfig,
		}

		ctx, stop := context.WithCancel(context.Background())
		defer stop()

		ps, err := storage.PrepareRun()
		if err != nil {
			t.Fatalf("Failed to prepare storage: %v", err)
		}
		go ps.Run(ctx)

		prepared, err := s.PrepareRun()
		if err != nil {
			t.Fatalf("PrepareRun failed: %v", err)
		}

		serverErr := make(chan error, 1)
		go func() {
			serverErr <- prepared.Run(ctx)
		}()

		test.WaitForStatus(t, healthAddr, "", 5*time.Second, test.IsServing)

		stop()

		select {
		case err := <-serverErr:
			if err != nil && err != context.Canceled && !errors.Is(err, grpc.ErrServerStopped) {
				t.Errorf("server exited with error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("server failed to shut down within grace period")
		}
	})
}
