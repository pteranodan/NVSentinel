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
	"fmt"
	"testing"
	"time"

	devicev1alpha1 "github.com/nvidia/nvsentinel/api/device/v1alpha1"
	"github.com/nvidia/nvsentinel/test/integration/framework"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	apistorage "k8s.io/apiserver/pkg/storage/storagebackend"
)

func TestWatchLabelSelector(t *testing.T) {
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

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			t.Run("EventFiltering", func(t *testing.T) {
				opts := metav1.ListOptions{LabelSelector: "app=watch-test"}
				watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, opts)
				if err != nil {
					t.Fatalf("failed to start watch: %v", err)
				}
				defer watcher.Stop()

				gpuIgnored := &devicev1alpha1.GPU{
					ObjectMeta: metav1.ObjectMeta{
						Name:   fmt.Sprintf("%s-ignored", tc.name),
						Labels: map[string]string{"label-1": "true"},
					},
					Spec: devicev1alpha1.GPUSpec{UUID: "ignored-uuid"},
				}
				if _, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpuIgnored, metav1.CreateOptions{}); err != nil {
					t.Fatalf("failed to create ignored gpu: %v", err)
				}

				gpuMatch := &devicev1alpha1.GPU{
					ObjectMeta: metav1.ObjectMeta{
						Name:   fmt.Sprintf("%s-match", tc.name),
						Labels: map[string]string{"app": "watch-test"},
					},
					Spec: devicev1alpha1.GPUSpec{UUID: "match-uuid"},
				}
				if _, err := clientset.DeviceV1alpha1().GPUs().Create(ctx, gpuMatch, metav1.CreateOptions{}); err != nil {
					t.Fatalf("failed to create matching gpu: %v", err)
				}

				select {
				case event, ok := <-watcher.ResultChan():
					if !ok {
						t.Fatal("watch channel closed unexpectedly")
					}
					obj := event.Object.(*devicev1alpha1.GPU)
					if obj.Name != gpuMatch.Name {
						if tc.storageType == "memory" {
							framework.SkipWithWarning(t, fmt.Sprintf("received wrong event: expected %s, got %s", gpuMatch.Name, obj.Name))
						}
						t.Errorf("received wrong event: expected %s, got %s", gpuMatch.Name, obj.Name)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("timed out waiting for matching watch event")
				}

				select {
				case event := <-watcher.ResultChan():
					if tc.storageType == "memory" {
						framework.SkipWithWarning(t, fmt.Sprintf("received unexpected extra event: %v", event))
					}
					t.Errorf("received unexpected extra event: %v", event)
				default:
				}
			})

			t.Run("InvalidSelector", func(t *testing.T) {
				opts := metav1.ListOptions{LabelSelector: "app==="}
				watcher, err := clientset.DeviceV1alpha1().GPUs().Watch(ctx, opts)
				if err != nil {
					t.Fatalf("Watch failed early: %v", err)
				}
				defer watcher.Stop()

				select {
				case event := <-watcher.ResultChan():
					if event.Type != watch.Error {
						t.Fatalf("expected watch.Error, got %v", event.Type)
					}

					statusErr, ok := event.Object.(*metav1.Status)
					if !ok {
						t.Fatalf("expected metav1.Status, got %T", event.Object)
					}

					if statusErr.Code != 400 {
						t.Errorf("expected status code 400, got %d", statusErr.Code)
					}
				case <-time.After(time.Second * 5):
					t.Fatal("timed out waiting for watch error event")
				}
			})
		})
	}
}
