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

package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/checkpoint"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type IterableCheckpointManager interface {
	ListKeys() []string
	checkpoint.CheckpointManager
}

type iterableCheckpointManager struct {
	base string
	checkpoint.CheckpointManager
}

func (icp *iterableCheckpointManager) ListKeys() []string {
	entries, err := os.ReadDir(icp.base)
	if err != nil {
		return []string{}
	}

	var keys []string
	for _, entry := range entries {
		keys = append(keys, entry.Name())
	}
	return keys

}

func NewIterableCheckpointManager(base string) IterableCheckpointManager {
	return &iterableCheckpointManager{
		base,
		checkpoint.NewSimpleCheckpointManager(base),
	}
}

type ghostRecord struct {
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	SocketFile string    `json:"socketFile"`
	UID        types.UID `json:"uid"`
}

var GhostRecordGlobalStore GhostRecordStore

type GhostRecordStore struct {
	cache             map[string]ghostRecord
	checkpointManager checkpoint.CheckpointManager
	sync.Mutex
}

func InitializeGhostRecordCache(iterableCPManager IterableCheckpointManager) *GhostRecordStore {

	GhostRecordGlobalStore = GhostRecordStore{
		cache:             make(map[string]ghostRecord),
		checkpointManager: iterableCPManager,
	}

	keys := iterableCPManager.ListKeys()
	for _, key := range keys {
		ghostRecord := ghostRecord{}
		if err := GhostRecordGlobalStore.checkpointManager.Get(key, &ghostRecord); err != nil {
			log.Log.Reason(err).Errorf("Unable to read ghost record checkpoint, %s", key)
			continue
		}
		key := ghostRecord.Namespace + "/" + ghostRecord.Name
		GhostRecordGlobalStore.cache[key] = ghostRecord
		log.Log.Infof("Added ghost record for key %s", key)
	}
	return &GhostRecordGlobalStore
}

func (store *GhostRecordStore) LastKnownUID(key string) types.UID {
	store.Lock()
	defer store.Unlock()

	record, ok := store.cache[key]
	if !ok {
		return ""
	}

	return record.UID
}

func (store *GhostRecordStore) list() []ghostRecord {
	store.Lock()
	defer store.Unlock()

	var records []ghostRecord

	for _, record := range store.cache {
		records = append(records, record)
	}

	return records
}

func (store *GhostRecordStore) findBySocket(socketFile string) (ghostRecord, bool) {
	store.Lock()
	defer store.Unlock()

	for _, record := range store.cache {
		if filepath.Clean(record.SocketFile) == socketFile {
			return record, true
		}
	}

	return ghostRecord{}, false
}

func (store *GhostRecordStore) Exists(namespace string, name string) bool {
	store.Lock()
	defer store.Unlock()

	key := namespace + "/" + name
	_, ok := store.cache[key]

	return ok
}

func (store *GhostRecordStore) Add(namespace string, name string, socketFile string, uid types.UID) (err error) {
	store.Lock()
	defer store.Unlock()
	if name == "" {
		return fmt.Errorf("can not add ghost record when 'name' is not provided")
	} else if namespace == "" {
		return fmt.Errorf("can not add ghost record when 'namespace' is not provided")
	} else if string(uid) == "" {
		return fmt.Errorf("unable to add ghost record with empty UID")
	} else if socketFile == "" {
		return fmt.Errorf("unable to add ghost record without a socketFile")
	}

	key := namespace + "/" + name
	record, ok := store.cache[key]
	if !ok {
		// record doesn't exist, so add new one.
		record := ghostRecord{
			Name:       name,
			Namespace:  namespace,
			SocketFile: socketFile,
			UID:        uid,
		}
		if err := store.checkpointManager.Store(string(uid), &record); err != nil {
			return fmt.Errorf("failed to checkpoint %s, %w", uid, err)
		}
		store.cache[key] = record
	}

	// This protects us from stomping on a previous ghost record
	// that was not cleaned up properly. A ghost record that was
	// not deleted indicates that the VMI shutdown process did not
	// properly handle cleanup of local data.
	if ok && record.UID != uid {
		return fmt.Errorf("can not add ghost record when entry already exists with differing UID")
	}

	if ok && filepath.Clean(record.SocketFile) != socketFile {
		return fmt.Errorf("can not add ghost record when entry already exists with differing socket file location")
	}

	return nil
}

func (store *GhostRecordStore) Delete(namespace string, name string) error {
	store.Lock()
	defer store.Unlock()
	key := namespace + "/" + name
	record, ok := store.cache[key]
	if !ok {
		// already deleted
		return nil
	}

	if string(record.UID) == "" {
		return fmt.Errorf("unable to remove ghost record with empty UID")
	}

	if err := store.checkpointManager.Delete(string(record.UID)); err != nil {
		return fmt.Errorf("failed to delete checkpoint %s, %w", record.UID, err)
	}

	delete(store.cache, key)

	return nil
}

func NewSharedInformer(virtShareDir string, watchdogTimeout int, recorder record.EventRecorder, vmiStore cache.Store, resyncPeriod time.Duration) cache.SharedInformer {
	lw := newListWatchFromNotify(virtShareDir, watchdogTimeout, recorder, vmiStore, resyncPeriod)
	return cache.NewSharedInformer(lw, &api.Domain{}, 0)
}
