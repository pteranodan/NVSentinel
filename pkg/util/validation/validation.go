//  Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package validation

import (
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

func IsUnixSocketURI(uri string) []string {
	const prefix = "unix://"
	var errs []string

	if !strings.HasPrefix(uri, prefix) {
		errs = append(errs, fmt.Sprintf("must start with %q", prefix))
		return errs
	}

	path := strings.TrimPrefix(uri, prefix)
	if path == "" {
		errs = append(errs, "path is required")
		return errs
	}

	if !filepath.IsAbs(path) {
		errs = append(errs, fmt.Sprintf("path %q must be an absolute path", path))
	}

	if strings.HasSuffix(path, string(filepath.Separator)) {
		errs = append(errs, fmt.Sprintf("path %q must not end with a trailing slash", path))
	}

	return errs
}

func IsTCPAddress(addr string) []string {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return []string{err.Error()}
	}

	var errs []string
	if host != "" && host != "localhost" {
		if ip := net.ParseIP(host); ip == nil {
			errs = append(errs, fmt.Sprintf("invalid IP or hostname: %q", host))
		}
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		errs = append(errs, fmt.Sprintf("invalid port: %q", portStr))
	} else {
		errs = append(errs, validation.IsValidPortNum(port)...)
	}

	return errs
}
