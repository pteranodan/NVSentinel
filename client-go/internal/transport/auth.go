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
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc/credentials"
)

// TokenSource provides access to an authentication token.
type TokenSource interface {
	// Token returns the current token and its expiration time.
	// Returns an error if the token cannot be retrieved.
	Token(ctx context.Context) (string, time.Time, error)
}

// staticTokenSource always returns the same token.
type staticTokenSource struct {
	token string
}

// NewStaticTokenSource returns a TokenSource that always returns the given token.
func NewStaticTokenSource(token string) TokenSource {
	return &staticTokenSource{token: token}
}

func (s *staticTokenSource) Token(ctx context.Context) (string, time.Time, error) {
	return s.token, time.Time{}, nil
}

// dynamicTokenSource calls a provider function to fetch a token.
type dynamicTokenSource struct {
	provider func(ctx context.Context) (string, time.Time, error)
}

// NewDynamicTokenSource returns a TokenSource that fetches a token from the given provider.
func NewDynamicTokenSource(provider func(ctx context.Context) (string, time.Time, error)) TokenSource {
	return &dynamicTokenSource{provider: provider}
}

func (d *dynamicTokenSource) Token(ctx context.Context) (string, time.Time, error) {
	return d.provider(ctx)
}

// CachedTokenSource wraps another TokenSource and caches tokens until expiry.
type CachedTokenSource struct {
	source TokenSource
	mu     sync.RWMutex
	token  string
	expiry time.Time
}

// NewCachedTokenSource returns a TokenSource that caches tokens from the given source.
func NewCachedTokenSource(source TokenSource) TokenSource {
	return &CachedTokenSource{source: source}
}

func (c *CachedTokenSource) Token(ctx context.Context) (string, time.Time, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.expiry) {
		token := c.token
		expiry := c.expiry
		c.mu.RUnlock()
		return token, expiry, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// double-check after acquiring write lock
	if c.token != "" && time.Now().Before(c.expiry) {
		return c.token, c.expiry, nil
	}

	t, exp, err := c.source.Token(ctx)
	if err != nil {
		return "", time.Time{}, err
	}
	if !exp.IsZero() && !exp.After(time.Now()) {
		return "", time.Time{}, fmt.Errorf("token expired (expiry=%v)", exp)
	}

	c.token = t
	c.expiry = exp

	return t, exp, nil
}

// TokenPerRPCCredentials wraps a TokenSource to implement grpc.PerRPCCredentials.
type TokenPerRPCCredentials struct {
	source TokenSource
}

// NewTokenPerRPCCredentials creates grpc credentials from a TokenSource.
func NewTokenPerRPCCredentials(source TokenSource) credentials.PerRPCCredentials {
	return &TokenPerRPCCredentials{source: source}
}

// GetRequestMetadata returns the current token as an authorization header.
func (t *TokenPerRPCCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	token, _, err := t.source.Token(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]string{
		"authorization": "Bearer " + token,
	}, nil
}

// RequireTransportSecurity returns false, indicating TLS is not required.
func (t *TokenPerRPCCredentials) RequireTransportSecurity() bool {
	return false
}
