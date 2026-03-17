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
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
)

const (
	UnsupportedMediaTypePrefix = "unsupported media type:"
	UnprocessableContextPrefix = "unprocessable content:"
)

// NewAPIError maps a gRPC status error into a standard Kubernetes API error.
func NewAPIError(err error, resource string, name string) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return err
	}

	gr := schema.GroupResource{Group: "device.nvidia.com", Resource: resource}

	switch st.Code() {
	case codes.NotFound:
		return apierrors.NewNotFound(gr, name)
	case codes.AlreadyExists:
		return apierrors.NewAlreadyExists(gr, name)
	case codes.InvalidArgument:
		if strings.HasPrefix(st.Message(), UnsupportedMediaTypePrefix) {
			msg := strings.TrimSpace(strings.TrimPrefix(st.Message(), UnsupportedMediaTypePrefix))
			return NewUnsupportedMediaType(msg)
		}
		if strings.HasPrefix(st.Message(), UnprocessableContextPrefix) {
			msg := strings.TrimSpace(strings.TrimPrefix(st.Message(), UnprocessableContextPrefix))
			return NewUnprocessableContentType(msg)
		}
		return apierrors.NewBadRequest(st.Message())
	case codes.OutOfRange:
		return apierrors.NewResourceExpired(st.Message())
	case codes.Aborted:
		return apierrors.NewConflict(gr, name, errors.New(st.Message()))
	case codes.Unavailable, codes.DeadlineExceeded:
		return apierrors.NewServiceUnavailable(st.Message())
	case codes.Internal:
		return apierrors.NewInternalError(err)
	}

	return err
}

// NewGRPCError maps internal storage or API machinery errors to gRPC status codes.
func NewGRPCError(err error, resource string, name string) error {
	if err == nil {
		return nil
	}

	if st, ok := status.FromError(err); ok {
		return st.Err()
	}

	// K8s API errors
	if apierrors.IsInvalid(err) {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if apierrors.IsBadRequest(err) {
		if statusErr, ok := err.(apierrors.APIStatus); ok {
			return status.Error(codes.InvalidArgument, statusErr.Status().Message)
		}
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}
	if apierrors.IsResourceExpired(err) || apierrors.IsGone(err) {
		return status.Errorf(codes.OutOfRange, "%v", err)
	}

	// Storage errors
	if storage.IsNotFound(err) {
		return status.Errorf(codes.NotFound, "%s %q not found", resource, name)
	}
	if storage.IsExist(err) {
		return status.Errorf(codes.AlreadyExists, "%s %q already exists", resource, name)
	}
	if storage.IsConflict(err) {
		return status.Errorf(codes.Aborted, "Operation cannot be fulfilled on %s %q: the object has been modified; please apply your changes to the latest version and try again", resource, name)
	}
	// Kine/storage can wrap precondition failures as InvalidObj.
	if storage.IsInvalidObj(err) && strings.Contains(strings.ToLower(err.Error()), "precondition") {
		return status.Errorf(codes.Aborted, "Operation cannot be fulfilled on %s %q: the object has been modified; please apply your changes to the latest version and try again", resource, name)
	}
	if storage.IsInvalidObj(err) {
		return status.Errorf(codes.InvalidArgument, "Internal error occurred: stored object is invalid: %v", err)
	}
	if storage.IsInvalidError(err) {
		return status.Errorf(codes.OutOfRange, "Too old resource version: %v", err)
	}
	if storage.IsUnreachable(err) || storage.IsRequestTimeout(err) {
		return status.Error(codes.Unavailable, "The server is currently unable to handle the request: storage is unreachable")
	}

	return status.Error(codes.Internal, "Internal error occurred: unexpected error")
}

func NewUnsupportedMediaType(message string) *apierrors.StatusError {
	return &apierrors.StatusError{metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    415,
		Reason:  metav1.StatusReasonUnsupportedMediaType,
		Message: message,
	}}
}

func NewUnprocessableContentType(message string) *apierrors.StatusError {
	return &apierrors.StatusError{metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    422,
		Reason:  metav1.StatusReasonInvalid,
		Message: message,
	}}
}
