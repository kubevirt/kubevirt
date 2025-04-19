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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// LoadKubevirtMetadata loads and returns all KubeVirt metadata.
// It serves as a convenient helper for processing the `KubeVirtMetadata`
// data structure as a whole.
// It is recommended to use individual loaders when possible.
func LoadKubevirtMetadata(metadataCache *Cache) api.KubeVirtMetadata {
	var kubevirtMetadata api.KubeVirtMetadata
	if value, exists := metadataCache.UID.Load(); exists {
		kubevirtMetadata.UID = value
	}
	if value, exists := metadataCache.GracePeriod.Load(); exists {
		kubevirtMetadata.GracePeriod = &value
	}
	if value, exists := metadataCache.Migration.Load(); exists {
		kubevirtMetadata.Migration = &value
	}
	if value, exists := metadataCache.AccessCredential.Load(); exists {
		kubevirtMetadata.AccessCredential = &value
	}
	if value, exists := metadataCache.MemoryDump.Load(); exists {
		kubevirtMetadata.MemoryDump = &value
	}
	return kubevirtMetadata
}
