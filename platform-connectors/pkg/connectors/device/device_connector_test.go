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

package device

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nvidia/device-api/api/device/v1alpha1"
	"github.com/nvidia/device-api/client-go/clientset/device/fake"
	pb "github.com/nvidia/nvsentinel/data-models/pkg/protos"
	"github.com/nvidia/nvsentinel/platform-connectors/pkg/ringbuffer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clienttesting "k8s.io/client-go/testing"
)

type statusPatch struct {
	Status v1alpha1.GPUStatus `json:"status"`
}

func TestFetchAndProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	fakeGPU := &v1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: "gpu-0"},
	}

	fakeCS := fake.NewSimpleClientset(fakeGPU)
	rb := ringbuffer.NewRingBuffer(t.Name(), ctx)
	stopCh := make(chan struct{})

	connector := NewConnector(fakeCS, rb, stopCh)

	go connector.FetchAndProcessHealthMetric(ctx)

	eventPayload := &pb.HealthEvents{
		Events: []*pb.HealthEvent{{
			Id:        "integration-test-1",
			CheckName: "Thermal",
			IsHealthy: false,
			Message:   "Critical Temp",
			EntitiesImpacted: []*pb.Entity{
				{EntityType: "GPU", EntityValue: "GPU-0"},
			},
		}},
	}

	queuedItem := &ringbuffer.QueuedHealthEvents{
		Events: eventPayload,
	}

	rb.Enqueue(queuedItem)

	require.Eventually(t, func() bool {
		return len(fakeCS.Actions()) > 0
	}, 2*time.Second, 10*time.Millisecond, "Connector failed to process event from ring buffer")

	close(stopCh)
}

func TestFetchAndProcess_HighVolumeThroughput(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-0"}})
	rb := ringbuffer.NewRingBuffer(t.Name(), ctx)
	stopCh := make(chan struct{})
	connector := NewConnector(fakeCS, rb, stopCh)

	go connector.FetchAndProcessHealthMetric(ctx)

	for range 50 {
		rb.Enqueue(&ringbuffer.QueuedHealthEvents{
			Events: &pb.HealthEvents{
				Events: []*pb.HealthEvent{{
					CheckName:        "SerialStress",
					IsHealthy:        true,
					EntitiesImpacted: []*pb.Entity{{EntityType: "GPU", EntityValue: "GPU-0"}},
				}},
			},
		})
	}

	require.Eventually(t, func() bool {
		return len(fakeCS.Actions()) == 50
	}, 3*time.Second, 100*time.Millisecond)

	close(stopCh)
}

func TestFetchAndProcess_BatchDeduplication(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-0"}})
	rb := ringbuffer.NewRingBuffer(t.Name(), ctx)
	stopCh := make(chan struct{})
	connector := NewConnector(fakeCS, rb, stopCh)

	go connector.FetchAndProcessHealthMetric(ctx)

	now := time.Now()
	var events []*pb.HealthEvent
	for i := range 50 {
		events = append(events, &pb.HealthEvent{
			CheckName:          "BatchStress",
			IsHealthy:          i != 25,
			Message:            fmt.Sprintf("Message %d", i),
			EntitiesImpacted:   []*pb.Entity{{EntityType: "GPU", EntityValue: "GPU-0"}},
			GeneratedTimestamp: timestamppb.New(now.Add(time.Duration(i) * time.Second)),
		})
	}

	rb.Enqueue(&ringbuffer.QueuedHealthEvents{
		Events: &pb.HealthEvents{Events: events},
	})

	require.Eventually(t, func() bool {
		return len(fakeCS.Actions()) == 1
	}, 2*time.Second, 50*time.Millisecond)

	close(stopCh)

	patchAction := fakeCS.Actions()[0].(clienttesting.PatchAction)
	var p statusPatch
	err := json.Unmarshal(patchAction.GetPatch(), &p)
	require.NoError(t, err)

	require.Len(t, p.Status.Conditions, 1)
	cond := p.Status.Conditions[0]

	assert.Equal(t, metav1.ConditionFalse, cond.Status, "Status should reflect index 49 (Healthy)")
	assert.Equal(t, "Message 49", cond.Message, "Message should reflect index 49")
	expectedTime := now.Add(49 * time.Second).Truncate(time.Second)
	assert.True(t, cond.LastTransitionTime.Time.Equal(expectedTime) ||
		cond.LastTransitionTime.Time.After(expectedTime), "Timestamp should be index 49's time")
}

