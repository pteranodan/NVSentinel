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

package v1alpha1

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"google.golang.org/grpc"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// DefaultRPCTimeout is the default timeout applied to gRPC calls if the context has no deadline.
const DefaultRPCTimeout = 30 * time.Second

// GPUInterface provides methods to interact with GPU resources.
type GPUInterface interface {
	// Get retrieves a GPU by name.
	// Returns an error if the GPU does not exist or the request fails.
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*devicev1alpha1.GPU, error)

	// List returns all GPUs.
	// Only ResourceVersion in ListOptions is respected; other filters are ignored.
	List(ctx context.Context, opts metav1.ListOptions) (*devicev1alpha1.GPUList, error)

	// Watch starts a watch for GPU events beginning from the provided ResourceVersion.
	// The returned watch.Interface delivers Added, Modified, and Deleted events.
	// Canceling the context stops the watch.
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)

	// Informer returns a SharedIndexInformer for GPU resources with default options.
	// The informer is lazily initialized and cached.
	//Informer() cache.SharedIndexInformer

	// InformerWithOptions returns a SharedIndexInformer configured with the given options.
	// The informer is lazily initialized and cached.
	//InformerWithOptions(opts GPUInformerOptions) cache.SharedIndexInformer
}

// GPUClient implements GPUInterface.
type GPUClient struct {
	client pb.GpuServiceClient
	logger logr.Logger
	//mu       sync.Mutex
	//informer GPUInformer
}

// NewGPUClient creates a new GPU client using the given gRPC connection and logger.
// If the logger is empty, it defaults to logr.Discard().
func NewGPUClient(conn *grpc.ClientConn, logger logr.Logger) *GPUClient {
	if (logger == logr.Logger{}) {
		logger = logr.Discard()
	}

	return &GPUClient{
		client: pb.NewGpuServiceClient(conn),
		logger: logger,
	}
}

// Get retrieves a GPU by name.
// If the context has no deadline, DefaultRPCTimeout is applied.
func (c *GPUClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*devicev1alpha1.GPU, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultRPCTimeout)
		defer cancel()
	}

	req := &pb.GetGpuRequest{
		Name: name,
	}

	resp, err := c.client.GetGpu(ctx, req)
	if err != nil {
		return nil, err
	}

	return devicev1alpha1.FromProto(resp.Gpu), nil
}

// List returns all GPUs.
// Only ResourceVersion in ListOptions is respected; other filters are ignored.
// If the context has no deadline, DefaultRPCTimeout is applied.
func (c *GPUClient) List(ctx context.Context, opts metav1.ListOptions) (*devicev1alpha1.GPUList, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultRPCTimeout)
		defer cancel()
	}

	req := &pb.ListGpusRequest{}
	resp, err := c.client.ListGpus(ctx, req)
	if err != nil {
		return nil, err
	}

	return devicev1alpha1.FromProtoList(resp.GpuList), nil
}

// Watch starts a watch for GPU resources from the given ResourceVersion.
// The returned watch.Interface delivers Added, Modified, and Deleted events.
// Canceling the context stops the watch.
func (c *GPUClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	ctx, cancel := context.WithCancel(ctx)

	req := &pb.WatchGpusRequest{
		ResourceVersion: opts.ResourceVersion,
	}

	stream, err := c.client.WatchGpus(ctx, req)
	if err != nil {
		cancel()
		return nil, err
	}

	return newStreamWatcher(stream, cancel, c.logger), nil
}

// Informer returns a SharedIndexInformer with default options.
// The informer is lazily initialized, thread-safe, and cached.
//func (c *GPUClient) Informer() cache.SharedIndexInformer {
//	return c.getOrCreateInformer(GPUInformerOptions{})
//}

// InformerWithOptions returns a SharedIndexInformer configured with the given options.
// The informer is lazily initialized, thread-safe, and cached.
//func (c *GPUClient) InformerWithOptions(opts GPUInformerOptions) cache.SharedIndexInformer {
//	return c.getOrCreateInformer(opts)
//}

// getOrCreateInformer lazily initializes and caches the GPUInformer with the given options.
// It is thread-safe.
//func (c *GPUClient) getOrCreateInformer(opts GPUInformerOptions) cache.SharedIndexInformer {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//
//	if c.informer == nil {
//		if opts.Indexers == nil {
//			opts.Indexers = cache.Indexers{}
//		}
//
//		c.informer = NewGPUInformer(c, opts)
//	}
//
//	return c.informer.Informer()
//}
