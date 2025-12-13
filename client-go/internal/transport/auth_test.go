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

package transport

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenPerRPCCredentials_RequireTransportSecurity(t *testing.T) {
	src := NewStaticTokenSource("token")
	creds := NewTokenPerRPCCredentials(src)

	if creds.RequireTransportSecurity() != false {
		t.Errorf("RequireTransportSecurity() = true, want false")
	}
}

func TestStaticTokenSource(t *testing.T) {
	const expected = "static-token-123"

	src := NewStaticTokenSource(expected)
	creds := NewTokenPerRPCCredentials(src)

	md, err := creds.GetRequestMetadata(context.Background())
	if err != nil {
		t.Fatalf("GetRequestMetadata() failed: %v", err)
	}

	if auth, ok := md["authorization"]; !ok || auth != "Bearer "+expected {
		t.Errorf("authorization header = %q, want %q", auth, "Bearer "+expected)
	}
}

func TestDynamicTokenSource(t *testing.T) {
	const expected = "dynamic-token-123"

	tests := []struct {
		name      string
		token     string
		err       error
		wantAuth  string
		expectErr bool
	}{
		{"valid token", expected, nil, "Bearer " + expected, false},
		{"provider error", "", errors.New("fetch failed"), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int32
			provider := func(ctx context.Context) (string, time.Time, error) {
				atomic.AddInt32(&calls, 1)
				return tt.token, time.Now().Add(time.Minute), tt.err
			}

			src := NewDynamicTokenSource(provider)
			creds := NewTokenPerRPCCredentials(src)

			md, err := creds.GetRequestMetadata(context.Background())
			if tt.expectErr {
				if err == nil || err.Error() != tt.err.Error() {
					t.Errorf("error = %v, want %v", err, tt.err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if md["authorization"] != tt.wantAuth {
					t.Errorf("authorization header = %q, want %q", md["authorization"], tt.wantAuth)
				}
			}

			if atomic.LoadInt32(&calls) != 1 {
				t.Errorf("provider called %d times, want 1", calls)
			}
		})
	}
}

func TestCachedTokenSource(t *testing.T) {
	var calls int32
	now := time.Now()

	provider := func(ctx context.Context) (string, time.Time, error) {
		atomic.AddInt32(&calls, 1)
		return "cached-token", now.Add(time.Minute), nil
	}

	src := NewCachedTokenSource(NewDynamicTokenSource(provider))

	// First call fetches token
	tok, exp, err := src.Token(context.Background())
	if err != nil || tok != "cached-token" || exp != now.Add(time.Minute) {
		t.Fatalf("first Token() = %v, %v, %v; want cached-token, expiry=%v, nil", tok, exp, err, now.Add(time.Minute))
	}

	// Second call should use cache
	tok2, _, err2 := src.Token(context.Background())
	if err2 != nil || tok2 != "cached-token" {
		t.Fatalf("second Token() = %v, %v; want cached-token, nil", tok2, err2)
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("provider called %d times, want 1", calls)
	}
}

func TestCachedTokenSource_ExpiredToken(t *testing.T) {
	expiredProvider := func(ctx context.Context) (string, time.Time, error) {
		return "token", time.Now().Add(-time.Minute), nil
	}

	src := NewCachedTokenSource(NewDynamicTokenSource(expiredProvider))
	_, _, err := src.Token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Errorf("expected expired token error, got %v", err)
	}
}

func TestCachedTokenSource_ConcurrentAccess(t *testing.T) {
	const expectedToken = "token-123"
	const workers = 10

	var calls int32
	provider := func(ctx context.Context) (string, time.Time, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(10 * time.Millisecond) // simulate delay
		return expectedToken, time.Now().Add(time.Minute), nil
	}

	src := NewCachedTokenSource(NewDynamicTokenSource(provider))

	done := make(chan struct{})
	for i := 0; i < workers; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			token, _, err := src.Token(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if token != expectedToken {
				t.Errorf("token mismatch: got %q, want %q", token, expectedToken)
			}
		}()
	}

	for i := 0; i < workers; i++ {
		<-done
	}

	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("provider called %d times, want 1", calls)
	}
}
