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

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

// CBTOverlayHook creates the CBT overlays the incoming domain references.
func CBTOverlayHook(_ *convertertypes.ConverterContext, _ *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if domain.Devices == nil {
		return nil
	}
	for i := range domain.Devices.Disks {
		source := domain.Devices.Disks[i].Source
		if source == nil || source.DataStore == nil || source.File == nil {
			continue
		}
		dataPath, err := dataFilePath(source.DataStore)
		if err != nil {
			return err
		}
		if err := storage.CreateQCOW2Overlay(source.File.File, dataPath); err != nil {
			return fmt.Errorf("failed to create CBT overlay %s: %w", source.File.File, err)
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
