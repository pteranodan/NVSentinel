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
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/api"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/metrics"
	"github.com/nvidia/nvsentinel/pkg/storage"
	"github.com/nvidia/nvsentinel/pkg/storage/storagebackend"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/admin"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"
)

type DeviceAPIServer struct {
	NodeName            string
	BindAddress         string
	HealthAddress       string
	MetricsAddress      string
	ShutdownGracePeriod time.Duration
	APIGroups           []*api.GroupInfo

	DeviceServer    *grpc.Server
	HealthServer    *health.Server
	AdminServer     *grpc.Server
	Metrics         *grpcprom.ServerMetrics
	MetricsRegistry *prometheus.Registry
	StorageManager  *storagebackend.StorageManager
	StorageFactory  storage.StorageFactory
}

func (c *CompletedConfig) New(s *storagebackend.StorageManager) (*DeviceAPIServer, error) {
	klog.V(4).InfoS("Creating new Device Server",
		"node", c.NodeName,
		"bind", c.BindAddress,
		"serverOptionsCount", len(c.ServerOptions))

	deviceSrv := grpc.NewServer(c.ServerOptions...)

	adminSrv := grpc.NewServer()

	return &DeviceAPIServer{
		NodeName:            c.NodeName,
		BindAddress:         c.BindAddress,
		HealthAddress:       c.HealthAddress,
		MetricsAddress:      c.MetricsAddress,
		ShutdownGracePeriod: c.ShutdownGracePeriod,
		APIGroups:           c.APIGroups,
		DeviceServer:        deviceSrv,
		AdminServer:         adminSrv,
		Metrics:             c.ServerMetrics,
		MetricsRegistry:     c.ServerMetricsRegistry,
		StorageManager:      s,
		StorageFactory:      c.StorageFactory,
	}, nil
}

type preparedDeviceAPIServer struct {
	*DeviceAPIServer
}

func (s *DeviceAPIServer) PrepareRun() (preparedDeviceAPIServer, error) {
	if s.HealthAddress != "" {
		s.HealthServer = health.NewServer()
		healthpb.RegisterHealthServer(s.AdminServer, s.HealthServer)
		s.HealthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}

	reflection.Register(s.DeviceServer)
	reflection.Register(s.AdminServer)

	admin.Register(s.AdminServer)

	if s.Metrics != nil {
		s.Metrics.InitializeMetrics(s.DeviceServer)
		s.Metrics.InitializeMetrics(s.AdminServer)
	}

	klog.V(3).InfoS("gRPC services registered",
		"node", s.NodeName,
		"health", s.HealthAddress != "",
		"reflection", true,
		"admin/channelz", true,
		"metrics", s.Metrics != nil)

	return preparedDeviceAPIServer{s}, nil
}

func (s *preparedDeviceAPIServer) Run(ctx context.Context) error {
	return s.DeviceAPIServer.run(ctx)
}

func (s *DeviceAPIServer) run(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	socketPath := strings.TrimPrefix(s.BindAddress, "unix://")
	lis, err := s.createUDSListener(ctx, socketPath)
	if err != nil {
		return err
	}
	defer lis.Close()

	if s.HealthAddress != "" {
		go s.serveHealth(ctx)
	}
	if s.MetricsAddress != "" {
		go s.serveMetrics(ctx)
	}
	go s.handleShutdown(ctx, socketPath)

	if err := s.waitForStorage(ctx); err != nil {
		return fmt.Errorf("failed to wait for storage readiness: %w", err)
	}

	if err := s.installAPIGroups(ctx); err != nil {
		return err
	}

	if s.HealthServer != nil {
		s.HealthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
		logger.V(2).Info("gRPC serving status updated", "status", healthpb.HealthCheckResponse_SERVING.String())
	}

	logger.Info("Starting Device API Server", "address", s.BindAddress)

	return s.DeviceServer.Serve(lis)
}

func (s *DeviceAPIServer) handleShutdown(ctx context.Context, socketPath string) {
	logger := klog.FromContext(ctx)

	<-ctx.Done()

	logger.V(2).Info("Received termination signal, starting graceful shutdown")

	if s.HealthServer != nil {
		s.HealthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		logger.V(2).Info("gRPC serving status updated", "status", healthpb.HealthCheckResponse_NOT_SERVING.String())
	}

	logger.Info("Shutting down servers",
		"address", s.BindAddress,
		"adminAddress", s.HealthAddress,
		"gracePeriod", s.ShutdownGracePeriod)

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.ShutdownGracePeriod)
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.DeviceServer.GracefulStop()
		if s.AdminServer != nil {
			s.AdminServer.GracefulStop()
		}
		close(done)
	}()

	select {
	case <-done:
		logger.V(2).Info("gRPC servers stopped gracefully")

	case <-shutdownCtx.Done():
		logger.Info("Graceful shutdown timed out; forcing stop", "timeout", s.ShutdownGracePeriod)
		s.DeviceServer.Stop()
		s.AdminServer.Stop()
	}

	logger.V(2).Info("gRPC servers stopped successfully")

	if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
		logger.Error(err, "Failed to remove socket file during shutdown", "path", socketPath)
	}
}

