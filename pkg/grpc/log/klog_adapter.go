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

package log

import (
	"google.golang.org/grpc/grpclog"
	"k8s.io/klog/v2"
)

var _ grpclog.LoggerV2 = (*KlogAdapter)(nil)

// KlogAdapter implements the grpclog.LoggerV2 interface using klog.
// It allows gRPC internal logs to be routed through the standard
// Kubernetes logging framework.
type KlogAdapter struct {
	Verbosity uint32
}

func (k *KlogAdapter) Info(args ...interface{}) {
	if k.V(4) {
		klog.Info(args...)
	}
}
func (k *KlogAdapter) Infoln(args ...interface{}) {
	if k.V(4) {
		klog.Infoln(args...)
	}
}
func (k *KlogAdapter) Infof(format string, args ...interface{}) {
	if k.V(4) {
		klog.Infof(format, args...)
	}
}

func (k *KlogAdapter) Warning(args ...interface{}) {
	if k.V(2) {
		klog.Warning(args...)
	}
}

func (k *KlogAdapter) Warningln(args ...interface{}) {
	if k.V(2) {
		klog.Warningln(args...)
	}
}

func (k *KlogAdapter) Warningf(format string, args ...interface{}) {
	if k.V(2) {
		klog.Warningf(format, args...)
	}
}
func (k *KlogAdapter) Error(args ...interface{})                 { klog.Error(args...) }
func (k *KlogAdapter) Errorln(args ...interface{})               { klog.Errorln(args...) }
func (k *KlogAdapter) Errorf(format string, args ...interface{}) { klog.Errorf(format, args...) }
func (k *KlogAdapter) Fatal(args ...interface{})                 { klog.Fatal(args...) }
func (k *KlogAdapter) Fatalln(args ...interface{})               { klog.Fatalln(args...) }
func (k *KlogAdapter) Fatalf(format string, args ...interface{}) { klog.Fatalf(format, args...) }

// V reports whether verbosity at the call site is at least the requested level.
func (k *KlogAdapter) V(l int) bool {
	return uint32(l) <= k.Verbosity
}
