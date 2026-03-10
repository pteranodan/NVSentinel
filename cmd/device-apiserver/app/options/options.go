//  Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package options

import (
	"io"

	cp "github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/options"
	nvgrpclog "github.com/nvidia/nvsentinel/pkg/grpc/log"
	nvlogrus "github.com/nvidia/nvsentinel/pkg/logrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/grpclog"
	cliflag "k8s.io/component-base/cli/flag"
	logsapi "k8s.io/component-base/logs/api/v1"
)

type ServerRunOptions struct {
	*cp.Options
}

type completedOptions struct {
	cp.CompletedOptions
}

type CompletedOptions struct {
	*completedOptions
}

func NewServerRunOptions() *ServerRunOptions {
	return &ServerRunOptions{
		Options: cp.NewOptions(),
	}
}

func (s *ServerRunOptions) Flags() cliflag.NamedFlagSets {
	fss := cliflag.NamedFlagSets{}
	if s == nil || s.Options == nil {
		return fss
	}

	s.AddFlags(&fss)

	return fss
}

// ApplyLogging configures the global logging state using the server's log options.
// It sets up klog, redirects gRPC logs, and bridges logrus to klog.
func (o *ServerRunOptions) ApplyLogging() error {
	if o == nil || o.Options == nil || o.Logs == nil {
		return nil
	}

	logsapi.ReapplyHandling = logsapi.ReapplyHandlingIgnoreUnchanged
	if err := logsapi.ValidateAndApply(o.Options.Logs, nil); err != nil {
		return err
	}

	grpclog.SetLoggerV2(&nvgrpclog.KlogAdapter{
		Verbosity: uint32(o.Options.Logs.Verbosity),
	})

	logrus.SetFormatter(&nvlogrus.KlogFormatter{
		Verbosity: uint32(o.Options.Logs.Verbosity),
	})
	// Prevent double-printing
	logrus.SetOutput(io.Discard)

	return nil
}

func (o *ServerRunOptions) Complete() (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{completedOptions: &completedOptions{}}, nil
	}

	controlplane, err := o.Options.Complete()
	if err != nil {
		return CompletedOptions{}, err
	}

	completed := completedOptions{
		CompletedOptions: controlplane,
	}

	return CompletedOptions{
		completedOptions: &completed,
	}, nil
}

func (o completedOptions) Validate() []error {
	return o.CompletedOptions.Validate()
}
