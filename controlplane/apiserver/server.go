package apiserver

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"k8s.io/klog/v2"
)

type DeviceAPIServer struct {
	NodeName    string
	BindAddress string
	GRPCServer  *grpc.Server
}

func (c *CompletedConfig) New() (*DeviceAPIServer, error) {
	klog.V(4).InfoS("Creating new Device APIServer",
		"node", c.NodeName,
		"bind", c.BindAddress,
		"serverOptionsCount", len(c.ServerOptions),
	)
	grpcServer := grpc.NewServer(c.ServerOptions...)

	RegisterServices(grpcServer, c)

	return &DeviceAPIServer{
		NodeName:    c.NodeName,
		BindAddress: c.BindAddress,
		GRPCServer:  grpcServer,
	}, nil
}

type preparedDeviceAPIServer struct {
	*DeviceAPIServer
}

func (s *DeviceAPIServer) PrepareRun() preparedDeviceAPIServer {
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s.GRPCServer, healthServer)

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	reflection.Register(s.GRPCServer)

	// TODO(dhuenecke): Install internal metrics or statusz logic here

	klog.V(3).InfoS("gRPC services registered",
		"node", s.NodeName,
		"health", true,
		"reflection", true,
	)

	return preparedDeviceAPIServer{s}
}

func (s *preparedDeviceAPIServer) Run(ctx context.Context) error {
	return s.DeviceAPIServer.run(ctx)
}

func (s *DeviceAPIServer) run(ctx context.Context) error {
	logger := klog.FromContext(ctx)

	socketPath := strings.TrimPrefix(s.BindAddress, "unix://")
	socketDir := filepath.Dir(socketPath)

	if err := os.MkdirAll(socketDir, 0750); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	if _, err := os.Stat(socketPath); err == nil {
		logger.V(2).Info("Removing stale socket file", "path", socketPath)
		if err := os.Remove(socketPath); err != nil {
			return fmt.Errorf("failed to remove stale socket: %w", err)
		}
	}

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", socketPath, err)
	}

	if err := os.Chmod(socketPath, 0660); err != nil {
		return fmt.Errorf("failed to secure socket: %w", err)
	}
	logger.V(2).Info("Unix socket permissions set", "path", socketPath, "mode", "0660")

	go func() {
		<-ctx.Done()
		logger.Info("Shutting down Device API Server gracefully", "address", s.BindAddress)
		s.GRPCServer.GracefulStop()
		if err := os.Remove(socketPath); err != nil {
			logger.Error(err, "failed to remove socket file during shutdown", "path", socketPath)
		}
	}()

	logger.Info("Starting Device API Server", "address", s.BindAddress)
	return s.GRPCServer.Serve(lis)
}
