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

package apiserver

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"
	"time"

	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/api"
	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/metrics"
	netutils "github.com/nvidia/nvsentinel/pkg/util/net"
	"github.com/nvidia/nvsentinel/pkg/version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/admin"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/klog/v2"
)

const storageTypeMemory = "memory"

// Server is a struct that contains a control plane device apiserver instance
// that can be run to start serving the APIs.
type Server struct {
	BindAddress          string
	HealthAddress        string
	ServiceMonitorPeriod time.Duration
	MetricsAddress       string
	ShutdownGracePeriod  time.Duration
	PprofAddress         string

	DeviceServer     *grpc.Server
	HealthServer     *health.Server
	AdminServer      *grpc.Server
	AdminCleanup     func()
	Metrics          *metrics.ServerMetrics
	StorageConfig    storagebackend.Config
	ServiceProviders []api.ServiceProvider
	services         []api.Service

	wg sync.WaitGroup
}

// New returns a new instance of Server from the given config.
func (c *CompletedConfig) New() (*Server, error) {
	klog.V(4).InfoS("Creating Device API Server", "bind", c.BindAddress, "serverOptionsCount", len(c.ServerOptions))

	deviceSrv := grpc.NewServer(c.ServerOptions...)

	adminSrv := grpc.NewServer()

	return &Server{
		BindAddress:          c.BindAddress,
		HealthAddress:        c.HealthAddress,
		ServiceMonitorPeriod: c.ServiceMonitorPeriod,
		MetricsAddress:       c.MetricsAddress,
		ShutdownGracePeriod:  c.ShutdownGracePeriod,
		PprofAddress:         c.PprofAddress,
		DeviceServer:         deviceSrv,
		AdminServer:          adminSrv,
		Metrics:              c.ServerMetrics,
		StorageConfig:        c.Storage,
		ServiceProviders:     c.ServiceProviders,
	}, nil
}

type preparedServer struct {
	// TODO: add comment
	*Server
}

// TODO: add docs
func (s *Server) PrepareRun() (preparedServer, error) {
	if s.HealthAddress != "" {
		s.HealthServer = health.NewServer()
		healthpb.RegisterHealthServer(s.AdminServer, s.HealthServer)
		healthpb.RegisterHealthServer(s.DeviceServer, s.HealthServer)
		s.HealthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}

	reflection.Register(s.DeviceServer)
	reflection.Register(s.AdminServer)

	adminCleanup, err := admin.Register(s.AdminServer)
	if err != nil {
		return preparedServer{}, fmt.Errorf("failed to register gRPC admin services: %w", err)
	}
	s.AdminCleanup = adminCleanup

	if s.Metrics != nil {
		s.Metrics.InitializeMetrics(s.DeviceServer)
		s.Metrics.InitializeMetrics(s.AdminServer)
	}

	klog.V(3).InfoS("gRPC services registered",
		"health", s.HealthAddress != "",
		"reflection", true,
		"admin/channelz", true,
		"metrics", s.Metrics != nil)

	return preparedServer{s}, nil
}

// TODO: add docs
func (s *preparedServer) Run(ctx context.Context) error {
	return s.run(ctx)
}

func (s *Server) run(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	if err := s.waitForStorageBackend(ctx); err != nil {
		return err
	}

	defer func() {
		for _, svc := range s.services {
			svc.Cleanup()
		}
	}()

	if err := s.installApiServices(ctx); err != nil {
		return err
	}

	if s.HealthServer != nil {
		s.wg.Add(1)
		go func() { defer s.wg.Done(); s.serveHealth(ctx) }()
	}

	if s.MetricsAddress != "" {
		s.wg.Add(1)
		go func() { defer s.wg.Done(); s.serveMetrics(ctx) }()
	}

	if s.PprofAddress != "" {
		s.wg.Add(1)
		go func() { defer s.wg.Done(); s.servePprof(ctx) }()
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		<-ctx.Done()

		logger.V(2).Info("Received termination signal, starting graceful shutdown...")

		if s.HealthServer != nil {
			for _, svc := range s.services {
				s.HealthServer.SetServingStatus(svc.Name(), healthpb.HealthCheckResponse_NOT_SERVING)
			}

			s.HealthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		}

		s.DeviceServer.GracefulStop()

		if s.AdminServer != nil {
			adminDone := make(chan struct{})
			go func() {
				s.AdminServer.GracefulStop()
				close(adminDone)
			}()

			select {
			case <-adminDone:
			case <-time.After(s.ShutdownGracePeriod):
				logger.V(2).Info("Admin server graceful stop timed out, forcing stop")
				s.AdminServer.Stop()
			}
		}

		if s.AdminCleanup != nil {
			s.AdminCleanup()
		}
	}()

	socketPath := strings.TrimPrefix(s.BindAddress, "unix://")
	lis, cleanup, err := netutils.CreateUDSListener(ctx, socketPath, 0666)
	if err != nil {
		return err
	}
	defer cleanup()

	logger.Info("Starting Device API Server", "address", s.BindAddress)

	serveErr := s.DeviceServer.Serve(lis)
	if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) {
		logger.Error(serveErr, "Server exited unexpectedly")
	}

	s.wg.Wait()

	return serveErr
}

