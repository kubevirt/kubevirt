/*
 * This file is part of the KubeVirt project
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
 *
 * Copyright The KubeVirt Authors.
 *
 */

package mountrecord

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/checkpoint"
)

var errNoVMIUID = errors.New("mount record requires a VMI UID")

type Entry struct {
	TargetFile string `json:"targetFile"`
	SocketFile string `json:"socketFile,omitempty"`
}

type record struct {
	MountTargetEntries []Entry `json:"mountTargetEntries"`
	// TODO: remove once v1.9 is no longer a possible rollback target.
	UsesSafePaths bool `json:"usesSafePaths"`
}

type Store struct {
	checkpointManager checkpoint.CheckpointManager
	lock              sync.Mutex
	cache             map[types.UID][]Entry
}

func NewStore(checkpointManager checkpoint.CheckpointManager) *Store {
	return &Store{
		checkpointManager: checkpointManager,
		cache:             map[types.UID][]Entry{},
	}
}

func (s *Store) Entries(vmiUID types.UID) ([]Entry, error) {
	if vmiUID == "" {
		return nil, errNoVMIUID
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	entries, err := s.load(vmiUID)
	return slices.Clone(entries), err
}

func (s *Store) Add(vmiUID types.UID, entries ...Entry) error {
	if vmiUID == "" {
		return errNoVMIUID
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	current, err := s.load(vmiUID)
	if err != nil {
		return err
	}
	updated := slices.Clone(current)
	for _, entry := range entries {
		if !slices.ContainsFunc(updated, func(e Entry) bool { return e.TargetFile == entry.TargetFile }) {
			updated = append(updated, entry)
		}
	}
	if len(updated) == len(current) {
		return nil
	}
	return s.store(vmiUID, updated)
}

func (s *Store) Replace(vmiUID types.UID, entries []Entry) error {
	if vmiUID == "" {
		return errNoVMIUID
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	if cached, ok := s.cache[vmiUID]; ok && slices.Equal(cached, entries) {
		return nil
	}
	return s.store(vmiUID, slices.Clone(entries))
}

func (s *Store) Delete(vmiUID types.UID) error {
	if vmiUID == "" {
		return errNoVMIUID
	}
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.checkpointManager.Delete(string(vmiUID)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to delete mount record of VMI %s: %w", vmiUID, err)
	}
	delete(s.cache, vmiUID)
	return nil
}

func (s *Store) load(vmiUID types.UID) ([]Entry, error) {
	if entries, ok := s.cache[vmiUID]; ok {
		return entries, nil
	}

	var persisted record
	err := s.checkpointManager.Get(string(vmiUID), &persisted)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get mount record of VMI %s: %w", vmiUID, err)
	}

	s.cache[vmiUID] = persisted.MountTargetEntries
	return persisted.MountTargetEntries, nil
}

func (s *Store) store(vmiUID types.UID, entries []Entry) error {
	if err := s.checkpointManager.Store(string(vmiUID), &record{MountTargetEntries: entries, UsesSafePaths: true}); err != nil {
		return fmt.Errorf("failed to store mount record of VMI %s: %w", vmiUID, err)
	}
	s.cache[vmiUID] = entries
	return nil
}