func TestConnector_GracefulStop(t *testing.T) {
	ctx := context.Background()
	rb := ringbuffer.NewRingBuffer(t.Name(), ctx)
	stopCh := make(chan struct{})
	connector := NewConnector(nil, rb, stopCh)

	loopExited := make(chan struct{})
	go func() {
		connector.FetchAndProcessHealthMetric(ctx)
		close(loopExited)
	}()

	close(stopCh)

	select {
	case <-loopExited:
	case <-time.After(1 * time.Second):
		t.Fatal("FetchAndProcessHealthMetric loop did not exit after stopCh was closed")
	}
}

func TestProcessHealthEvents_NormalizesToK8sNaming(t *testing.T) {
	//  Name must be DNS subdomain name
	gpuName := "gpu-636c7467-3136"
	fakeGPU := &v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: gpuName}}
	fakeCS := fake.NewSimpleClientset(fakeGPU)

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	healthEvents := &pb.HealthEvents{
		Events: []*pb.HealthEvent{
			{
				CheckName: "Memory",
				EntitiesImpacted: []*pb.Entity{
					// Entity has raw GPU UUID
					{EntityType: "GPU", EntityValue: "GPU-636C7467-3136"},
				},
			},
		},
	}

	err := connector.processHealthEvents(context.Background(), healthEvents)
	require.NoError(t, err)

	patch := fakeCS.Actions()[0].(clienttesting.PatchAction)
	assert.Equal(t, gpuName, patch.GetName(), "Connector must lowercase UUIDs for standard K8s API compatibility")
}

func TestProcessHealthEvents_GroupingIsCaseInsensitive(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-0"}})

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	healthEvents := &pb.HealthEvents{
		Events: []*pb.HealthEvent{
			{CheckName: "A", EntitiesImpacted: []*pb.Entity{{EntityType: "GPU", EntityValue: "GPU-0"}}},
			{CheckName: "B", EntitiesImpacted: []*pb.Entity{{EntityType: "GPU", EntityValue: "gpu-0"}}},
		},
	}

	err := connector.processHealthEvents(context.Background(), healthEvents)
	require.NoError(t, err)

	assert.Len(t, fakeCS.Actions(), 1, "Events with different casing for the same UUID should be merged")
}

func TestProcessHealthEvents_EmptyEntityValue(t *testing.T) {
	fakeCS := fake.NewSimpleClientset()

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	healthEvents := &pb.HealthEvents{
		Events: []*pb.HealthEvent{
			{
				CheckName: "EmptyTest",
				EntitiesImpacted: []*pb.Entity{
					{EntityType: "GPU", EntityValue: ""},
				},
			},
		},
	}

	err := connector.processHealthEvents(context.Background(), healthEvents)

	assert.NoError(t, err, "Should not return error with an empty name")
	assert.Empty(t, fakeCS.Actions(), "Should not have attempted a patch with an empty name")
}

func TestProcessHealthEvents_FilterStoreOnly(t *testing.T) {
	fakeGPU := &v1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{
			Name: "gpu-0",
		},
	}

	fakeCS := fake.NewSimpleClientset(fakeGPU)

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	healthEvents := &pb.HealthEvents{
		Events: []*pb.HealthEvent{
			{
				Id:                 "skip-me",
				ProcessingStrategy: pb.ProcessingStrategy_STORE_ONLY,
				EntitiesImpacted:   []*pb.Entity{{EntityType: "GPU", EntityValue: "GPU-0"}},
			},
			{
				Id:                 "process-me",
				ProcessingStrategy: pb.ProcessingStrategy_UNSPECIFIED,
				EntitiesImpacted:   []*pb.Entity{{EntityType: "GPU", EntityValue: "GPU-0"}},
				CheckName:          "FanSpeed",
				IsHealthy:          true,
			},
		},
	}

	err := connector.processHealthEvents(context.Background(), healthEvents)
	require.NoError(t, err)

	actions := fakeCS.Actions()
	assert.Equal(t, 1, len(actions), "Should have ignored the STORE_ONLY event")
}

