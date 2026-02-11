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

package apimachinery_test

import (
	"context"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	_, err := clientset.DeviceV1alpha1().GPUs().List(ctx, metav1.ListOptions{})

	if err == nil {
		t.Fatal("List succeeded despite 1ns context timeout, expected DeadlineExceeded error")
	}

	errStr := err.Error()
	if !strings.Contains(errStr, "deadline exceeded") && !strings.Contains(errStr, "context canceled") && !strings.Contains(errStr, "timeout") {
		t.Errorf("Expected timeout error, got: %v", err)
	}
}
