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

package metadata

import (
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type Cache struct {
	UID              SafeData[types.UID]
	Migration        SafeData[api.MigrationMetadata]
	GracePeriod      SafeData[api.GracePeriodMetadata]
	AccessCredential SafeData[api.AccessCredentialMetadata]
	MemoryDump       SafeData[api.MemoryDumpMetadata]

	notificationSignal chan struct{}
}

func NewCache() *Cache {
	cache := &Cache{
		notificationSignal: make(chan struct{}, 1),
	}
	cache.UID.dirtyChanel = cache.notificationSignal
	cache.Migration.dirtyChanel = cache.notificationSignal
	cache.GracePeriod.dirtyChanel = cache.notificationSignal
	cache.AccessCredential.dirtyChanel = cache.notificationSignal
	cache.MemoryDump.dirtyChanel = cache.notificationSignal
	return cache
}

// Listen to a notification signal about the cache content changes.
// Notifications are sent implicitly when the cache data is being changed.
func (c *Cache) Listen() <-chan struct{} {
	return c.notificationSignal
}

// ResetNotification clears the notification signal.
func (c *Cache) ResetNotification() {
	select {
	case <-c.notificationSignal:
	default:
	}
}
