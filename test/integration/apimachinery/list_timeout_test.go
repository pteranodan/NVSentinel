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

//go:build integration

package apimachinery_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/nvidia/nvsentinel/test/integration/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestListTimeout(t *testing.T) {
	testCases := []struct {
		name        string
		storageType string
	}{
		{name: "OnDisk", storageType: apistorage.StorageTypeETCD3},
		{name: "InMemory", storageType: "memory"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			clientset, teardown := framework.SetupServer(t, tc.storageType)
			defer teardown()

			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Nanosecond)
			defer cancel()

			_, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})

			if err == nil {
				t.Fatal("list succeeded despite 100ns context timeout, expected 'DeadlineExceeded' error")
			}

			errStr := strings.ToLower(err.Error())
			validError := strings.Contains(errStr, "deadline exceeded") ||
				strings.Contains(errStr, "context canceled") ||
				strings.Contains(errStr, "timeout")

			if !validError {
				t.Errorf("expected timeout related error, got: %v", err)
			}
		})
	}
}
