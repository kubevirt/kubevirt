/*
Copyright 2024 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package checkpoint

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Warning: The interface doesn't require thread-safe behaviour
type CheckpointManager interface {
	// Retrieves checkpoint for given key. The value needs to be pointer to struct.
	// os.ErrNotExist is returned if there is no checkpoint for given key.
	Get(key string, value interface{}) error
	// Stores checkpoint for given key. The value needs to be pointer to struct.
	Store(key string, value interface{}) error
	// os.ErrNotExist is returned if there is no checkpoint for given key.
	Delete(key string) error
}

// Provides checkpoint manager that uses key to store checkpoints
// within provided directory. The manager uses std json package to
// encode/decode the stucts.
// This is thread-safe per key which is usual usage.
func NewSimpleCheckpointManager(path string) *simpleCheckpointManager {
	return &simpleCheckpointManager{path}
}

var _ CheckpointManager = &simpleCheckpointManager{}

type simpleCheckpointManager struct {
	basePath string
}

func (cp *simpleCheckpointManager) Get(key string, value interface{}) error {
	b, err := os.ReadFile(filepath.Join(cp.basePath, key))
	if err != nil {
		return err
	}
	return json.Unmarshal(b, value)
}

func (cp *simpleCheckpointManager) Store(key string, value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cp.basePath, key), b, 0o600)
}

func (cp *simpleCheckpointManager) Delete(key string) error {
	return os.Remove(filepath.Join(cp.basePath, key))
}
