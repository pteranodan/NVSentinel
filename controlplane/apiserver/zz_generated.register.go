package apiserver

import (
	"os"

	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
)

func RegisterServices(svr *grpc.Server, completedConfig *CompletedConfig) {
	const resource = "/registry"

	storage, err := completedConfig.NewStorage(
		resource,
		func() runtime.Object { return &devicev1alpha1.GPU{} },
		func() runtime.Object { return &devicev1alpha1.GPUList{} },
	)
	if err != nil {
		klog.ErrorS(err, "Unable to create storage backend",
			"node", completedConfig.NodeName,
			"resource", resource,
		)
		os.Exit(1)
	}

	gpuService := NewGPUService(storage, completedConfig.NodeName)
	pb.RegisterGpuServiceServer(svr, gpuService)

	klog.V(2).InfoS("Registered GpuService with storage backend",
		"node", completedConfig.NodeName,
		"resource", resource,
	)
}
