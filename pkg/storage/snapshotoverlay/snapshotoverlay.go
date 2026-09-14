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

package snapshotoverlay

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

// ScratchDir returns where the scratch volume of the VMI's External snapshot is
// mounted. The volume has to be attached to the VMI: the launcher is handed a
// path, and an unknown name would let the caller pick an arbitrary directory.
func ScratchDir(vmi *v1.VirtualMachineInstance, volumeName string) (string, error) {
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return "", err
	}
	if reference == nil || volumeName != storagetypes.ScratchPVCName(reference.ContentUID) {
		return "", fmt.Errorf("snapshot overlay request has no matching transaction")
	}
	for _, utilityVolume := range vmi.Spec.UtilityVolumes {
		if utilityVolume.Name == volumeName {
			return hotplugdisk.GetVolumeMountDir(volumeName), nil
		}
	}

	return "", fmt.Errorf("%q is not a utility volume of this VirtualMachineInstance", volumeName)
}
