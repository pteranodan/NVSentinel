package v1alpha1

import (
	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	pb "github.com/nvidia/nvsentinel/api/gen/go/device/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
)

// gpuStreamAdapter wraps the GPU gRPC stream to provide events.
type gpuStreamAdapter struct {
	stream pb.GpuService_WatchGpusClient
}

func (a *gpuStreamAdapter) Next() (string, runtime.Object, error) {
	resp, err := a.stream.Recv()
	if err != nil {
		return "", nil, err
	}

	obj := devicev1alpha1.FromProto(resp.GetObject())

	return resp.GetType(), obj, nil
}

func (a *gpuStreamAdapter) Close() error {
	return a.stream.CloseSend()
}