func (s *DeviceAPIServer) createUDSListener(ctx context.Context, socketPath string) (net.Listener, error) {
	logger := klog.FromContext(ctx)

	socketDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create socket directory %q: %w", socketDir, err)
	}

	if _, err := os.Stat(socketPath); err == nil {
		d := net.Dialer{Timeout: 100 * time.Millisecond}
		conn, dialErr := d.DialContext(ctx, "unix", socketPath)

		if dialErr == nil {
			conn.Close()
			return nil, fmt.Errorf("socket %q is already in use", socketPath)
		}

		logger.V(2).Info("Removing stale socket file", "path", socketPath)
		if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to remove stale socket %q: %w", socketPath, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to stat socket %q: %w", socketPath, err)
	}

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on unix socket %q: %w", socketPath, err)
	}

	if err := os.Chmod(socketPath, 0660); err != nil {
		lis.Close()
		return nil, fmt.Errorf("failed to secure socket %q: %w", socketPath, err)
	}

	return lis, nil
}

func (s *DeviceAPIServer) serveHealth(ctx context.Context) {
	logger := klog.FromContext(ctx)

	lis, err := net.Listen("tcp", s.HealthAddress)
	if err != nil {
		logger.Error(err, "Failed to listen on health port", "address", s.HealthAddress)
		return
	}

	logger.V(2).Info("Starting health server", "address", s.HealthAddress)
	if err := s.AdminServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		logger.Error(err, "Health server stopped unexpectedly")
	}
}

func (s *DeviceAPIServer) serveMetrics(ctx context.Context) {
	logger := klog.FromContext(ctx)

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,            // System, Go, Kine
		metrics.DefaultServerMetrics.Registry, // Device API gRPC
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{}))

	metricsSrv := &http.Server{
		Addr:    s.MetricsAddress,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		logger.V(2).Info("Shutting down metrics server", "protocol", "HTTP", "address", s.MetricsAddress)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.ShutdownGracePeriod)
		defer cancel()

		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error(err, "Metrics server graceful shutdown failed; forcing close", "protocol", "HTTP", "address", s.MetricsAddress)
			metricsSrv.Close()
		}
	}()

	logger.V(2).Info("Starting metrics server", "protocol", "HTTP", "address", s.MetricsAddress)
	if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error(err, "Metrics server failed to listen and serve", "protocol", "HTTP", "address", s.MetricsAddress)
	}
}

func (s *DeviceAPIServer) waitForStorage(ctx context.Context) error {
	if s.StorageManager == nil {
		return fmt.Errorf("internal error: storage backend not initialized")
	}

	logger := klog.FromContext(ctx)
	startTime := time.Now()

	if s.StorageManager.IsReady() {
		return nil
	}

	heartbeat := time.NewTicker(5 * time.Second)
	defer heartbeat.Stop()

	const msg = "Waiting for storage backend to initialize"
	logger.Info(msg)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-s.StorageManager.Ready():
			logger.V(2).Info("Storage backend is ready", "duration", time.Since(startTime).Round(time.Second))
			return nil

		case <-heartbeat.C:
			logger.Info(msg, "elapsed", time.Since(startTime).Round(time.Second))
		}
	}
}

func (s *DeviceAPIServer) installAPIGroups(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	for _, info := range s.APIGroups {
		for version, installFn := range info.VersionedInstallers {
			logger.Info("Installing gRPC service",
				"group", info.GroupName,
				"version", version)

			fullServiceName := fmt.Sprintf("%s/%s", info.GroupName, version)

			if err := installFn(s.StorageFactory, s.NodeName, s.DeviceServer); err != nil {
				return fmt.Errorf("failed to install %s: %w", fullServiceName, err)
			}

			if s.HealthServer != nil {
				s.HealthServer.SetServingStatus(fullServiceName, healthpb.HealthCheckResponse_SERVING)
				s.HealthServer.SetServingStatus(info.GroupName, healthpb.HealthCheckResponse_SERVING)
			}
		}
	}
	return nil
}
