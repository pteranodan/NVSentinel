// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientset

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"github.com/nvidia/nvsentinel/client/config"
	"github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const defaultRPCTimeout = 30 * time.Second

func setupFakeServer(t *testing.T) (socketPath string, cleanup func()) {
	dir := t.TempDir()
	socketPath = filepath.Join(dir, "gpu.sock")

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	fakeServer := newFakeGpuServer()
	pb.RegisterGpuServiceServer(grpcServer, fakeServer)

	go grpcServer.Serve(lis)

	cleanup = func() {
		grpcServer.Stop()
		lis.Close()
	}
	return
}

func TestClientset_Constructors(t *testing.T) {
	g := gomega.NewWithT(t)
	socketPath, cleanup := setupFakeServer(t)
	defer cleanup()

	t.Run("NewForConfig succeeds with valid target", func(t *testing.T) {
		cfg := &config.Config{Target: "unix://" + socketPath}
		cs, err := NewForConfig(cfg)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(cs).ToNot(gomega.BeNil())
		t.Cleanup(func() { cs.Close() })
	})

	t.Run("NewForConfigOrDie panics on empty target", func(t *testing.T) {
		g.Expect(func() { NewForConfigOrDie(&config.Config{}) }).To(gomega.Panic())
	})

	t.Run("invalid durations cause error", func(t *testing.T) {
		tests := []struct {
			name string
			cfg  *config.Config
		}{
			{"negative KeepAliveTime", &config.Config{Target: "unix://" + socketPath, KeepAliveTime: -1 * time.Second}},
			{"negative KeepAliveTimeout", &config.Config{Target: "unix://" + socketPath, KeepAliveTimeout: -1 * time.Second}},
			{"negative IdleTimeout", &config.Config{Target: "unix://" + socketPath, IdleTimeout: -1 * time.Second}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				g.Expect(func() { NewForConfigOrDie(tt.cfg) }).To(gomega.Panic())
			})
		}
	})

	t.Run("zero durations default correctly", func(t *testing.T) {
		cfg := &config.Config{Target: "unix://" + socketPath}
		cs, err := NewForConfig(cfg)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(cs.config.KeepAliveTime).To(gomega.Equal(config.DefaultKeepAliveTime))
		g.Expect(cs.config.KeepAliveTimeout).To(gomega.Equal(config.DefaultKeepAliveTimeout))
		g.Expect(cs.config.IdleTimeout).To(gomega.Equal(config.DefaultIdleTimeout))
	})
}

func TestClientset_GPUs(t *testing.T) {
	socketPath, cleanup := setupFakeServer(t)
	defer cleanup()

	cfg := &config.Config{Target: "unix://" + socketPath}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		}),
	}
	conn, err := grpc.NewClient(cfg.Target, opts...)
	if err != nil {
		t.Fatalf("failed to dial target %s: %v", cfg.Target, err)
	}
	t.Cleanup(func() { conn.Close() })

	clientset, err := NewForConfigAndClient(cfg, conn)
	if err != nil {
		t.Fatalf("failed to create Clientset: %v", err)
	}

	gpuClient := clientset.Device().V1alpha1().GPUs()

	t.Run("Get returns expected results", func(t *testing.T) {
		// success
		gpu, err := gpuClient.Get(context.Background(), "gpu-1", metav1.GetOptions{})
		if err != nil || gpu.Name != "gpu-1" {
			t.Fatalf("unexpected Get result: %v", err)
		}

		// missing GPU
		_, err = gpuClient.Get(context.Background(), "missing-gpu", metav1.GetOptions{})
		if err == nil {
			t.Fatalf("expected error for missing GPU")
		}

		// canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = gpuClient.Get(ctx, "gpu-1", metav1.GetOptions{})
		if status.Code(err) != codes.Canceled {
			t.Fatalf("expected canceled code, got: %v", status.Code(err))
		}
	})

	t.Run("List returns expected results", func(t *testing.T) {
		// success
		gpuList, err := gpuClient.List(context.Background(), metav1.ListOptions{})
		if err != nil || len(gpuList.Items) != 3 {
			t.Fatalf("unexpected GPU list: %v", err)
		}

		// canceled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = gpuClient.List(ctx, metav1.ListOptions{})
		if status.Code(err) != codes.Canceled {
			t.Fatalf("expected canceled code, got: %v", status.Code(err))
		}
	})

	t.Run("Watch returns expected events", func(t *testing.T) {
		g := gomega.NewWithT(t)
		watcher, err := gpuClient.Watch(context.Background(), metav1.ListOptions{})
		g.Expect(err).ToNot(gomega.HaveOccurred())
		defer watcher.Stop()

		names := map[string]struct{}{}
		for e := range watcher.ResultChan() {
			if gObj, ok := e.Object.(*devicev1alpha1.GPU); ok {
				names[gObj.Name] = struct{}{}
			}
		}

		g.Expect(names).To(gomega.HaveKey("gpu-1"))
		g.Expect(names).To(gomega.HaveKey("gpu-2"))
		g.Expect(names).To(gomega.HaveKey("gpu-3"))
	})
}

// fakeGpuServer implements pb.GpuServiceServer for testing.
type fakeGpuServer struct {
	pb.UnimplementedGpuServiceServer
	gpus    map[string]*pb.Gpu
	lastCtx context.Context
}

func newFakeGpuServer() *fakeGpuServer {
	return &fakeGpuServer{
		gpus: map[string]*pb.Gpu{
			"gpu-1": {Name: "gpu-1", Spec: &pb.GpuSpec{Uuid: "GPU-1"}},
			"gpu-2": {Name: "gpu-2", Spec: &pb.GpuSpec{Uuid: "GPU-2"}},
			"gpu-3": {Name: "gpu-3", Spec: &pb.GpuSpec{Uuid: "GPU-3"}},
		},
	}
}

func (f *fakeGpuServer) GetGpu(ctx context.Context, req *pb.GetGpuRequest) (*pb.GetGpuResponse, error) {
	f.lastCtx = ctx
	gpu, ok := f.gpus[req.Name]
	if !ok {
		return nil, fmt.Errorf("%s not found", req.Name)
	}
	return &pb.GetGpuResponse{Gpu: gpu}, nil
}

func (f *fakeGpuServer) ListGpus(ctx context.Context, req *pb.ListGpusRequest) (*pb.ListGpusResponse, error) {
	f.lastCtx = ctx
	list := pb.GpuList{}
	for _, g := range f.gpus {
		list.Items = append(list.Items, g)
	}
	return &pb.ListGpusResponse{GpuList: &list}, nil
}

func (f *fakeGpuServer) WatchGpus(req *pb.WatchGpusRequest, stream pb.GpuService_WatchGpusServer) error {
	for _, g := range f.gpus {
		if err := stream.Send(&pb.WatchGpusResponse{
			Type:   "ADDED",
			Object: g,
		}); err != nil {
			return err
		}
	}
	return nil
}