func (s *Server) waitForStorageBackend(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	if s.StorageConfig.Type == storageTypeMemory {
		logger.V(2).Info("Storage backend is in-memory; skipping backend readiness checks")
		return nil
	}

	timeout := s.StorageConfig.ReadycheckTimeout

	logger.V(4).Info("Waiting for storage backend readiness", "timeout", timeout)

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

			_, err := cli.Status(rpcCtx, s.StorageConfig.Transport.ServerList[0])
			if err == nil {
				return true, nil
			}

			// If Status fails (e.g., missing dbstat extension), try a Get
			_, err = cli.Get(rpcCtx, "/", clientv3.WithLimit(1))
			if err == nil {
				return true, nil
			}

			logger.V(4).Info("Storage backend not ready yet", "err", err)
			return false, nil // Keep polling
		},
	)
	if err != nil {
		return fmt.Errorf("timed out waiting %v for storage backend readiness: %w", timeout, err)
	}

	return nil
}

func (s *Server) etcdClient() (*clientv3.Client, error) {
	var tlsConfig *tls.Config
	var err error

	if s.StorageConfig.Transport.CertFile != "" {
		tlsInfo := &transport.TLSInfo{
			CertFile:           s.StorageConfig.Transport.CertFile,
			KeyFile:            s.StorageConfig.Transport.KeyFile,
			TrustedCAFile:      s.StorageConfig.Transport.TrustedCAFile,
			InsecureSkipVerify: true,
		}
		tlsConfig, err = tlsInfo.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get etcd TLS config: %w", err)
		}
	}

	// Use a silent logger for the etcd client to suppress noisy dbstat warnings
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	etcdLogger, err := zapConfig.Build()
	if err != nil {
		etcdLogger = zap.NewNop()
	}

	return clientv3.New(clientv3.Config{
		Endpoints:   s.StorageConfig.Transport.ServerList,
		DialTimeout: 2 * time.Second,
		TLS:         tlsConfig,
		Logger:      etcdLogger,
		DialOptions: []grpc.DialOption{
			grpc.WithBlock(),
		},
	})
}

func (s *Server) installApiServices(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	for _, sp := range s.ServiceProviders {
		svc, err := sp.Install(s.DeviceServer, s.StorageConfig)
		if err != nil {
			return fmt.Errorf("failed to install API service: %w", err)
		}

		name := svc.Name()

		if !svc.IsReady() {
			return fmt.Errorf("API service %q installed but failed readiness check", name)
		}

		s.services = append(s.services, svc)

		if s.HealthServer != nil {
			s.HealthServer.SetServingStatus(name, healthpb.HealthCheckResponse_SERVING)
		}

		if s.Metrics != nil {
			s.Metrics.ServiceHealthStatus.WithLabelValues(name).Set(1.0)
		}
	}

	if s.HealthServer != nil {
		s.HealthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	}

	logger.V(2).Info("API services installed and ready", "count", len(s.services))
	return nil
}

func (s *Server) serveHealth(ctx context.Context) {
	logger := klog.FromContext(ctx)

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", s.HealthAddress)
	if err != nil {
		logger.Error(err, "Failed to listen on health port", "address", s.HealthAddress)
		return
	}
	logger.Info("TCP Health listener officially bound", "address", s.HealthAddress)

	// Shutdown listener immediately on cancellation
	// to unblock Serve and reject new conns.
	go func() {
		<-ctx.Done()

		if err := lis.Close(); err != nil {
			logger.Error(err, "Failed to close health listener", "address", s.HealthAddress)
		}
	}()

	logger.V(2).Info("Starting health server", "address", s.HealthAddress)

	serveErr := s.AdminServer.Serve(lis)
	if serveErr != nil && !errors.Is(serveErr, grpc.ErrServerStopped) && !errors.Is(serveErr, net.ErrClosed) {
		logger.Error(serveErr, "Health server stopped unexpectedly")
	}
}

func (s *Server) serveMetrics(ctx context.Context) {
	logger := klog.FromContext(ctx)

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,            // System, Go, Kine
		metrics.DefaultServerMetrics.Registry, // Device API gRPC
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{}))
	mux.Handle("/version", version.Handler())

	metricsSrv := &http.Server{
		Addr:              s.MetricsAddress,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", s.MetricsAddress)
	if err != nil {
		logger.Error(err, "Failed to listen on metrics port", "address", s.MetricsAddress)
		return
	}

	go func() {
		<-ctx.Done()

		logger.V(2).Info("Shutting down metrics server", "address", s.MetricsAddress)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.ShutdownGracePeriod)
		defer cancel()

		if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error(err, "Metrics server graceful shutdown failed; forcing close", "address", s.MetricsAddress)
			metricsSrv.Close()
		}
	}()

	logger.V(2).Info("Starting metrics server", "address", s.MetricsAddress)

	serveErr := metricsSrv.Serve(lis)
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && !errors.Is(serveErr, net.ErrClosed) {
		logger.Error(serveErr, "Metrics server stopped unexpectedly", "address", s.MetricsAddress)
	}
}

func (s *Server) servePprof(ctx context.Context) {
	logger := klog.FromContext(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	pprofSrv := &http.Server{
		Addr:              s.PprofAddress,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", s.PprofAddress)
	if err != nil {
		logger.Error(err, "Failed to listen on pprof port", "address", s.PprofAddress)
		return
	}

	go func() {
		<-ctx.Done()

		logger.V(2).Info("Shutting down pprof server", "address", s.PprofAddress)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.ShutdownGracePeriod)
		defer cancel()

		if err := pprofSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error(err, "Pprof server graceful shutdown failed; forcing close", "address", s.PprofAddress)
			pprofSrv.Close()
		}
	}()

	logger.V(2).Info("Starting pprof server", "address", s.PprofAddress)

	serveErr := pprofSrv.Serve(lis)
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) && !errors.Is(serveErr, net.ErrClosed) {
		logger.Error(serveErr, "Pprof server stopped unexpectedly", "address", s.PprofAddress)
	}
}
