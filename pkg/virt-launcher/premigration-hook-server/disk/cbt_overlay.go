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
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/disksource"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

// CBTOverlayHook creates CBT qcow2 overlays on the migration target when DestXML
// arrives. The overlay virtual size must match the source image size (qemu
// mirror requires equal sizes). The source size is required in kubevirt DestXML
// metadata (migrationDiskSizes); guessing from destination PVC/disk info is
// unreliable across filesystem and block volumes.
func CBTOverlayHook(_ *convertertypes.ConverterContext, vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if !cbt.HasCBTStateEnabled(vmi.Status.ChangedBlockTracking) {
		return nil
	}
	if domain.Devices == nil {
		return nil
	}

	logger := log.Log.Object(vmi)
	for i := range domain.Devices.Disks {
		disk := &domain.Devices.Disks[i]
		if disk.Source == nil || disk.Source.DataStore == nil || disk.Source.DataStore.Source == nil {
			continue
		}

		volumeName := volumeNameFromDiskAlias(disk)
		if volumeName == "" {
			return fmt.Errorf("CBT overlay hook: disk missing alias")
		}

		rds := disksource.Resolve(apiDiskFromLibvirtXML(disk))
		backendPath := rds.BackendPath()
		if backendPath == "" {
			return fmt.Errorf("CBT overlay hook: volume %s: dataStore source has no file or block path", volumeName)
		}

		overlaySize, ok := storage.MigrationDiskSize(domain, volumeName)
		if !ok {
			return fmt.Errorf("CBT overlay hook: volume %s: missing source virtual size in DestXML metadata", volumeName)
		}

		overlayPath := cbt.GetQCOW2OverlayPath(vmi, volumeName)
		logger.Infof("Creating CBT overlay for migration target: volume=%s overlay=%s dataStore=%s block=%v size=%d",
			volumeName, overlayPath, backendPath, rds.BackendIsBlock(), overlaySize)

		if err := storage.CreateQCOW2Overlay(overlayPath, backendPath, overlaySize); err != nil {
			return fmt.Errorf("CBT overlay hook: failed to create overlay for volume %s: %w", volumeName, err)
		}

		disk.Source.File = &libvirtxml.DomainDiskSourceFile{File: overlayPath}
		disk.Source.Block = nil
		// Defensive: CBT disks must not keep slices (rejected on dataStore; invalid on overlay).
		disk.Source.Slices = nil
		disk.Source.DataStore.Source.Slices = nil
	}

	storage.ClearMigrationDiskSizes(domain)
	return nil
}

func volumeNameFromDiskAlias(disk *libvirtxml.DomainDisk) string {
	if disk == nil || disk.Alias == nil {
		return ""
	}
	return api.UserAliasToName(disk.Alias.Name)
}

// apiDiskFromLibvirtXML maps the libvirtxml disk fields disksource.Resolve needs.
// The premigration hook works on DestXML (libvirtxml), while ResolvedDiskSource
// is built for the launcher api.Disk schema.
func apiDiskFromLibvirtXML(disk *libvirtxml.DomainDisk) api.Disk {
	out := api.Disk{}
	if disk == nil || disk.Source == nil {
		return out
	}
	if disk.Source.File != nil {
		out.Source.File = disk.Source.File.File
	}
	if disk.Source.Block != nil {
		out.Source.Dev = disk.Source.Block.Dev
	}
	if disk.Source.DataStore == nil || disk.Source.DataStore.Source == nil {
		return out
	}
	dsSrc := &api.DiskSource{}
	if disk.Source.DataStore.Source.File != nil {
		dsSrc.File = disk.Source.DataStore.Source.File.File
	}
	if disk.Source.DataStore.Source.Block != nil {
		dsSrc.Dev = disk.Source.DataStore.Source.Block.Dev
	}
	out.Source.DataStore = &api.DataStore{Source: dsSrc}
	return out
}
