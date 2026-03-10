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

package logrus

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/klog/v2"
)

// KlogFormatter implements logrus.Formatter by redirecting logrus entries
// to klog. It ensures that dependencies using logrus (e.g., Kine)
// conform to the global klog configuration and verbosity.
type KlogFormatter struct {
	Verbosity uint32
}

// Format redirects the logrus entry to the appropriate klog function.
func (k *KlogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	msg := entry.Message
	if len(entry.Data) > 0 {
		msg = fmt.Sprintf("%s %s", msg, formatFields(entry.Data))
	}

	switch entry.Level {
	case logrus.PanicLevel, logrus.FatalLevel:
		klog.Fatal(msg)
	case logrus.ErrorLevel:
		klog.Error(msg)
	case logrus.WarnLevel:
		if k.V(2) {
			klog.Warning(msg)
		}
	case logrus.InfoLevel:
		if k.V(4) {
			klog.Info(msg)
		}
	case logrus.DebugLevel:
		if k.V(6) {
			klog.V(6).Info(msg)
		}
	case logrus.TraceLevel:
		if k.V(8) {
			klog.V(8).Info(msg)
		}
	}
	return nil, nil
}

// formatFields serializes logrus fields into a sorted key=value string.
func formatFields(data logrus.Fields) string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(" ")
		}
		fmt.Fprintf(&sb, "%s=%v", k, data[k])
	}
	return sb.String()
}

// V reports whether verbosity level l is at least the requested level.
func (k *KlogFormatter) V(l int) bool {
	return uint32(l) <= k.Verbosity
}
