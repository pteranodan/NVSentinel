package testutils

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NewUnixAddr creates a temporary directory and returns a path for a UDS socket.
func NewUnixAddr(t testing.TB, path string) string {
	t.Helper()

	d, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := os.RemoveAll(d); err != nil {
			t.Error(err)
		}
	})

	return filepath.Join(d, path)
}

// GetFreeTCPAddress finds an available port on the loopback interface.
func GetFreeTCPAddress(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find a free port: %v", err)
	}
	addr := l.Addr().String()

	l.Close()

	return addr
}

// WaitForRunning blocks until a gRPC health check returns successfully or the timeout is reached.
func WaitForRunning(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Bypass gRPC's internal DNS resolver for IP:Port strings
	dialTarget := fmt.Sprintf("passthrough:///%s", addr)
	conn, err := grpc.DialContext(ctx, dialTarget,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("Failed to dial %s: %v", addr, err)
	}
	defer conn.Close()

	client := healthpb.NewHealthClient(conn)
	err = wait.PollUntilContextTimeout(ctx, 200*time.Millisecond, timeout, true, func(ctx context.Context) (bool, error) {
		resp, err := client.Check(ctx, &healthpb.HealthCheckRequest{Service: ""})
		if err != nil {
			return false, nil
		}
		return resp.GetStatus() == healthpb.HealthCheckResponse_SERVING, nil
	})

	if err != nil {
		t.Fatalf("Server never reached SERVING status on %s: %v", addr, err)
	}
}
