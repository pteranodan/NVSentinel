//  Copyright (c) 2026, NVIDIA CORPORATION.  All rights reserved.
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

package filesystem

import (
	"os"
	"strings"
	"testing"
)

func TestCheckPermissions(t *testing.T) {
	tests := []struct {
		name    string
		perm    os.FileMode
		wantErr string
	}{
		{
			name:    "Directory is writable and readable",
			perm:    0750,
			wantErr: "",
		},
		{
			name:    "Directory is read-only",
			perm:    0550,
			wantErr: "write check failed",
		},
		{
			name:    "Directory is write-only",
			perm:    0330,
			wantErr: "read check failed",
		},
		{
			name:    "Directory is inaccessible",
			perm:    0000,
			wantErr: "read check failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if err := os.Chmod(dir, tt.perm); err != nil {
				t.Fatal(err)
			}

			err := CheckPermissions(dir)

			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatal("expected error, but got nil")
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}
