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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package cache

import (
	"fmt"
	"os"
	"sync"
	"time"

	"k8s.io/client-go/tools/record"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/checkpoint"
	"kubevirt.io/kubevirt/pkg/util"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
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

	keys := []string{}
	for _, entry := range entries {
		keys = append(keys, entry.Name())
	}
	return keys

}

func newIterableCheckpointManager(base string) IterableCheckpointManager {
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

var ghostRecordGlobalStore GhostRecordStore

type GhostRecordStore struct {
	cache             map[string]ghostRecord
	checkpointManager checkpoint.CheckpointManager
	sync.Mutex
}

func InitializeGhostRecordCache(directoryPath string) error {

	iterablecpm := newIterableCheckpointManager(directoryPath)
	ghostRecordGlobalStore = GhostRecordStore{
		cache:             make(map[string]ghostRecord),
		checkpointManager: iterablecpm,
	}

	err := util.MkdirAllWithNosec(directoryPath)
	if err != nil {
		return err
	}

	keys := iterablecpm.ListKeys()
	for _, key := range keys {
		ghostRecord := ghostRecord{}
		if err := ghostRecordGlobalStore.checkpointManager.Get(key, &ghostRecord); err != nil {
			log.Log.Reason(err).Errorf("Unable to read ghost record checkpoint, %s", key)
			continue
		}
		key := ghostRecord.Namespace + "/" + ghostRecord.Name
		ghostRecordGlobalStore.cache[key] = ghostRecord
		log.Log.Infof("Added ghost record for key %s", key)
	}
	return nil
}

func LastKnownUIDFromGhostRecordCache(key string) types.UID {
	ghostRecordGlobalStore.Lock()
	defer ghostRecordGlobalStore.Unlock()

	record, ok := ghostRecordGlobalStore.cache[key]
	if !ok {
		return ""
	}

	return record.UID
}

func getGhostRecords() ([]ghostRecord, error) {
	ghostRecordGlobalStore.Lock()
	defer ghostRecordGlobalStore.Unlock()

	var records []ghostRecord

	for _, record := range ghostRecordGlobalStore.cache {
		records = append(records, record)
	}

	return records, nil
}

func findGhostRecordBySocket(socketFile string) (ghostRecord, bool) {
	ghostRecordGlobalStore.Lock()
	defer ghostRecordGlobalStore.Unlock()

	for _, record := range ghostRecordGlobalStore.cache {
		if record.SocketFile == socketFile {
			return record, true
		}
	}

	return ghostRecord{}, false
}

func HasGhostRecord(namespace string, name string) bool {
	ghostRecordGlobalStore.Lock()
	defer ghostRecordGlobalStore.Unlock()

	key := namespace + "/" + name
	_, ok := ghostRecordGlobalStore.cache[key]

	return ok
}

func AddGhostRecord(namespace string, name string, socketFile string, uid types.UID) (err error) {
	ghostRecordGlobalStore.Lock()
	defer ghostRecordGlobalStore.Unlock()
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
	record, ok := ghostRecordGlobalStore.cache[key]
	if !ok {
		// record doesn't exist, so add new one.
		record := ghostRecord{
			Name:       name,
			Namespace:  namespace,
			SocketFile: socketFile,
			UID:        uid,
		}
		if err := ghostRecordGlobalStore.checkpointManager.Store(string(uid), &record); err != nil {
			return fmt.Errorf("failed to checkpoint %s, %w", uid, err)
		}
		ghostRecordGlobalStore.cache[key] = record
	}

	// This protects us from stomping on a previous ghost record
	// that was not cleaned up properly. A ghost record that was
	// not deleted indicates that the VMI shutdown process did not
	// properly handle cleanup of local data.
	if ok && record.UID != uid {
		return fmt.Errorf("can not add ghost record when entry already exists with differing UID")
	}

	if ok && record.SocketFile != socketFile {
		return fmt.Errorf("can not add ghost record when entry already exists with differing socket file location")
	}

	return nil
}

func DeleteGhostRecord(namespace string, name string) error {
	ghostRecordGlobalStore.Lock()
	defer ghostRecordGlobalStore.Unlock()
	key := namespace + "/" + name
	record, ok := ghostRecordGlobalStore.cache[key]
	if !ok {
		// already deleted
		return nil
	}

	if string(record.UID) == "" {
		return fmt.Errorf("unable to remove ghost record with empty UID")
	}

	if err := ghostRecordGlobalStore.checkpointManager.Delete(string(record.UID)); err != nil {
		return fmt.Errorf("failed to delete checkpoint %s, %w", record.UID, err)
	}

	delete(ghostRecordGlobalStore.cache, key)

	return nil
}

func listSockets() ([]string, error) {
	var sockets []string

	knownSocketFiles, err := cmdclient.ListAllSockets()
	if err != nil {
		return sockets, err
	}
	ghostRecords, err := getGhostRecords()
	if err != nil {
		return sockets, err
	}

	sockets = append(sockets, knownSocketFiles...)

	for _, record := range ghostRecords {
		exists := false
		for _, socket := range knownSocketFiles {
			if record.SocketFile == socket {
				exists = true
				break
			}
		}
		if !exists {
			sockets = append(sockets, record.SocketFile)
		}
	}

	return sockets, nil
}

func NewSharedInformer(virtShareDir string, watchdogTimeout int, recorder record.EventRecorder, vmiStore cache.Store, resyncPeriod time.Duration) cache.SharedInformer {
	lw := newListWatchFromNotify(virtShareDir, watchdogTimeout, recorder, vmiStore, resyncPeriod)
	return cache.NewSharedInformer(lw, &api.Domain{}, 0)
}
