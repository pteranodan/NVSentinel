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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/storage"
)

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
		return apierrors.NewBadRequest(st.Message())
	case codes.Aborted:
		return apierrors.NewConflict(gr, name, errors.New(st.Message()))
	case codes.Unavailable, codes.DeadlineExceeded:
		return apierrors.NewServiceUnavailable(st.Message())
	case codes.Internal:
		return apierrors.NewInternalError(err)
	}

	return err
}

func NewGRPCError(err error, resource string, name string) error {
	if err == nil {
		return nil
	}

	if st, ok := status.FromError(err); ok {
		return st.Err()
	}

	if storage.IsNotFound(err) {
		return status.Errorf(codes.NotFound, "%s %q not found", resource, name)
	}
	if storage.IsExist(err) {
		return status.Errorf(codes.AlreadyExists, "%s %q already exists", resource, name)
	}
	if storage.IsConflict(err) {
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
	if apierrors.IsInvalid(err) || apierrors.IsBadRequest(err) {
		return status.Errorf(codes.InvalidArgument, "%v", err)
	}

	return status.Error(codes.Internal, "Internal error occurred: unexpected error")
}
