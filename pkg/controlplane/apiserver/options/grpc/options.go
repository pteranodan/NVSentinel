// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package grpc

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Options struct {
	BindAddress          string
	MaxConcurrentStreams uint32
	MaxRecvMsgSize       int
	MaxSendMsgSize       int
	MaxConnectionIdle    time.Duration
	KeepAliveTime        time.Duration
	KeepAliveTimeout     time.Duration
	MinPingInterval      time.Duration
	PermitWithoutStream  bool
}

type completedOptions struct {
	Options
}

type CompletedOptions struct {
	*completedOptions
}

func NewOptions() *Options {
	return &Options{
		BindAddress:          "unix:///var/run/nvidia-device-api/device-api.sock",
		MaxConcurrentStreams: 250,
		MaxRecvMsgSize:       4194304,  // 4MiB
		MaxSendMsgSize:       16777216, // 16MiB
		MaxConnectionIdle:    5 * time.Minute,
		KeepAliveTime:        1 * time.Minute,
		KeepAliveTimeout:     10 * time.Second,
		MinPingInterval:      5 * time.Second,
		PermitWithoutStream:  true,
	}
}

// AddFlags adds flags related to gRPC for a specific APIServer to the specified FlagSet
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	if o == nil {
		return
	}

	fs.StringVar(&o.BindAddress, "bind-address", o.BindAddress,
		"The address on which to listen for gRPC requests.")
	fs.Uint32Var(&o.MaxConcurrentStreams, "max-streams-per-connection", o.MaxConcurrentStreams,
		"The maximum number of concurrent streams allowed per connection.")
	fs.IntVar(&o.MaxRecvMsgSize, "max-recv-msg-size", o.MaxRecvMsgSize,
		"The maximum message size in bytes the server can receive.")
	fs.IntVar(&o.MaxSendMsgSize, "max-send-msg-size", o.MaxSendMsgSize,
		"The maximum message size in bytes the server can send.")
	fs.DurationVar(&o.KeepAliveTime, "grpc-keepalive-time", o.KeepAliveTime,
		"Duration after which a keepalive probe is sent.")
	fs.DurationVar(&o.KeepAliveTimeout, "grpc-keepalive-timeout", o.KeepAliveTimeout,
		"Duration the server waits for a keepalive response.")
}

func (o *Options) Complete() (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{}, nil
	}

	if o.BindAddress == "" {
		o.BindAddress = "unix:///var/run/nvidia-device-api/device-api.sock"
	}

	if o.MaxConcurrentStreams == 0 {
		o.MaxConcurrentStreams = 250
	}

	if o.MaxRecvMsgSize == 0 {
		o.MaxRecvMsgSize = 4194304 // 4MiB
	}

	if o.MaxSendMsgSize == 0 {
		o.MaxSendMsgSize = 16777216 // 16MiB
	}

	if o.MaxConnectionIdle == 0 {
		o.MaxConnectionIdle = 5 * time.Minute
	}

	if o.KeepAliveTime == 0 {
		o.KeepAliveTime = 1 * time.Minute
	}

	if o.KeepAliveTimeout == 0 {
		o.KeepAliveTimeout = 10 * time.Second
	}

	if o.MinPingInterval == 0 {
		o.MinPingInterval = 5 * time.Second
	}

	o.PermitWithoutStream = true

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

	if !strings.HasPrefix(o.BindAddress, "unix://") {
		allErrors = append(allErrors, fmt.Errorf("bind-address %q: must start with 'unix://'", o.BindAddress))
		return allErrors
	}
	path := strings.TrimPrefix(o.BindAddress, "unix://")
	if !filepath.IsAbs(path) {
		allErrors = append(allErrors, fmt.Errorf("bind-address path %q: must be an absolute path", path))
	}
	if strings.HasSuffix(path, string(filepath.Separator)) {
		allErrors = append(allErrors, fmt.Errorf("bind-address path %q: must not end with a trailing slash", path))
	}

	if o.MaxConcurrentStreams > 10000 {
		allErrors = append(allErrors, fmt.Errorf("max-streams-per-connection: %d must be 10000 or less", o.MaxConcurrentStreams))
	}
	if o.MaxRecvMsgSize > 4194304 {
		allErrors = append(allErrors, fmt.Errorf("max-recv-msg-size: %d must be 4MiB or less", o.MaxRecvMsgSize))
	}
	if o.MaxSendMsgSize > 16777216 {
		allErrors = append(allErrors, fmt.Errorf("max-send-msg-size: %d must be 16MiB or less", o.MaxSendMsgSize))
	}

	if o.KeepAliveTime < 0 {
		allErrors = append(allErrors, fmt.Errorf("grpc-keepalive-time: %v must be 0s or greater", o.KeepAliveTime))
	}
	if o.KeepAliveTimeout < 0 {
		allErrors = append(allErrors, fmt.Errorf("grpc-keepalive-timeout: %v must be 0s or greater", o.KeepAliveTimeout))
	}
	if o.MinPingInterval < 5*time.Second {
		allErrors = append(allErrors, fmt.Errorf("min-ping-interval: %v must be at least 5s", o.MinPingInterval))
	}
	if o.KeepAliveTimeout >= o.KeepAliveTime && o.KeepAliveTime > 0 {
		allErrors = append(allErrors, fmt.Errorf("grpc-keepalive-timeout: %v must be less than grpc-keepalive-time (%v)", o.KeepAliveTimeout, o.KeepAliveTime))
	}
	if o.KeepAliveTime < o.MinPingInterval && o.KeepAliveTime > 0 {
		allErrors = append(allErrors, fmt.Errorf("grpc-keepalive-time: %v must be greater than or equal to min-ping-interval (%v)", o.KeepAliveTime, o.MinPingInterval))
	}

	if !o.PermitWithoutStream {
		allErrors = append(allErrors, fmt.Errorf("permit-without-stream: %v must be true to allow keepalive pings without active streams", o.PermitWithoutStream))
	}

	return allErrors
}

func (o *Options) ApplyTo(
	bindAddress *string,
	serverOpts *[]grpc.ServerOption,
) error {
	if o == nil {
		return nil
	}

	*bindAddress = o.BindAddress

	*serverOpts = append(*serverOpts, grpc.MaxConcurrentStreams(o.MaxConcurrentStreams))
	*serverOpts = append(*serverOpts, grpc.MaxRecvMsgSize(o.MaxRecvMsgSize))
	*serverOpts = append(*serverOpts, grpc.MaxSendMsgSize(o.MaxSendMsgSize))

	*serverOpts = append(*serverOpts, grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: o.MaxConnectionIdle,
		Time:              o.KeepAliveTime,
		Timeout:           o.KeepAliveTimeout,
	}))

	*serverOpts = append(*serverOpts, grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
		MinTime:             o.MinPingInterval,
		PermitWithoutStream: o.PermitWithoutStream,
	}))

	return nil
}