func TestProcessHealthEvents_EntityTypeFiltering(t *testing.T) {
	gpuName := "gpu-0"
	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: gpuName},
	})

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	healthEvents := &pb.HealthEvents{
		Events: []*pb.HealthEvent{
			{
				Id:        "gpu-event",
				CheckName: "Memory",
				IsHealthy: false,
				EntitiesImpacted: []*pb.Entity{
					{EntityType: "GPU", EntityValue: "GPU-0"},
				},
			},
			{
				Id:        "cpu-event",
				CheckName: "CoreTemp",
				IsHealthy: false,
				EntitiesImpacted: []*pb.Entity{
					{EntityType: "CPU", EntityValue: "CPU-0"},
				},
			},
			{
				Id:        "nic-event",
				CheckName: "LinkDown",
				IsHealthy: false,
				EntitiesImpacted: []*pb.Entity{
					{EntityType: "NIC", EntityValue: "eth0"},
				},
			},
		},
	}

	err := connector.processHealthEvents(context.Background(), healthEvents)
	require.NoError(t, err)

	actions := fakeCS.Actions()
	assert.Equal(t, 1, len(actions), "Expected exactly one API call for the GPU entity")

	patchAction := actions[0].(clienttesting.PatchAction)
	assert.Equal(t, gpuName, patchAction.GetName(), "Patch should target the GPU name")
}

func TestProcessHealthEvents_NoApplicableEvents(t *testing.T) {
	fakeCS := fake.NewSimpleClientset()

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	healthEvents := &pb.HealthEvents{
		Events: []*pb.HealthEvent{
			{
				Id:                 "only-store",
				ProcessingStrategy: pb.ProcessingStrategy_STORE_ONLY,
				EntitiesImpacted:   []*pb.Entity{{EntityType: "GPU", EntityValue: "gpu-0"}},
			},
			{
				Id:               "only-cpu",
				EntitiesImpacted: []*pb.Entity{{EntityType: "CPU", EntityValue: "cpu-0"}},
			},
		},
	}

	err := connector.processHealthEvents(context.Background(), healthEvents)

	assert.NoError(t, err)
	assert.Empty(t, fakeCS.Actions(), "Should not perform any API actions if no applicable events")
}

func TestProcessGPUEvents_EmptyCheckName(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-0"}})

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	events := []*pb.HealthEvent{
		{
			CheckName: "",
			IsHealthy: true,
			EntitiesImpacted: []*pb.Entity{
				{EntityType: "GPU", EntityValue: "GPU-0"},
			},
		},
	}

	err := connector.processGPUEvents(context.Background(), "gpu-0", events)
	assert.NoError(t, err)

	for _, action := range fakeCS.Actions() {
		if p, ok := action.(clienttesting.PatchAction); ok {
			var patch statusPatch
			err = json.Unmarshal(p.GetPatch(), &patch)
			assert.NoError(t, err)
			assert.NotEmpty(t, patch.Status.Conditions, "Should not send a patch with zero conditions")
		}
	}
}

func TestProcessHealthEvents_MultipleGPU(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(
		&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-0"}},
		&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-1"}},
	)

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	healthEvents := &pb.HealthEvents{
		Events: []*pb.HealthEvent{
			{
				CheckName: "Fabric",
				IsHealthy: false,
				EntitiesImpacted: []*pb.Entity{
					{EntityType: "GPU", EntityValue: "GPU-0"},
					{EntityType: "GPU", EntityValue: "GPU-1"},
				},
			},
		},
	}

	err := connector.processHealthEvents(context.Background(), healthEvents)
	assert.NoError(t, err)

	actions := fakeCS.Actions()
	assert.Equal(t, 2, len(actions), "One event impacting two GPUs should result in two patches")
}

