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
	"errors"
	"fmt"
	"io"
	"os"
)

// IsWriteable checks if a directory or file path is writable.
func IsWriteable(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		return false, err
	}

	file, err := os.CreateTemp(path, ".write_test_")
	if err != nil {
		return false, err
	}

	file.Close()
	_ = os.Remove(file.Name())

	return true, nil
}

// IsReadable checks if a directory or file path is readable.
func IsReadable(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		return false, err
	}

	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	return true, nil
}

// CheckPermissions verifies that the provided path has both read and write permissions.
// It returns an error if either check fails
func CheckPermissions(path string) error {
	if ok, err := IsReadable(path); !ok {
		return fmt.Errorf("read check failed: %w", err)
	}
	if ok, err := IsWriteable(path); !ok {
		return fmt.Errorf("write check failed: %w", err)
	}
	return nil
}
