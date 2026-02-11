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

package errors

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/storage"
)

func TestNewGRPCError(t *testing.T) {
	resource := "gpus"
	name := "test-gpu"

	tests := []struct {
		name         string
		input        error
		expectedCode codes.Code
	}{
		{
			name:         "storage not found",
			input:        storage.NewKeyNotFoundError("key", 0),
			expectedCode: codes.NotFound,
		},
		{
			name:         "storage conflict",
			input:        storage.NewResourceVersionConflictsError("key", 0),
			expectedCode: codes.Aborted,
		},
		{
			name:         "storage already exists",
			input:        storage.NewKeyExistsError("key", 0),
			expectedCode: codes.AlreadyExists,
		},
		{
			name:         "passthrough grpc error",
			input:        status.Error(codes.DataLoss, "custom error"),
			expectedCode: codes.DataLoss,
		},
		{
			name:         "unrecognized error to internal",
			input:        errors.New("raw mystery error"),
			expectedCode: codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := NewGRPCError(tt.input, resource, name)
			st, ok := status.FromError(gotErr)
			if !ok {
				t.Fatal("expected grpc status error")
			}
			if st.Code() != tt.expectedCode {
				t.Errorf("expected code %v, got %v", tt.expectedCode, st.Code())
			}
		})
	}
}

func TestNewAPIError(t *testing.T) {
	resource := "gpus"
	name := "test-gpu"

	tests := []struct {
		name     string
		input    error
		reason   string
		expected func(error) bool
	}{
		{
			name:     "grpc not found to api not found",
			input:    status.Error(codes.NotFound, "not found"),
			reason:   "StatusReasonNotFound (404)",
			expected: apierrors.IsNotFound,
		},
		{
			name:     "grpc aborted to api conflict",
			input:    status.Error(codes.Aborted, "conflict"),
			reason:   "StatusReasonConflict (409)",
			expected: apierrors.IsConflict,
		},
		{
			name:     "grpc invalid argument to api bad request",
			input:    status.Error(codes.InvalidArgument, "bad data"),
			reason:   "StatusReasonBadRequest (400)",
			expected: apierrors.IsBadRequest,
		},
		{
			name:     "grpc unavailable to api service unavailable",
			input:    status.Error(codes.Unavailable, "down"),
			reason:   "StatusReasonServiceUnavailable (503)",
			expected: apierrors.IsServiceUnavailable,
		},
		{
			name:     "raw error passthrough",
			input:    errors.New("not a status"),
			reason:   "original error string",
			expected: func(err error) bool { return err.Error() == "not a status" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := NewAPIError(tt.input, resource, name)
			if !tt.expected(gotErr) {
				t.Errorf("expected %s, got: %v", tt.reason, gotErr)
			}
		})
	}
}
