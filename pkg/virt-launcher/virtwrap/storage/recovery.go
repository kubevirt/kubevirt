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
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"

	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

var overlayVolumeMountDir = hotplugdisk.GetVolumeMountDir

func expectedOverlayReference(vmi *v1.VirtualMachineInstance, dir string) (*storagetypes.SnapshotOverlayReference, error) {
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return nil, err
	}
	if reference == nil {
		return nil, fmt.Errorf("external snapshot transaction reference is missing")
	}
	if overlayScratchDir(vmi) != dir || dir == "" {
		return nil, fmt.Errorf("expected scratch utility volume is not attached")
	}
	return reference, nil
}

func overlayScratchDir(vmi *v1.VirtualMachineInstance) string {
	pending := vmi.Annotations[v1.PendingOverlayRecoveryAnnotation]
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil || reference == nil {
		return ""
	}
	name := storagetypes.ScratchPVCName(reference.ContentUID)
	if pending != "" && pending != name {
		return ""
	}
	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Name == name && volume.ClaimName == name && volume.Type != nil && *volume.Type == v1.SnapshotOverlay {
			return filepath.Clean(overlayVolumeMountDir(volume.Name))
		}
	}
	return ""
}
