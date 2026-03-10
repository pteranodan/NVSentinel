// Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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

// Package options contains flags and options for initializing a device apiserver.
package options

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	nvgrpc "github.com/nvidia/nvsentinel/pkg/grpc/options"
	storagebackend "github.com/nvidia/nvsentinel/pkg/storage/storagebackend/options"
	nvvalidation "github.com/nvidia/nvsentinel/pkg/util/validation"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/validation"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	logsapi "k8s.io/component-base/logs/api/v1"
)

const defaultBindAddress = "unix:///var/run/nvidia-device-api/device-api.sock"

// Options define the flags and validation for a device controlplane.
type Options struct {
	NodeName             string
	BindAddress          string
	HealthAddress        string
	ServiceMonitorPeriod time.Duration
	MetricsAddress       string
	ShutdownGracePeriod  time.Duration
	EnablePprof          bool
	PprofAddress         string

	GRPC    *nvgrpc.Options
	Storage *storagebackend.Options
	Logs    *logs.Options
}

// completedOptions is a private wrapper that enforces a call of Complete() before Run can be invoked.
type completedOptions struct {
	Options

	Server  []grpc.ServerOption
	Storage storagebackend.CompletedOptions
}

type CompletedOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedOptions
}

// NewOptions creates a new Server Options object with default parameters.
func NewOptions() *Options {
	grpcOpts := nvgrpc.NewOptions()

	storageOpts := storagebackend.NewOptions()
	storageOpts.GRPC = grpcOpts

	return &Options{
		BindAddress:          defaultBindAddress,
		HealthAddress:        ":50051",
		ServiceMonitorPeriod: 10 * time.Second,
		MetricsAddress:       ":9090",
		ShutdownGracePeriod:  25 * time.Second,
		PprofAddress:         ":6060",
		GRPC:                 grpcOpts,
		Storage:              storageOpts,
		Logs:                 logs.NewOptions(),
	}
}

func (o *Options) AddFlags(fss *cliflag.NamedFlagSets) {
	if o == nil {
		return
	}

	genericFs := fss.FlagSet("generic")

	genericFs.StringVar(&o.NodeName, "hostname-override", o.NodeName,
		"If non-empty, will use this string as identification instead of the actual hostname. Must be a valid DNS subdomain.")

	genericFs.StringVar(&o.BindAddress, "bind-address", o.BindAddress,
		"The unix socket address on which to listen for gRPC requests. Must be a unix:// absolute path.")

	genericFs.StringVar(&o.HealthAddress, "health-probe-bind-address", o.HealthAddress,
		"The TCP address (IP:port) to serve gRPC health and reflection. Defaults to ':50051'.")
	genericFs.DurationVar(&o.ServiceMonitorPeriod, "service-monitor-period", o.ServiceMonitorPeriod,
		"The period for syncing internal service status. Must be between 0s and 1m.")

	genericFs.StringVar(&o.MetricsAddress, "metrics-bind-address", o.MetricsAddress,
		"The TCP address (IP:port) to serve HTTP metrics. Defaults to ':9090'.")

	genericFs.DurationVar(&o.ShutdownGracePeriod, "shutdown-grace-period", o.ShutdownGracePeriod,
		"The maximum duration to wait for the server to shut down gracefully before forcing a stop. Must be between 0s and 10m.")

	debugFs := fss.FlagSet("debug")

	debugFs.BoolVar(&o.EnablePprof, "enable-pprof", o.EnablePprof,
		"Enable runtime profiling via pprof.")
	debugFs.StringVar(&o.PprofAddress, "pprof-bind-address", o.PprofAddress,
		"The address the pprof profiler should bind to. Defaults to ':6060'.")

	o.GRPC.AddFlags(fss)
	o.Storage.AddFlags(fss)
	logsapi.AddFlags(o.Logs, fss.FlagSet("logs"))
}

func (o *Options) Complete() (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{completedOptions: &completedOptions{}}, nil
	}

	completed := completedOptions{
		Options: *o,
	}

	nodeName := o.NodeName
	if nodeName == "" {
		if envNodeName := os.Getenv("NODE_NAME"); envNodeName != "" {
			nodeName = envNodeName
		} else {
			var err error
			nodeName, err = os.Hostname()
			if err != nil {
				return CompletedOptions{}, fmt.Errorf("failed to resolve hostname: %w", err)
			}
		}
	}
	completed.NodeName = strings.ToLower(strings.TrimSpace(nodeName))

	completed.BindAddress = o.BindAddress
	completed.HealthAddress = o.HealthAddress
	completed.ServiceMonitorPeriod = o.ServiceMonitorPeriod
	completed.MetricsAddress = o.MetricsAddress
	completed.ShutdownGracePeriod = o.ShutdownGracePeriod
	completed.EnablePprof = o.EnablePprof
	completed.PprofAddress = o.PprofAddress

	completedGRPC, err := o.GRPC.Complete()
	if err != nil {
		return CompletedOptions{}, fmt.Errorf("failed to complete grpc options: %w", err)
	}

	if err := completedGRPC.ApplyTo(&completed.Server); err != nil {
		return CompletedOptions{}, fmt.Errorf("failed to apply grpc options: %w", err)
	}

	completedStorage, err := o.Storage.Complete()
	if err != nil {
		return CompletedOptions{}, fmt.Errorf("failed to complete storage options: %w", err)
	}
	completed.Storage = completedStorage

	return CompletedOptions{
		completedOptions: &completed,
	}, nil
}

