// Copyright (c) 2025, NVIDIA CORPORATION.  All rights reserved.
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

package v1alpha1

import (
	"errors"
	"fmt"
)

// IsValid checks that all mandatory fields on the GPU object are present and correctly formatted.
func (g *GPU) IsValid() error {
	if g.Spec.NodeName == "" {
		return errors.New("spec.nodeName is required")
	}
	if !NameRegex.MatchString(g.Spec.NodeName) {
		return fmt.Errorf("spec.nodeName '%s' is invalid: must be a valid DNS-1123 subdomain", g.Spec.NodeName)
	}

	if g.Spec.ID == "" {
		return errors.New("spec.id is required")
	}
	if !GPUIDRegex.MatchString(g.Spec.ID) {
		return fmt.Errorf("spec.id '%s' is invalid: must be a valid NVIDIA GPU UUID (e.g. GPU-1234...)", g.Spec.ID)
	}

	if g.Name != "" && !NameRegex.MatchString(g.Name) {
		return fmt.Errorf("name '%s' is invalid: must be a valid DNS-1123 subdomain", g.Name)
	}

	expectedName, err := g.ComputeCanonicalName()
	if err != nil {
		return fmt.Errorf("failed to compute canonical name: %w", err)
	}

	if g.Name != "" && g.Name != expectedName {
		return fmt.Errorf("name '%s' is invalid; expected '%s'", g.Name, expectedName)
	}

	return nil
}