func TestProcessHealthEvents_PartialFailure(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-1"}})

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	healthEvents := &pb.HealthEvents{
		Events: []*pb.HealthEvent{
			{
				CheckName: "Health",
				EntitiesImpacted: []*pb.Entity{
					{EntityType: "GPU", EntityValue: "gpu-0"}, // Will fail (Not Found)
					{EntityType: "GPU", EntityValue: "gpu-1"},
				},
			},
		},
	}

	err := connector.processHealthEvents(context.Background(), healthEvents)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gpu-0")

	actions := fakeCS.Actions()
	foundGpu1 := false
	for _, action := range actions {
		if p, ok := action.(clienttesting.PatchAction); ok && p.GetName() == "gpu-1" {
			foundGpu1 = true
		}
	}
	assert.True(t, foundGpu1, "GPU-1 should have been patched even if GPU-0 failed")
}

func TestProcessGPUEvents_LatestEventWins(t *testing.T) {
	gpuName := "gpu-0"
	fakeGPU := &v1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{
			Name: gpuName,
		},
	}

	fakeCS := fake.NewSimpleClientset(fakeGPU)

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	events := []*pb.HealthEvent{
		{
			Id:                 "old-event",
			CheckName:          "Memory",
			IsHealthy:          true,
			Message:            "Previously healthy",
			GeneratedTimestamp: timestamppb.New(time.Now().Add(-10 * time.Minute)),
		},
		{
			Id:                 "new-event",
			CheckName:          "Memory",
			IsHealthy:          false,
			Message:            "Thermal throttling detected",
			GeneratedTimestamp: timestamppb.New(time.Now()),
		},
	}

	err := connector.processGPUEvents(context.Background(), gpuName, events)
	require.NoError(t, err)

	actions := fakeCS.Actions()
	require.Len(t, actions, 1, "Should have performed exactly one Patch action")

	patchAction := actions[0].(clienttesting.PatchAction)
	assert.Equal(t, gpuName, patchAction.GetName())
	assert.Equal(t, "status", patchAction.GetSubresource())

	var receivedPatch statusPatch
	err = json.Unmarshal(patchAction.GetPatch(), &receivedPatch)
	require.NoError(t, err)

	require.Len(t, receivedPatch.Status.Conditions, 1)
	cond := receivedPatch.Status.Conditions[0]

	assert.Equal(t, "Memory", cond.Type)
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, "MemoryIsNotHealthy", cond.Reason)
	assert.Equal(t, "Thermal throttling detected", cond.Message)
}

func TestProcessGPUEvents_MultipleChecks(t *testing.T) {
	gpuName := "gpu-0"
	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: gpuName}})

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	events := []*pb.HealthEvent{
		{
			CheckName: "Memory",
			IsHealthy: true,
			Message:   "Memory OK",
		},
		{
			CheckName: "Power",
			IsHealthy: false,
			Message:   "Power Cable Unplugged",
		},
	}

	err := connector.processGPUEvents(context.Background(), gpuName, events)
	require.NoError(t, err)

	patchAction := fakeCS.Actions()[0].(clienttesting.PatchAction)
	var receivedPatch statusPatch
	err = json.Unmarshal(patchAction.GetPatch(), &receivedPatch)
	assert.NoError(t, err)

	assert.Len(t, receivedPatch.Status.Conditions, 2, "Both unique checks should be present in the patch")

	conditionTypes := []string{
		receivedPatch.Status.Conditions[0].Type,
		receivedPatch.Status.Conditions[1].Type,
	}
	assert.Contains(t, conditionTypes, "Memory")
	assert.Contains(t, conditionTypes, "Power")
}

func TestProcessGPUEvents_MessageTruncation(t *testing.T) {
	gpuName := "gpu-0"
	fakeGPU := &v1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{
			Name: gpuName,
		},
	}

	fakeCS := fake.NewSimpleClientset(fakeGPU)

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	longMsg := strings.Repeat("A", 2000)
	events := []*pb.HealthEvent{
		{
			CheckName: "Storage",
			IsHealthy: false,
			Message:   longMsg,
		},
	}

	err := connector.processGPUEvents(context.Background(), gpuName, events)
	require.NoError(t, err)

	patchAction := fakeCS.Actions()[0].(clienttesting.PatchAction)
	var p statusPatch
	err = json.Unmarshal(patchAction.GetPatch(), &p)
	assert.NoError(t, err)

	msg := p.Status.Conditions[0].Message
	assert.True(t, len(msg) <= 1024, "Message should be truncated to 1KB")
	assert.Contains(t, msg, "[truncated]")
}

