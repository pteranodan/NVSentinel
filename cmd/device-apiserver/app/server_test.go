package app

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/nvidia/nvsentinel/cmd/device-apiserver/app/options"
	"github.com/nvidia/nvsentinel/pkg/util/testutils"
)

func TestRun(t *testing.T) {
	opts := options.NewServerRunOptions()

	localSocket := testutils.NewUnixAddr(t, "integration.sock")
	kineSocket := fmt.Sprintf("unix://%s", testutils.NewUnixAddr(t, "kine.sock"))
	healthAddr := testutils.GetFreeTCPAddress(t)

	opts.GRPC.BindAddress = "unix://" + localSocket
	opts.HealthAddress = healthAddr
	opts.NodeName = "test-node"

	tmpDir := t.TempDir()
	opts.Storage.DatabaseDir = tmpDir
	opts.Storage.KineSocketPath = kineSocket
	opts.Storage.KineConfig.Endpoint = fmt.Sprintf("sqlite://%s/db.sqlite", tmpDir)
	opts.Storage.KineConfig.Listener = kineSocket

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	completedOpts, err := opts.Complete(ctx)
	if err != nil {
		t.Fatalf("Failed to complete options: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, completedOpts)
	}()

	testutils.WaitForRunning(t, healthAddr, 5*time.Second)

	// 2. LOGIC CHECK: (Optional)
	// You could dial the Device API here to ensure data can be fetched.

	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Errorf("App exited with unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("App leaked goroutines and failed to shut down within grace period")
	}

	if _, err := os.Stat(localSocket); err == nil {
		t.Errorf("UDS socket file %q still exists after shutdown", localSocket)
	}
}
