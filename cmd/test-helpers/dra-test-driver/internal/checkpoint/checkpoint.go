/*
 * Copyright The Kubernetes Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package checkpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/types"
)

// Checkpoint contains data about devices prepared for each ResourceClaim the
// driver is responsible for. It is serialized to a JSON file that can be read
// by the driver to recover intermediate state.
type Checkpoint struct {
	PreparedClaims []PreparedClaim `json:"preparedClaims,omitempty"`
}

type PreparedClaim struct {
	UID            types.UID         `json:"uid,omitempty"`
	Name           string            `json:"name,omitempty"`
	Devices        []string          `json:"devices,omitempty"`
	DeviceRequests map[string]string `json:"deviceRequests,omitempty"`
}

// Read returns the Checkpoint at the given path. If the path doesn't exist,
// returns an empty Checkpoint and no error.
func Read(path string) (*Checkpoint, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &Checkpoint{}, nil
		}
		return nil, err
	}

	checkpoint := &Checkpoint{}
	if err := json.Unmarshal(data, checkpoint); err != nil {
		return nil, fmt.Errorf("unmarshal JSON from %s: %w", path, err)
	}

	return checkpoint, nil
}

// Write writes checkpoint to the file at path. The file is overwritten if it
// already exists and is created if it does not already exist.
func Write(path string, checkpoint *Checkpoint) (err error) {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "tmp-checkpoint-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer func() {
		if err1 := tmp.Close(); err1 != nil && err == nil {
			err = fmt.Errorf("close temp file: %w", err1)
		}
		if err != nil {
			os.Remove(tmpName)
		}
	}()

	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(checkpoint); err != nil {
		return fmt.Errorf("encode to temp file %s: %w", tmp.Name(), err)
	}

	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmp.Name(), path, err)
	}

	return nil
}