func (o *CompletedOptions) Validate() []error {
	if o == nil {
		return nil
	}

	allErrors := []error{}

	if o.NodeName == "" {
		allErrors = append(allErrors, fmt.Errorf("--hostname-override: required"))
	} else {
		if validationErrors := validation.IsDNS1123Subdomain(o.NodeName); len(validationErrors) > 0 {
			for _, errDesc := range validationErrors {
				allErrors = append(allErrors, fmt.Errorf("--hostname-override %q: %s", o.NodeName, errDesc))
			}
		}
	}

	if o.BindAddress == "" {
		allErrors = append(allErrors, fmt.Errorf("--bind-address: required"))
	} else {
		if validationErrors := nvvalidation.IsUnixSocketURI(o.BindAddress); len(validationErrors) > 0 {
			for _, errDesc := range validationErrors {
				allErrors = append(allErrors, fmt.Errorf("--bind-address %q: %s", o.BindAddress, errDesc))
			}
		}
	}

	if o.HealthAddress == "" {
		allErrors = append(allErrors, fmt.Errorf("--health-probe-bind-address: required"))
	} else {
		if validationErrors := nvvalidation.IsTCPAddress(o.HealthAddress); len(validationErrors) > 0 {
			for _, errDesc := range validationErrors {
				allErrors = append(allErrors, fmt.Errorf("--health-probe-bind-address %q: %s", o.HealthAddress, errDesc))
			}
		}
	}

	if o.ServiceMonitorPeriod < 0 {
		allErrors = append(allErrors, fmt.Errorf("--service-monitor-period %v: must be greater than or equal to 0s", o.ServiceMonitorPeriod))
	} else if o.ServiceMonitorPeriod > 1*time.Minute {
		allErrors = append(allErrors, fmt.Errorf("--service-monitor-period %v: must be 1m or less", o.ServiceMonitorPeriod))
	}

	if o.MetricsAddress != "" {
		if validationErrors := nvvalidation.IsTCPAddress(o.MetricsAddress); len(validationErrors) > 0 {
			for _, errDesc := range validationErrors {
				allErrors = append(allErrors, fmt.Errorf("--metrics-bind-address %q: %s", o.MetricsAddress, errDesc))
			}
		}
	}

	if o.ShutdownGracePeriod < 0 {
		allErrors = append(allErrors, fmt.Errorf("--shutdown-grace-period %v: must be greater than or equal to 0s", o.ShutdownGracePeriod))
	} else if o.ShutdownGracePeriod > 10*time.Minute {
		allErrors = append(allErrors, fmt.Errorf("--shutdown-grace-period %v: must be 10m or less", o.ShutdownGracePeriod))
	}

	if o.EnablePprof && o.PprofAddress == "" {
		allErrors = append(allErrors, fmt.Errorf("--pprof-bind-address: required"))
	}

	if o.EnablePprof && o.PprofAddress != "" {
		if validationErrors := nvvalidation.IsTCPAddress(o.PprofAddress); len(validationErrors) > 0 {
			for _, errDesc := range validationErrors {
				allErrors = append(allErrors, fmt.Errorf("--pprof-bind-address %q: %s", o.PprofAddress, errDesc))
			}
		}
	}

	addresses := map[string]string{
		"--health-probe-bind-address": o.HealthAddress,
		"--metrics-bind-address":      o.MetricsAddress,
	}
	if o.EnablePprof {
		addresses["--pprof-bind-address"] = o.PprofAddress
	}

	ports := make(map[string]string)
	for name, addr := range addresses {
		if addr == "" {
			continue
		}
		_, port, err := net.SplitHostPort(addr)
		if err == nil {
			for otherName, otherPort := range ports {
				if port == otherPort {
					allErrors = append(allErrors, fmt.Errorf("%s and %s: must not use the same port %q", name, otherName, port))
				}
			}
			ports[name] = port
		}
	}

	allErrors = append(allErrors, o.GRPC.Validate()...)
	allErrors = append(allErrors, o.Storage.Validate()...)

	if o.Logs != nil {
		if logErrs := logsapi.Validate(o.Logs, nil, nil); len(logErrs) > 0 {
			allErrors = append(allErrors, logErrs.ToAggregate().Errors()...)
		}
	}

	return allErrors
}
