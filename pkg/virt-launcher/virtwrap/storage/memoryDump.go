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

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	api "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func (m *StorageManager) MemoryDump(vmi *v1.VirtualMachineInstance, dumpPath string) error {
	select {
	case m.memoryDumpInProgress <- struct{}{}:
	default:
		log.Log.Object(vmi).Infof("memory-dump is in progress")
		return nil
	}

	go func() {
		defer func() { <-m.memoryDumpInProgress }()
		if err := m.memoryDump(vmi, dumpPath); err != nil {
			log.Log.Object(vmi).Reason(err).Error(FailedDomainMemoryDump)
		}
	}()
	return nil
}

func (m *StorageManager) memoryDump(vmi *v1.VirtualMachineInstance, dumpPath string) error {
	logger := log.Log.Object(vmi)

	if m.shouldSkipMemoryDump(dumpPath) {
		return nil
	}
	m.initializeMemoryDumpMetadata(dumpPath)

	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.virConn.LookupDomainByName(domName)
	if dom == nil || err != nil {
		return err
	}
	defer dom.Free()
	// keep trying to do memory dump even if remove previous one failed
	removePreviousMemoryDump(filepath.Dir(dumpPath))

	logger.Infof("Starting memory dump")
	failed := false
	reason := ""
	err = dom.CoreDumpWithFormat(dumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY)
	if err != nil {
		failed = true
		reason = fmt.Sprintf("%s: %s", FailedDomainMemoryDump, err)
	} else {
		logger.Infof("Completed memory dump successfully")
	}

	m.setMemoryDumpResult(failed, reason)
	return err
}

func (m *StorageManager) shouldSkipMemoryDump(dumpPath string) bool {
	memoryDumpMetadata, _ := m.metadataCache.MemoryDump.Load()
	if memoryDumpMetadata.FileName == filepath.Base(dumpPath) {
		// memory dump still in progress or have just completed
		// no need to trigger another one
		return true
	}
	return false
}

func (m *StorageManager) initializeMemoryDumpMetadata(dumpPath string) {
	m.metadataCache.MemoryDump.WithSafeBlock(func(memoryDumpMetadata *api.MemoryDumpMetadata, initialized bool) {
		now := metav1.Now()
		*memoryDumpMetadata = api.MemoryDumpMetadata{
			FileName:       filepath.Base(dumpPath),
			StartTimestamp: &now,
		}
	})
	log.Log.V(4).Infof("initialize memory dump metadata: %s", m.metadataCache.MemoryDump.String())
}

func (m *StorageManager) setMemoryDumpResult(failed bool, reason string) {
	m.metadataCache.MemoryDump.WithSafeBlock(func(memoryDumpMetadata *api.MemoryDumpMetadata, initialized bool) {
		if !initialized {
			// nothing to report if memory dump metadata is empty
			return
		}

		now := metav1.Now()
		memoryDumpMetadata.Completed = true
		memoryDumpMetadata.EndTimestamp = &now
		memoryDumpMetadata.Failed = failed
		memoryDumpMetadata.FailureReason = reason
	})
	log.Log.V(4).Infof("set memory dump results in metadata: %s", m.metadataCache.MemoryDump.String())
}

func removePreviousMemoryDump(dir string) {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to remove older memory dumps")
		return
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "memory.dump") {
			err = os.Remove(filepath.Join(dir, file.Name()))
			if err != nil {
				log.Log.Reason(err).Errorf("failed to remove older memory dumps")
			}
		}
	}
}
