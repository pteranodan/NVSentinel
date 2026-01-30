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
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/pflag"
)

func TestAddFlags(t *testing.T) {
	fs := pflag.NewFlagSet("test", pflag.PanicOnError)
	o := NewOptions()
	o.AddFlags(fs)

	args := []string{
		"--bind-address=unix:///tmp/test.sock",
		"--max-streams-per-connection=500",
		"--max-recv-msg-size=1048576",
		"--max-send-msg-size=2097152",
		"--grpc-keepalive-time=30s",
		"--grpc-keepalive-timeout=5s",
	}
	fs.Parse(args)

	expected := &Options{
		BindAddress:          "unix:///tmp/test.sock",
		MaxConcurrentStreams: 500,
		MaxRecvMsgSize:       1048576,
		MaxSendMsgSize:       2097152,
		MaxConnectionIdle:    5 * time.Minute,
		KeepAliveTime:        30 * time.Second,
		KeepAliveTimeout:     5 * time.Second,
		MinPingInterval:      5 * time.Second,
		PermitWithoutStream:  true,
	}

	if !reflect.DeepEqual(expected, o) {
		t.Errorf("Difference detected:\n%s", cmp.Diff(expected, o))
	}
}

func TestComplete(t *testing.T) {
	testCases := []struct {
		name     string
		input    *Options
		expected Options
	}{
		{
			name:     "returns empty struct when nil",
			input:    nil,
			expected: Options{},
		},
		{
			name:  "defaults empty options",
			input: &Options{},
			expected: Options{
				BindAddress:          "unix:///var/run/nvidia-device-api/device-api.sock",
				MaxConcurrentStreams: 250,
				MaxRecvMsgSize:       4194304,
				MaxSendMsgSize:       16777216,
				MaxConnectionIdle:    5 * time.Minute,
				KeepAliveTime:        1 * time.Minute,
				KeepAliveTimeout:     10 * time.Second,
				MinPingInterval:      5 * time.Second,
				PermitWithoutStream:  true,
			},
		},
		{
			name: "enforces permit without stream",
			input: &Options{
				PermitWithoutStream: false,
			},
			expected: Options{
				BindAddress:          "unix:///var/run/nvidia-device-api/device-api.sock",
				MaxConcurrentStreams: 250,
				MaxRecvMsgSize:       4194304,
				MaxSendMsgSize:       16777216,
				MaxConnectionIdle:    5 * time.Minute,
				KeepAliveTime:        1 * time.Minute,
				KeepAliveTimeout:     10 * time.Second,
				MinPingInterval:      5 * time.Second,
				PermitWithoutStream:  true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			completed, _ := tc.input.Complete()
			if tc.input == nil {
				if completed.completedOptions != nil {
					t.Error("internal pointer must be nil when input is nil")
				}
				return
			}
			if !reflect.DeepEqual(tc.expected, completed.Options) {
				t.Errorf("Difference:\n%s", cmp.Diff(tc.expected, completed.Options))
			}
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name        string
		modify      func(*Options)
		expectError bool
	}{
		{
			name:        "default options are valid",
			modify:      func(o *Options) {},
			expectError: false,
		},
		{
			name: "scheme must be 'unix://'",
			modify: func(o *Options) {
				o.BindAddress = "tcp://127.0.0.1:8080"
			},
			expectError: true,
		},
		{
			name: "path cannot be relative",
			modify: func(o *Options) {
				o.BindAddress = "unix://relative.sock"
			},
			expectError: true,
		},
		{
			name: "keepalive timeout greater than time",
			modify: func(o *Options) {
				o.KeepAliveTime = 10 * time.Second
				o.KeepAliveTimeout = 20 * time.Second
			},
			expectError: true,
		},
		{
			name: "permit without stream must be true",
			modify: func(o *Options) {
				o.PermitWithoutStream = false
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewOptions()
			tc.modify(o)
			errs := o.Validate()
			if tc.expectError && len(errs) == 0 {
				t.Error("Error expected")
			}
			if !tc.expectError && len(errs) > 0 {
				t.Errorf("Unexpected error: %v", errs)
			}
		})
	}
}
