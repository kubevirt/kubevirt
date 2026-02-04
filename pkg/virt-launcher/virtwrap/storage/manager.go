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
	"sync/atomic"

	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

const (
	FailedDomainMemoryDump   = "Domain memory dump failed"
	MaxConcurrentMemoryDumps = 1
)

type StorageManager struct {
	virConn                  cli.Connection
	metadataCache            *metadata.Cache
	memoryDumpInProgress     chan struct{}
	cancelSafetyUnfreezeChan chan struct{}
	freezeInProgress         atomic.Bool
}

func NewStorageManager(connection cli.Connection, metadataCache *metadata.Cache) *StorageManager {
	return &StorageManager{
		virConn:                  connection,
		metadataCache:            metadataCache,
		memoryDumpInProgress:     make(chan struct{}, MaxConcurrentMemoryDumps),
		cancelSafetyUnfreezeChan: make(chan struct{}),
	}
}

func (m *StorageManager) MigrationInProgress() bool {
	migrationMetadata, exists := m.metadataCache.Migration.Load()
	return exists && migrationMetadata.StartTimestamp != nil && migrationMetadata.EndTimestamp == nil
}
