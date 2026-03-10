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

// Package options contains flags and options for initializing a gRPC server.
package options

import (
	"fmt"
	"math"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	cliflag "k8s.io/component-base/cli/flag"
)

// Options define the flags and validation for a gRPC server.
type Options struct {
	MaxConcurrentStreams        uint32
	MaxRecvMsgSize              int
	MaxSendMsgSize              int
	WriteBufferSize             int
	SharedWriteBuffer           bool
	ReadBufferSize              int
	InitialWindowSize           int32
	InitialConnectionWindowSize int32
	MaxConnectionAge            time.Duration
	MaxConnectionAgeGrace       time.Duration
	MaxConnectionIdle           time.Duration
	KeepAliveTime               time.Duration
	KeepAliveTimeout            time.Duration
	MinPingInterval             time.Duration
	PermitWithoutStream         bool
}

type completedOptions struct {
	Options
}

type CompletedOptions struct {
	// Embed a private pointer that cannot be instantiated outside of this package.
	*completedOptions
}

// NewOptions creates a new gRPC Options object with default parameters.
func NewOptions() *Options {
	return &Options{
		MaxConcurrentStreams:        100,
		MaxRecvMsgSize:              2 << 20,   // 2MiB
		MaxSendMsgSize:              1 << 24,   // 16MiB
		WriteBufferSize:             512 << 10, // 512KiB
		SharedWriteBuffer:           true,
		ReadBufferSize:              512 << 10, // 512KiB
		InitialWindowSize:           4 << 20,   // 4MiB
		InitialConnectionWindowSize: 4 << 20,   // 4MiB
		MaxConnectionAge:            30 * time.Minute,
		MaxConnectionAgeGrace:       1 * time.Minute,
		MaxConnectionIdle:           10 * time.Minute,
		KeepAliveTime:               1 * time.Minute,
		KeepAliveTimeout:            20 * time.Second,
		MinPingInterval:             5 * time.Second,
		PermitWithoutStream:         false,
	}
}

func (o *Options) AddFlags(fss *cliflag.NamedFlagSets) {
	if o == nil {
		return
	}

	grpcFs := fss.FlagSet("grpc")

	grpcFs.Uint32Var(&o.MaxConcurrentStreams, "grpc-max-streams-per-connection", o.MaxConcurrentStreams,
		"The maximum number of concurrent streams allowed per connection. Must be between 1 and 10000.")

	grpcFs.IntVar(&o.MaxRecvMsgSize, "grpc-max-recv-msg-size", o.MaxRecvMsgSize,
		"The maximum message size in bytes the server can receive. Set to 0 to use gRPC default (4MiB). Must be at least 1024 bytes.")
	grpcFs.IntVar(&o.MaxSendMsgSize, "grpc-max-send-msg-size", o.MaxSendMsgSize,
		"The maximum message size in bytes the server can send. Set to 0 to use gRPC default (unlimited). Must be at least 1024 bytes.")
	grpcFs.IntVar(&o.WriteBufferSize, "grpc-write-buffer-size", o.WriteBufferSize,
		fmt.Sprintf("Size of the gRPC write buffer in bytes. Set to 0 to use gRPC default (32KiB). Must be between 0 and %d.", math.MaxInt32))
	grpcFs.BoolVar(&o.SharedWriteBuffer, "grpc-shared-write-buffer", o.SharedWriteBuffer,
		"Enable sharing of write buffers across transport streams.")
	grpcFs.IntVar(&o.ReadBufferSize, "grpc-read-buffer-size", o.ReadBufferSize,
		fmt.Sprintf("Size of the gRPC read buffer in bytes. Set to 0 to use gRPC default (32KiB). Must be between 0 and %d.", math.MaxInt32))
	grpcFs.Int32Var(&o.InitialWindowSize, "grpc-initial-window-size", o.InitialWindowSize,
		fmt.Sprintf("The initial HTTP/2 stream-level flow control window size in bytes. Set to 0 to use gRPC default (65535). Must be between 65535 and %d.", math.MaxInt32))
	grpcFs.Int32Var(&o.InitialConnectionWindowSize, "grpc-initial-connection-window-size", o.InitialConnectionWindowSize,
		fmt.Sprintf("The initial HTTP/2 connection-level flow control window size in bytes. Set to 0 to use gRPC default (65535). Must be between 65535 and %d.", math.MaxInt32))

	grpcFs.DurationVar(&o.MaxConnectionAge, "grpc-max-connection-age", o.MaxConnectionAge,
		"The maximum amount of time a connection may exist before being closed by the server. Must be at least 10s.")
	grpcFs.DurationVar(&o.MaxConnectionAgeGrace, "grpc-max-connection-age-grace", o.MaxConnectionAgeGrace,
		"An additive period after the maximum connection age after which the connection will be forcible closed by the server. Must be at least 5s.")
	grpcFs.DurationVar(&o.MaxConnectionIdle, "grpc-max-connection-idle", o.MaxConnectionIdle,
		"The maximum amount of time a connection may be idle before being closed by the server. Must be at least 5s or set to 0 to disable (infinity).")
	grpcFs.DurationVar(&o.KeepAliveTime, "grpc-keepalive-time", o.KeepAliveTime,
		"Duration after which a keepalive probe is sent. Must be at least 10s.")
	grpcFs.DurationVar(&o.KeepAliveTimeout, "grpc-keepalive-timeout", o.KeepAliveTimeout,
		"Duration the server waits for a keepalive response. Must be between 1s and 5m.")
	grpcFs.DurationVar(&o.MinPingInterval, "grpc-min-ping-interval", o.MinPingInterval,
		"The minimum amount of time a client should wait before sending a keepalive ping. Must be at least 5s.")
}

