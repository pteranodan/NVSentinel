package options

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/k3s-io/kine/pkg/endpoint"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server/options"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

type Options struct {
	DatabasePath                string
	CompactionInterval          time.Duration
	CompactionBatchSize         int64
	WatchProgressNotifyInterval time.Duration

	KineConfig     endpoint.Config
	KineSocketPath string
	DatabaseDir    string
	Etcd           *options.EtcdOptions
}

type completedOptions struct {
	Options
}

type CompletedOptions struct {
	*completedOptions
}

func NewOptions() *Options {
	return &Options{
		DatabasePath:                "/var/lib/nvidia-device-api/state.db",
		CompactionInterval:          5 * time.Minute,
		CompactionBatchSize:         1000,
		WatchProgressNotifyInterval: 5 * time.Second,
		Etcd:                        options.NewEtcdOptions(apistorage.NewDefaultConfig("/registry", nil)),
	}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.StringVar(&o.DatabasePath, "database-path", o.DatabasePath,
		"The path to the SQLite database file.")
	fs.DurationVar(&o.CompactionInterval, "compaction-interval", o.CompactionInterval,
		"The interval of compaction requests. If 0, the compaction request from apiserver is disabled.")
	fs.Int64Var(&o.CompactionBatchSize, "compaction-batch-size", o.CompactionBatchSize,
		"Number of revisions to compact in a single batch.")
	fs.DurationVar(&o.WatchProgressNotifyInterval, "watch-progress-notify-interval", o.WatchProgressNotifyInterval,
		"Interval between periodic watch progress notifications.")
}

func (o *Options) Complete() (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{}, nil
	}

	o.KineConfig.Endpoint = fmt.Sprintf("sqlite://%s?_journal=WAL&_timeout=5000&_synchronous=NORMAL&_fk=1", o.DatabasePath)
	o.DatabaseDir = filepath.Dir(o.DatabasePath)

	o.KineSocketPath = "/var/run/nvidia-device-api/kine.sock"
	o.KineConfig.Listener = "unix://" + o.KineSocketPath

	if len(o.Etcd.StorageConfig.Transport.ServerList) == 0 {
		o.Etcd.StorageConfig.Transport.ServerList = []string{o.KineConfig.Listener}
	}

	o.KineConfig.CompactInterval = o.CompactionInterval
	o.KineConfig.CompactBatchSize = o.CompactionBatchSize
	o.KineConfig.NotifyInterval = o.WatchProgressNotifyInterval

	completed := completedOptions{
		Options: *o,
	}

	return CompletedOptions{
		completedOptions: &completed,
	}, nil
}

func (o *Options) Validate() []error {
	if o == nil {
		return nil
	}

	allErrors := []error{}

	if o.DatabasePath == "" {
		allErrors = append(allErrors, fmt.Errorf("database-path is required"))
	} else if !filepath.IsAbs(o.DatabasePath) {
		allErrors = append(allErrors, fmt.Errorf("invalid database-path %q: must be an absolute path", o.DatabasePath))
	}

	if o.DatabaseDir == "" {
		allErrors = append(allErrors, fmt.Errorf("internal error: database directory was not intialized"))
	}

	if o.CompactionInterval < 0 {
		allErrors = append(allErrors, fmt.Errorf("invalid compaction-interval %v: must be non-negative", o.CompactionInterval))
	}
	if o.CompactionBatchSize <= 0 {
		allErrors = append(allErrors, fmt.Errorf("invalid compaction-batch-size %q: must be greater than 0", o.CompactionBatchSize))
	}

	if o.Etcd != nil {
		allErrors = append(allErrors, o.Etcd.Validate()...)
	}

	if o.KineSocketPath == "" {
		allErrors = append(allErrors, fmt.Errorf("internal error: storage socket path was not initialized"))
	}

	return allErrors
}

func (o *Options) ApplyTo(storageConfig *apistorage.Config) error {
	if o == nil {
		return nil
	}

	*storageConfig = o.Etcd.StorageConfig

	return nil
}
