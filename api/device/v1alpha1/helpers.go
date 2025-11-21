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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// HashLen defines the length of the truncated hex hash segment.
const HashLen = 12

// NameRegex defines the standard Kubernetes DNS-1123 Subdomain format.
var NameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

// GPUIDRegex defines the required NVIDIA GPU UUID format.
var GPUIDRegex = regexp.MustCompile(`^GPU-[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// GetName returns the currently assigned Name of the GPU.
//
// This is a non-mutating accessor.
func (g *GPU) GetName() string {
	return g.Name
}

// ComputeCanonicalName generates the fully expected name (GenerateName + Hash)
// based on the current Spec, without mutating the GPU object.
//
// Returns an error if the necessary Spec fields are missing.
func (g *GPU) ComputeCanonicalName() (string, error) {
	if g.Name != "" {
		return g.Name, nil
	}

	hashedID, err := computeDeterministicIDSegment(g.Spec.NodeName, g.Spec.ID)
	if err != nil {
		return "", err
	}

	return g.GenerateName + hashedID, nil
}

// EnsureName applies the naming policy:
//  1. If g.Name is set, returns the existing name.
//  2. If g.Name is empty, computes and sets Name = g.GenerateName + DeterministicHash.
//
// This function MUTATES the GPU object.
func (g *GPU) EnsureName() (string, error) {
	if g.Name != "" {
		return g.Name, nil
	}

	finalName, err := g.ComputeCanonicalName()
	if err != nil {
		return "", err
	}

	g.Name = finalName

	return g.Name, nil
}

// computeDeterministicIDSegment generates the stable, unique hash segment.
func computeDeterministicIDSegment(nodeName string, id string) (string, error) {
	if nodeName == "" || id == "" {
		return "", fmt.Errorf("spec.nodeName and spec.id are required to compute canonical path")
	}

	resourcePath := fmt.Sprintf("%s/%s", strings.ToLower(nodeName), strings.ToLower(id))

	hash := sha256.Sum256([]byte(resourcePath))
	hashStr := hex.EncodeToString(hash[:])[:HashLen]

	return hashStr, nil
}