func (o *Options) Complete() (CompletedOptions, error) {
	if o == nil {
		return CompletedOptions{completedOptions: &completedOptions{}}, nil
	}

	completed := completedOptions{
		Options: *o,
	}

	if o.MaxConcurrentStreams == 0 {
		completed.MaxConcurrentStreams = 100
	}

	if o.MaxRecvMsgSize == 0 {
		completed.MaxRecvMsgSize = 4194304 // 4MiB
	}

	if o.MaxSendMsgSize == 0 {
		completed.MaxSendMsgSize = math.MaxInt32
	}

	if o.WriteBufferSize == 0 {
		completed.WriteBufferSize = 32768 // 32KiB
	}

	if o.ReadBufferSize == 0 {
		completed.ReadBufferSize = 32768 // 32KiB
	}

	if o.InitialWindowSize == 0 {
		completed.InitialWindowSize = 65535
	}

	if o.InitialConnectionWindowSize == 0 {
		completed.InitialConnectionWindowSize = 65535
	}

	if o.KeepAliveTime == 0 {
		completed.KeepAliveTime = 1 * time.Minute
	}

	if o.KeepAliveTimeout == 0 {
		completed.KeepAliveTimeout = 10 * time.Second
	}

	if o.MinPingInterval == 0 {
		completed.MinPingInterval = 5 * time.Second
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

	if o.MaxConcurrentStreams < 1 || o.MaxConcurrentStreams > 10000 {
		allErrors = append(allErrors, fmt.Errorf("--grpc-max-streams-per-connection %d: must be between 1 and 10000", o.MaxConcurrentStreams))
	}

	if o.MaxRecvMsgSize < 1024 {
		allErrors = append(allErrors, fmt.Errorf("--grpc-max-recv-msg-size %d: must be at least 1024", o.MaxRecvMsgSize))
	}

	if o.MaxSendMsgSize < 1024 {
		allErrors = append(allErrors, fmt.Errorf("--grpc-max-send-msg-size %d: must be at least 1024", o.MaxSendMsgSize))
	}

	if o.WriteBufferSize < 0 || o.WriteBufferSize > math.MaxInt32 {
		allErrors = append(allErrors, fmt.Errorf("--grpc-write-buffer-size %d: must be between 0 and %d", o.WriteBufferSize, math.MaxInt32))
	}

	if o.ReadBufferSize < 0 || o.ReadBufferSize > math.MaxInt32 {
		allErrors = append(allErrors, fmt.Errorf("--grpc-read-buffer-size %d: must be between 0 and %d", o.ReadBufferSize, math.MaxInt32))
	}

	if o.InitialWindowSize < 65535 || o.InitialWindowSize > math.MaxInt32 {
		allErrors = append(allErrors, fmt.Errorf("--grpc-initial-window-size %d: must be between 65535 and %d", o.InitialWindowSize, math.MaxInt32))
	}

	if o.InitialConnectionWindowSize < 65535 || o.InitialConnectionWindowSize > math.MaxInt32 {
		allErrors = append(allErrors, fmt.Errorf("--grpc-initial-connection-window-size %d: must be between 65535 and %d", o.InitialConnectionWindowSize, math.MaxInt32))
	}

	if o.MaxConnectionAge < 10*time.Second {
		allErrors = append(allErrors, fmt.Errorf("--grpc-max-connection-age %v: must be at least 10s", o.MaxConnectionAge))
	}

	if o.MaxConnectionAgeGrace < 5*time.Second {
		allErrors = append(allErrors, fmt.Errorf("--grpc-max-connection-age-grace %v: must be at least 5s", o.MaxConnectionAgeGrace))
	}

	if o.MaxConnectionAgeGrace > o.MaxConnectionAge {
		allErrors = append(allErrors, fmt.Errorf("--grpc-max-connection-age-grace %v: must be less than --grpc-max-connection-age %v", o.MaxConnectionAgeGrace, o.MaxConnectionAge))
	}

	if o.MaxConnectionIdle != 0 && o.MaxConnectionIdle < 5*time.Second {
		allErrors = append(allErrors, fmt.Errorf("--grpc-max-connection-idle %v: must be at least 5s", o.MaxConnectionIdle))
	}

	if o.MaxConnectionIdle > o.MaxConnectionAge {
		allErrors = append(allErrors, fmt.Errorf("--grpc-max-connection-idle %v: must be less than --grpc-max-connection-age %v", o.MaxConnectionIdle, o.MaxConnectionAge))
	}

	if o.KeepAliveTime < 10*time.Second {
		allErrors = append(allErrors, fmt.Errorf("--grpc-keepalive-time %v: must be 10s or greater", o.KeepAliveTime))
	}

	if o.KeepAliveTimeout < 1*time.Second || o.KeepAliveTimeout > 5*time.Minute {
		allErrors = append(allErrors, fmt.Errorf("--grpc-keepalive-timeout %v: must be between 1s and 5m", o.KeepAliveTimeout))
	}

	if o.KeepAliveTimeout >= o.KeepAliveTime {
		allErrors = append(allErrors, fmt.Errorf("--grpc-keepalive-timeout %v: must be less than --grpc-keepalive-time %v", o.KeepAliveTimeout, o.KeepAliveTime))
	}

	if o.MinPingInterval < 5*time.Second {
		allErrors = append(allErrors, fmt.Errorf("--grpc-min-ping-interval %v: must be at least 5s", o.MinPingInterval))
	}

	return allErrors
}

// ApplyTo applies the completed gRPC options to the given slice of server options.
func (o *CompletedOptions) ApplyTo(serverOpts *[]grpc.ServerOption) error {
	if o == nil {
		return nil
	}

	*serverOpts = append(*serverOpts,
		grpc.MaxConcurrentStreams(o.MaxConcurrentStreams),
		grpc.MaxRecvMsgSize(o.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(o.MaxSendMsgSize),
		grpc.WriteBufferSize(o.WriteBufferSize),
		grpc.SharedWriteBuffer(o.SharedWriteBuffer),
		grpc.ReadBufferSize(o.ReadBufferSize),
		grpc.InitialWindowSize(o.InitialWindowSize),
		grpc.InitialConnWindowSize(o.InitialConnectionWindowSize),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge:      o.MaxConnectionAge,
			MaxConnectionAgeGrace: o.MaxConnectionAgeGrace,
			MaxConnectionIdle:     o.MaxConnectionIdle,
			Time:                  o.KeepAliveTime,
			Timeout:               o.KeepAliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             o.MinPingInterval,
			PermitWithoutStream: o.PermitWithoutStream,
		}),
	)

	return nil
}
