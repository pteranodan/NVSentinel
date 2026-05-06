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

package errutil

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// IsTemporaryError checks if the error is a temporary network error that should be retried
func IsTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	return IsContextError(err) ||
		IsKubernetesAPIError(err) ||
		IsNetworkError(err) ||
		IsSyscallError(err) ||
		IsStringBasedError(err) ||
		errors.Is(err, io.EOF) ||
		strings.Contains(err.Error(), "EOF")
}

// IsContextError checks if the error is a context-related error that should be retried
func IsContextError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// IsKubernetesAPIError checks if the error is a Kubernetes API error that should be retried
func IsKubernetesAPIError(err error) bool {
	return apierrors.IsTimeout(err) ||
		apierrors.IsServerTimeout(err) ||
		apierrors.IsServiceUnavailable(err) ||
		apierrors.IsTooManyRequests(err) ||
		apierrors.IsInternalError(err)
}

// IsNetworkError checks if the error is a network-related error that should be retried
func IsNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	return false
}

// IsSyscallError checks if the error is a syscall error that should be retried
func IsSyscallError(err error) bool {
	return errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.ETIMEDOUT) ||
		errors.Is(err, syscall.EHOSTUNREACH) ||
		errors.Is(err, syscall.ENETUNREACH) ||
		errors.Is(err, syscall.EPIPE)
}

// IsStringBasedError checks if the error message contains retryable error patterns
func IsStringBasedError(err error) bool {
	errStr := err.Error()

	return IsHTTPConnectionError(errStr) ||
		IsTLSError(errStr) ||
		IsDNSError(errStr) ||
		IsLoadBalancerError(errStr) ||
		IsKubernetesStringError(errStr)
}

// IsHTTPConnectionError checks for HTTP/2 and HTTP connection error patterns
func IsHTTPConnectionError(errStr string) bool {
	httpErrors := []string{
		"http2: client connection lost",
		"http2: server connection lost",
		"http2: connection closed",
		"connection reset by peer",
		"broken pipe",
		"connection refused",
		"connection timed out",
		"i/o timeout",
		"network is unreachable",
		"host is unreachable",
	}

	for _, pattern := range httpErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// IsTLSError checks for TLS/SSL handshake error patterns
func IsTLSError(errStr string) bool {
	tlsErrors := []string{
		"tls: handshake timeout",
		"tls: oversized record received",
		"remote error: tls:",
	}

	for _, pattern := range tlsErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// IsDNSError checks for DNS resolution error patterns
func IsDNSError(errStr string) bool {
	dnsErrors := []string{
		"no such host",
		"dns: no answer",
		"temporary failure in name resolution",
	}

	for _, pattern := range dnsErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// IsLoadBalancerError checks for load balancer and proxy error patterns
func IsLoadBalancerError(errStr string) bool {
	lbErrors := []string{
		"502 Bad Gateway",
		"503 Service Unavailable",
		"504 Gateway Timeout",
	}

	for _, pattern := range lbErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// IsKubernetesStringError checks for Kubernetes-specific error patterns
func IsKubernetesStringError(errStr string) bool {
	k8sErrors := []string{
		"the server is currently unable to handle the request",
		"etcd cluster is unavailable",
		"unable to connect to the server",
		"server is not ready",
	}

	for _, pattern := range k8sErrors {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}
