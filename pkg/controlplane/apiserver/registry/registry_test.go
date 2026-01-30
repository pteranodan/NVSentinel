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

package registry

import (
	"sync"
	"testing"

	"github.com/nvidia/nvsentinel/pkg/controlplane/apiserver/api"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestRegistry(t *testing.T) {
	t.Run("RegisterAndAll", func(t *testing.T) {
		lock.Lock()
		providers = nil
		lock.Unlock()

		p := &fakeProvider{}
		Register(p)

		all := All()
		if len(all) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(all))
		}
		if all[0] != p {
			t.Error("Returned provider does not match registered provider")
		}
	})

	t.Run("DefensiveCopy", func(t *testing.T) {
		p := &fakeProvider{}
		Register(p)

		all := All()
		all[0] = nil

		if All()[0] == nil {
			t.Error("Internal registry was mutated; All() did not return a deep copy of the slice")
		}
	})

	t.Run("ConcurrencyRace", func(t *testing.T) {
		const count = 100
		var wg sync.WaitGroup
		wg.Add(count * 2)

		for i := 0; i < count; i++ {
			go func() {
				defer wg.Done()
				Register(&fakeProvider{})
			}()
			go func() {
				defer wg.Done()
				_ = All()
			}()
		}
		wg.Wait()
	})
}

type fakeProvider struct {
	name string
}

func (d *fakeProvider) GroupName() string { return d.name }
func (d *fakeProvider) GroupVersion() schema.GroupVersion {
	return schema.GroupVersion{Group: d.name, Version: "v1"}
}
func (d *fakeProvider) NewGroupInfo() *api.GroupInfo { return &api.GroupInfo{} }
