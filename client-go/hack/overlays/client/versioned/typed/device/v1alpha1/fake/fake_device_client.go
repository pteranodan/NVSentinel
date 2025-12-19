package fake

import (
	v1alpha1 "github.com/nvidia/nvsentinel/client-go/client/versioned/typed/device/v1alpha1"
	"google.golang.org/grpc"
	testing "k8s.io/client-go/testing"
)

type FakeDeviceV1alpha1 struct {
	*testing.Fake
}

func (c *FakeDeviceV1alpha1) GPUs() v1alpha1.GPUInterface {
	return newFakeGPUs(c)
}

// ClientConn returns a ClientConn that is used to communicate
// with gRPC server by this client implementation.
//
// Note: the Fake implementation uses the ObjectTracker memory store, not an actual gRPC connection.
func (c *FakeDeviceV1alpha1) ClientConn() grpc.ClientConnInterface {
	var ret *grpc.ClientConn
	return ret
}
