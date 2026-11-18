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

// CBTOverlayHook creates the CBT overlays the incoming domain references,
// each at the path of the claim the target VMI names for its volume.
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
		volume := volumeForAlias(vmi, d.Alias)
		if volume == nil {
			return fmt.Errorf("no target volume for CBT overlay %s", d.Source.File.File)
		}
		d.Source.File.File = cbt.GetQCOW2OverlayPath(vmi, volume)
		if err := storage.CreateQCOW2Overlay(d.Source.File.File, dataPath); err != nil {
			return fmt.Errorf("failed to create CBT overlay %s: %w", d.Source.File.File, err)
		}
	}
	return nil
}

func volumeForAlias(vmi *v1.VirtualMachineInstance, alias *libvirtxml.DomainAlias) *v1.Volume {
	if alias == nil {
		return nil
	}
	name := strings.TrimPrefix(alias.Name, api.UserAliasPrefix)
	for i := range vmi.Spec.Volumes {
		if vmi.Spec.Volumes[i].Name == name {
			return &vmi.Spec.Volumes[i]
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
