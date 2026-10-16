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
	"fmt"
	"strings"

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

// CBTOverlayHook creates the CBT overlays the incoming domain references.
// A storage-migrated volume gets a fresh overlay keyed by its destination claim.
func CBTOverlayHook(_ *convertertypes.ConverterContext, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if domain.Devices == nil {
		return nil
	}
	for i := range domain.Devices.Disks {
		d := &domain.Devices.Disks[i]
		if d.Source == nil || d.Source.DataStore == nil || d.Source.File == nil {
			continue
		}
		dataPath, err := dataFilePath(d.Source.DataStore)
		if err != nil {
			return err
		}
		if d.Alias != nil {
			if migrated := migratedVolume(vmi, strings.TrimPrefix(d.Alias.Name, api.UserAliasPrefix)); migrated != nil {
				if migrated.DestinationPVCInfo == nil || migrated.DestinationPVCInfo.ClaimUID == "" {
					return fmt.Errorf("claim UID of the migration destination of volume %s is not known", migrated.VolumeName)
				}
				d.Source.File.File = cbt.QCOW2OverlayPathForClaim(vmi, migrated.DestinationPVCInfo.ClaimUID)
			}
		}
		if err := storage.CreateQCOW2Overlay(d.Source.File.File, dataPath); err != nil {
			return fmt.Errorf("failed to create CBT overlay %s: %w", d.Source.File.File, err)
		}
	}
	return nil
}

func migratedVolume(vmi *v1.VirtualMachineInstance, volumeName string) *v1.StorageMigratedVolumeInfo {
	for i := range vmi.Status.MigratedVolumes {
		if vmi.Status.MigratedVolumes[i].VolumeName == volumeName {
			return &vmi.Status.MigratedVolumes[i]
		}
	}
	return nil
}

func dataFilePath(dataStore *libvirtxml.DomainDiskDataStore) (string, error) {
	if dataStore.Source != nil {
		switch {
		case dataStore.Source.File != nil:
			return dataStore.Source.File.File, nil
		case dataStore.Source.Block != nil:
			return dataStore.Source.Block.Dev, nil
		}
	}
	return "", fmt.Errorf("CBT data store has no file or block source")
}