func TestProcessGPUEvents_TruncationBoundaries(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-0"}})

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	tests := []struct {
		nameLength int
		expected   string
	}{
		{1024, strings.Repeat("A", 1024)},
		{1025, strings.Repeat("A", 1024-len("... [truncated]")) + "... [truncated]"}, // Just over
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Length-%d", tt.nameLength), func(t *testing.T) {
			fakeCS.ClearActions()
			event := []*pb.HealthEvent{{
				CheckName: "LimitTest",
				Message:   strings.Repeat("A", tt.nameLength),
			}}

			err := connector.processGPUEvents(context.Background(), "gpu-0", event)
			require.NoError(t, err)

			var p statusPatch
			err = json.Unmarshal(fakeCS.Actions()[0].(clienttesting.PatchAction).GetPatch(), &p)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, p.Status.Conditions[0].Message)
			assert.LessOrEqual(t, len(p.Status.Conditions[0].Message), 1024)
		})
	}
}

func TestProcessGPUEvents_RespectsContextCancellation(t *testing.T) {
	gpuName := "gpu-0"
	fakeGPU := &v1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: gpuName},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fakeCS := fake.NewSimpleClientset(fakeGPU)
	fakeCS.PrependReactor("patch", "gpus", func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		select {
		case <-ctx.Done():
			return true, nil, ctx.Err()
		default:
			return false, nil, nil
		}
	})
	connector := &DeviceConnector{clientset: fakeCS}

	events := []*pb.HealthEvent{{CheckName: "Memory", IsHealthy: true}}

	err := connector.processGPUEvents(ctx, gpuName, events)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

func TestProcessGPUEvents_SortNilTimestamps(t *testing.T) {
	gpuName := "gpu-0"
	fakeGPU := &v1alpha1.GPU{
		ObjectMeta: metav1.ObjectMeta{Name: gpuName},
	}

	fakeCS := fake.NewSimpleClientset(fakeGPU)

	stopCh := make(chan struct{})
	defer close(stopCh)

	connector := &DeviceConnector{
		clientset: fakeCS,
		stopCh:    stopCh,
	}

	now := time.Now()
	events := []*pb.HealthEvent{
		{
			Id:                 "has-time",
			CheckName:          "Memory",
			Message:            "has-time",
			GeneratedTimestamp: timestamppb.New(now),
		},
		{
			Id:                 "no-time",
			CheckName:          "Memory",
			Message:            "no-time",
			GeneratedTimestamp: nil,
		},
	}

	err := connector.processGPUEvents(context.Background(), gpuName, events)
	require.NoError(t, err)

	actions := fakeCS.Actions()
	require.Len(t, actions, 1)

	patchAction := actions[0].(clienttesting.PatchAction)
	var p statusPatch
	err = json.Unmarshal(patchAction.GetPatch(), &p)
	assert.NoError(t, err)

	require.Len(t, p.Status.Conditions, 1)
	assert.Equal(t, "has-time", p.Status.Conditions[0].Message, "The latest (non-nil) event should have been used")
}

func TestProcessGPUEvents_HandlesAPIError(t *testing.T) {
	fakeCS := fake.NewSimpleClientset(&v1alpha1.GPU{ObjectMeta: metav1.ObjectMeta{Name: "gpu-0"}})

	fakeCS.PrependReactor("patch", "gpus", func(action clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("etcdserver: request timed out")
	})

	connector := &DeviceConnector{clientset: fakeCS}
	events := []*pb.HealthEvent{{CheckName: "Memory", IsHealthy: true}}

	err := connector.processGPUEvents(context.Background(), "gpu-0", events)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "etcdserver")
}
