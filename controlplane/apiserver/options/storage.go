package options

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/klog/v2"

	"github.com/k3s-io/kine/pkg/endpoint"
)

type StorageOptions struct {
	Etcd *options.EtcdOptions

	// DatastoreEndpoint is the Kine-compatible connection string.
	DatastoreEndpoint string
}

func NewStorageOptions() *StorageOptions {
	return &StorageOptions{
		Etcd: options.NewEtcdOptions(storagebackend.NewDefaultConfig("/registry", nil)),
	}
}

func (s *StorageOptions) AddFlags(fs *pflag.FlagSet) {
	if s == nil {
		return
	}

	// TODO(dhuenecke): hide unused etcd flags?
	s.Etcd.AddFlags(fs)

	fs.StringVar(&s.DatastoreEndpoint, "datastore-endpoint", s.DatastoreEndpoint,
		"The connection string for the underlying storage backend (e.g. 'sqlite://data.db'). "+
			"This is used by the internal Kine shim to translate etcd API calls into SQL queries.")
}

func (s *StorageOptions) Complete() error {
	if s == nil {
		return nil
	}

	if s.DatastoreEndpoint == "" {
		s.DatastoreEndpoint = "sqlite:///var/lib/nvidia-device-api/state.db" +
			"?_journal=WAL" + // Write-Ahead Log
			"&_timeout=5000" + // Busy timeout (5 seconds)
			"&_synchronous=NORMAL" + // Faster writes with Write-Ahead Log (WAL) mode
			"&_fk=1" // Enable foreign key enforcement
	}

	if s.DatastoreEndpoint != "" && len(s.Etcd.StorageConfig.Transport.ServerList) == 0 {
		s.Etcd.StorageConfig.Transport.ServerList = []string{"unix:///var/run/nvidia-device-api/kine.sock"}
	}
	return nil
}

func (s *StorageOptions) Validate() []error {
	if s == nil {
		return nil
	}

	return s.Etcd.Validate()
}

func (s *StorageOptions) ApplyTo(stopCh <-chan struct{}) error {
	if s == nil {
		return nil
	}

	if s.DatastoreEndpoint != "" {
		klog.V(4).InfoS("Initializing storage layer", "datasource", s.DatastoreEndpoint)

		if strings.HasPrefix(s.DatastoreEndpoint, "sqlite://") {
			dbPath := strings.TrimPrefix(s.DatastoreEndpoint, "sqlite://")
			if idx := strings.Index(dbPath, "?"); idx != -1 {
				dbPath = dbPath[:idx]
			}

			klog.V(2).InfoS("Ensuring sqlite directory exists", "path", filepath.Dir(dbPath))
			if err := os.MkdirAll(filepath.Dir(dbPath), 0750); err != nil {
				return fmt.Errorf("failed to create sqlite directory: %w", err)
			}
		}

		listenAddr := "unix:///var/run/nvidia-device-api/kine.sock"
		if len(s.Etcd.StorageConfig.Transport.ServerList) > 0 {
			listenAddr = s.Etcd.StorageConfig.Transport.ServerList[0]
			if !strings.HasPrefix(listenAddr, "unix://") && strings.HasPrefix(listenAddr, "/") {
				listenAddr = "unix://" + listenAddr
			}
		}

		socketPath := strings.TrimPrefix(listenAddr, "unix://")
		if strings.HasPrefix(socketPath, "/") {
			klog.V(4).InfoS("Cleaning up stale kine socket", "path", socketPath)
			if err := os.MkdirAll(filepath.Dir(socketPath), 0750); err != nil {
				return err
			}
			if err := os.Remove(socketPath); err != nil && !os.IsNotExist(err) {
				klog.V(2).InfoS("Unable to remove stale kine socket", "path", socketPath, "err", err)
			} else if err == nil {
				klog.V(4).InfoS("Cleaned up stale kine socket", "path", socketPath)
			}
		}

		kineConfig := endpoint.Config{
			Listener:         listenAddr,
			Endpoint:         s.DatastoreEndpoint,
			CompactBatchSize: 1000,
			NotifyInterval:   5 * time.Second,
		}

		klog.InfoS("Starting Kine storage endpoint", "listenAddr", listenAddr)
		ctx := wait.ContextForChannel(stopCh)
		_, err := endpoint.Listen(ctx, kineConfig)
		if err != nil {
			return fmt.Errorf("unable to initialize storage backend: %w", err)
		}

		if strings.HasPrefix(socketPath, "/") {
			klog.V(2).InfoS("Waiting for storage socket to be ready", "path", socketPath)
			err = wait.PollUntilContextTimeout(ctx, 50*time.Millisecond, 2*time.Second, true, func(ctx context.Context) (bool, error) {
				if _, err := os.Stat(socketPath); err != nil {
					return false, nil
				}
				if err := os.Chmod(socketPath, 0660); err != nil {
					klog.V(4).ErrorS(err, "Failed to chmod socket, retrying", "path", socketPath)
					return false, nil
				}
				return true, nil
			})
		}
		if err != nil {
			return fmt.Errorf("timed out waiting for storage socket %q to become ready: %w", socketPath, err)
		}

		klog.V(2).InfoS("Storage layer is ready", "listenAddr", listenAddr)
	}

	return nil
}
