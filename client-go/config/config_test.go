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

package config

import (
	"testing"
)

func TestNewDefaultConfig(t *testing.T) {
	tests := []struct {
		name         string
		targetArg    string
		envVarValue  string
		expectTarget string
	}{
		{
			name:         "Returns target from argument if provided",
			targetArg:    "unix:///tmp/arg.sock",
			envVarValue:  "unix:///tmp/env.sock",
			expectTarget: "unix:///tmp/arg.sock",
		},
		{
			name:         "Returns target from environment variable if target argument is empty",
			targetArg:    "",
			envVarValue:  "unix:///tmp/env.sock",
			expectTarget: "unix:///tmp/env.sock",
		},
		{
			name:         "Returns default target if target argument and environment variable are empty",
			targetArg:    "",
			envVarValue:  "",
			expectTarget: DefaultNvidiaDeviceAPISocket,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(NvidiaDeviceAPITargetEnvVar, tt.envVarValue)

			cfg, err := NewDefaultConfig(tt.targetArg)
			if err != nil {
				t.Fatalf("NewDefaultConfig() returned unexpected error: %v", err)
			}

			if cfg.Target != tt.expectTarget {
				t.Errorf("Target mismatch: got=%v, want=%v", cfg.Target, tt.expectTarget)
			}
			if cfg.UserAgent != DefaultNvidiaUserAgent {
				t.Errorf("UserAgent mismatch: got=%v, want=%v", cfg.UserAgent, DefaultNvidiaUserAgent)
			}
			if cfg.TokenSource != nil {
				t.Errorf("TokenSource mismatch: got=%v, want=nil", cfg.TokenSource)
			}
			if cfg.KeepAliveTime != DefaultKeepAliveTime {
				t.Errorf("KeepAliveTime mismatch: got=%v, want=%v", cfg.KeepAliveTime, DefaultKeepAliveTime)
			}
			if cfg.KeepAliveTimeout != DefaultKeepAliveTimeout {
				t.Errorf("KeepAliveTimeout mismatch: got=%v, want=%v", cfg.KeepAliveTimeout, DefaultKeepAliveTimeout)
			}
			if cfg.IdleTimeout != DefaultIdleTimeout {
				t.Errorf("IdleTimeout mismatch: got=%v, want=%v", cfg.IdleTimeout, DefaultIdleTimeout)
			}

			// Ensure logger is usable and does not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Logger caused a panic: %v", r)
					}
				}()
				cfg.Logger.V(1).Info("Test log message")
			}()
		})
	}
}
