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
			name:         "Explicit Argument overrides everything",
			targetArg:    "unix:///tmp/explicit.sock",
			envVarValue:  "unix:///tmp/env.sock",
			expectTarget: "unix:///tmp/explicit.sock",
		},
		{
			name:         "Env Var used when argument is empty",
			targetArg:    "",
			envVarValue:  "unix:///tmp/env.sock",
			expectTarget: "unix:///tmp/env.sock",
		},
		{
			name:         "Default used when arg and env are empty",
			targetArg:    "",
			envVarValue:  "",
			expectTarget: DefaultSocketPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(KubernetesDeviceApiTargetEnvVar, tt.envVarValue)

			cfg, err := NewDefaultConfig(tt.targetArg)
			if err != nil {
				t.Fatalf("NewDefaultConfig() unexpected error: %v", err)
			}

			if cfg.Target != tt.expectTarget {
				t.Errorf("Target = %v, want %v", cfg.Target, tt.expectTarget)
			}

			if cfg.UserAgent != DefaultUserAgent {
				t.Errorf("UserAgent = %v, want %v", cfg.UserAgent, DefaultUserAgent)
			}
			if cfg.KeepAliveTime != DefaultKeepAliveTime {
				t.Errorf("KeepAliveTime = %v, want %v", cfg.KeepAliveTime, DefaultKeepAliveTime)
			}
			if cfg.KeepAliveTimeout != DefaultKeepAliveTimeout {
				t.Errorf("KeepAliveTimeout = %v, want %v", cfg.KeepAliveTimeout, DefaultKeepAliveTimeout)
			}
			if cfg.IdleTimeout != DefaultIdleTimeout {
				t.Errorf("IdleTimeout = %v, want %v", cfg.IdleTimeout, DefaultIdleTimeout)
			}

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
