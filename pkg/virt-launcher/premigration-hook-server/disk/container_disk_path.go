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
 */

package disk

import (
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	containerdisk "kubevirt.io/kubevirt/pkg/container-disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

// ContainerDiskPathHook repoints container disk backing stores at the volume-name-based
// path the target created. Domains started before that naming still reference
// disk_<index>.img, which does not exist on the target. Matching on the ua-<volumeName>
// alias keeps this a no-op for domains already using the new naming.
//
// TODO: remove once VMs started before volume-name-based container disk paths can no
// longer be migrated. See https://github.com/kubevirt/kubevirt/issues/17229
func ContainerDiskPathHook(_ *convertertypes.ConverterContext, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if domain.Devices == nil {
		return nil
	}

	containerDiskVolumes := make(map[string]string)
	for _, volume := range vmi.Spec.Volumes {
		if volume.ContainerDisk != nil {
			containerDiskVolumes[api.UserAliasPrefix+volume.Name] = volume.Name
		}
	}
	if len(containerDiskVolumes) == 0 {
		return nil
	}

	for i := range domain.Devices.Disks {
		disk := &domain.Devices.Disks[i]
		if disk.Alias == nil {
			continue
		}
		volumeName, isContainerDisk := containerDiskVolumes[disk.Alias.Name]
		if !isContainerDisk {
			continue
		}
		// The container disk image is the backing store of the ephemeral overlay,
		// not the disk source itself.
		if disk.BackingStore == nil || disk.BackingStore.Source == nil || disk.BackingStore.Source.File == nil {
			continue
		}

		expectedPath := containerdisk.GetDiskTargetPathFromLauncherView(volumeName)
		if disk.BackingStore.Source.File.File == expectedPath {
			continue
		}

		log.Log.Object(vmi).Infof("containerDiskPathHook: updating container disk path from %s to %s",
			disk.BackingStore.Source.File.File, expectedPath)
		disk.BackingStore.Source.File.File = expectedPath
	}

	return nil
}
