package options

import (
	"context"
	"fmt"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
	"k8s.io/component-base/metrics"
)

type Options struct {
	NodeName string
	GRPC     *GRPCOptions
	Storage  *StorageOptions
	Metrics  *metrics.Options
	Logs     *logs.Options
}

func NewOptions() *Options {
	return &Options{
		GRPC:    NewGRPCOptions(),
		Storage: NewStorageOptions(),
		Metrics: metrics.NewOptions(),
		Logs:    logs.NewOptions(),
	}
}

func (s *Options) AddFlags(fss *cliflag.NamedFlagSets) {
	if s == nil {
		return
	}

	genericFs := fss.FlagSet("generic")
	genericFs.StringVar(&s.NodeName, "hostname-override", s.NodeName,
		"If non-empty, will use this string as identification instead of the actual hostname.")

	s.GRPC.AddFlags(fss.FlagSet("grpc"))
	s.Storage.AddFlags(fss.FlagSet("storage"))
	s.Metrics.AddFlags(fss.FlagSet("metrics"))
	logsapi.AddFlags(s.Logs, fss.FlagSet("logs"))
}

func (o *Options) Complete(ctx context.Context) (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{completedOptions: &completedOptions{}}, nil
	}

	if o.NodeName == "" {
		hostname, err := os.Hostname()
		if err != nil || hostname == "" {
			hostname = os.Getenv("NODE_NAME")
		}
		o.NodeName = strings.ToLower(hostname)
	}

	if err := o.GRPC.Complete(); err != nil {
		return CompletedOptions{}, err
	}

	if err := o.Storage.Complete(); err != nil {
		return CompletedOptions{}, err
	}

	completed := completedOptions{
		Options: *o,
	}

	// TODO: cross-component defaulting

	return CompletedOptions{
		completedOptions: &completed,
	}, nil
}

func (o *Options) Validate() []error {
	var errs []error

	if o.NodeName == "" {
		errs = append(errs, fmt.Errorf("hostname-override is required"))
	} else {
		if validationErrors := validation.IsDNS1123Subdomain(o.NodeName); len(validationErrors) > 0 {
			for _, errDesc := range validationErrors {
				errs = append(errs, fmt.Errorf("hostname-override %q is invalid: %s", o.NodeName, errDesc))
			}
		}
	}

	if o.GRPC != nil {
		errs = append(errs, o.GRPC.Validate()...)
	}

	if o.Storage != nil {
		errs = append(errs, o.Storage.Validate()...)
	}

	if o.Logs != nil {
		if logErrs := logsapi.Validate(o.Logs, nil, nil); len(logErrs) > 0 {
			errs = append(errs, logErrs.ToAggregate().Errors()...)
		}
	}

	return errs
}

type completedOptions struct {
	Options
}

type CompletedOptions struct {
	*completedOptions
}

func (o CompletedOptions) Validate() []error {
	var errs []error

	errs = append(errs, o.Options.Validate()...)

	return errs
}
