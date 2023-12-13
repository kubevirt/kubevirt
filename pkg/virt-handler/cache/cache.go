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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

const socketDialTimeout = 5

type ghostRecord struct {
	Name       string    `json:"name"`
	Namespace  string    `json:"namespace"`
	SocketFile string    `json:"socketFile"`
	UID        types.UID `json:"uid"`
}

var ghostRecordGlobalCache map[string]ghostRecord
var ghostRecordGlobalMutex sync.Mutex
var ghostRecordDir string

func InitializeGhostRecordCache(directoryPath string) error {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	ghostRecordGlobalCache = make(map[string]ghostRecord)
	ghostRecordDir = directoryPath
	err := util.MkdirAllWithNosec(ghostRecordDir)
	if err != nil {
		return err
	}

	files, err := os.ReadDir(ghostRecordDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		recordPath := filepath.Join(ghostRecordDir, file.Name())
		// #nosec no risk for path injection. Used only for testing and using static location
		fileBytes, err := os.ReadFile(recordPath)
		if err != nil {
			log.Log.Reason(err).Errorf("Unable to read ghost record file at path %s", recordPath)
			continue
		}

		ghostRecord := ghostRecord{}
		err = json.Unmarshal(fileBytes, &ghostRecord)
		if err != nil {
			log.Log.Reason(err).Errorf("Unable to unmarshal json contents of ghost record file at path %s", recordPath)
			continue
		}

		key := ghostRecord.Namespace + "/" + ghostRecord.Name
		ghostRecordGlobalCache[key] = ghostRecord
		log.Log.Infof("Added ghost record for key %s", key)
	}

	return nil
}

func LastKnownUIDFromGhostRecordCache(key string) types.UID {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	record, ok := ghostRecordGlobalCache[key]
	if !ok {
		return ""
	}

	return record.UID
}

func getGhostRecords() ([]ghostRecord, error) {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	var records []ghostRecord

	for _, record := range ghostRecordGlobalCache {
		records = append(records, record)
	}

	return records, nil
}

func findGhostRecordBySocket(socketFile string) (ghostRecord, bool) {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	for _, record := range ghostRecordGlobalCache {
		if record.SocketFile == socketFile {
			return record, true
		}
	}

	return ghostRecord{}, false
}

func HasGhostRecord(namespace string, name string) bool {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()

	key := namespace + "/" + name
	_, ok := ghostRecordGlobalCache[key]

	return ok
}

func AddGhostRecord(namespace string, name string, socketFile string, uid types.UID) (err error) {
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()
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
	recordPath := filepath.Join(ghostRecordDir, string(uid))

	record, ok := ghostRecordGlobalCache[key]
	if !ok {
		// record doesn't exist, so add new one.
		record := ghostRecord{
			Name:       name,
			Namespace:  namespace,
			SocketFile: socketFile,
			UID:        uid,
		}

		fileBytes, err := json.Marshal(&record)
		if err != nil {
			return err
		}
		f, err := os.Create(recordPath)
		if err != nil {
			return err
		}
		defer util.CloseIOAndCheckErr(f, &err)

		_, err = f.Write(fileBytes)
		if err != nil {
			return err
		}
		ghostRecordGlobalCache[key] = record
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
	ghostRecordGlobalMutex.Lock()
	defer ghostRecordGlobalMutex.Unlock()
	key := namespace + "/" + name
	record, ok := ghostRecordGlobalCache[key]
	if !ok {
		// already deleted
		return nil
	}

	if string(record.UID) == "" {
		return fmt.Errorf("unable to remove ghost record with empty UID")
	}

	recordPath := filepath.Join(ghostRecordDir, string(record.UID))
	err := os.RemoveAll(recordPath)
	if err != nil {
		return nil
	}

	delete(ghostRecordGlobalCache, key)

	return nil
}
