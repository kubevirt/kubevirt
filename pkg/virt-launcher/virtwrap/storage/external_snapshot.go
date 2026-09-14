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

	v1 "kubevirt.io/api/core/v1"
)

// ExternalSnapshot redirects the disks of volumes to qcow2 overlays under
// overlayDir in a single libvirt transaction. It returns once libvirt accepts the
// transaction and reports the result through the domain metadata cache. Idempotent.
func (m *StorageManager) ExternalSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string, volumes []string) error {
	return fmt.Errorf("external snapshot is not implemented")
}

// CommitSnapshot starts a live block commit of the overlays under overlayDir
// back into their base images. It returns once the commit has started and
// reports completion through the domain metadata cache. Idempotent.
func (m *StorageManager) CommitSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) error {
	return fmt.Errorf("snapshot overlay commit is not implemented")
}
