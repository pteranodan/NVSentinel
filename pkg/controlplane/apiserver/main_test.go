package apiserver

import (
	"io"
	"testing"

	"github.com/nvidia/nvsentinel/pkg/util/test"
	"google.golang.org/grpc/grpclog"
)

func init() {
	// silence transport-level gRPC logs
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(io.Discard, io.Discard, io.Discard))
}

func TestMain(m *testing.M) {
	test.VerifyTestMain(m)
}
