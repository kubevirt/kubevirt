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

// ExternalSnapshot redirects the writes of every snapshottable disk of the
// running domain to a qcow2 overlay under overlayDir, in a single libvirt
// transaction, leaving the base images read-only. It returns as soon as the
// transaction is accepted and reports the outcome through the metadata cache.
// Called against a domain already on overlays it is a no-op; it does not check
// which snapshot those overlays belong to, serialising snapshots of the same VM
// is the caller's job.
func (m *StorageManager) ExternalSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) error {
	return fmt.Errorf("external snapshot is not implemented")
}

// CommitSnapshot starts a live block-commit of the overlays under overlayDir
// back into their base images, and returns as soon as it has been started.
// Completion is reported through the metadata cache. The controller re-issues
// the RPC every reconcile: a second call while a commit runs is a no-op, and a
// retry after a failed one resumes from whatever the domain is running on.
func (m *StorageManager) CommitSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) error {
	return fmt.Errorf("snapshot overlay commit is not implemented")
}
