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
	"math"

	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

// GetDiskInfo looks up disk image info; overridable in unit tests.
var GetDiskInfo = osdisk.GetDiskInfo

// CBTOverlayHook creates CBT qcow2 overlays on the migration target when DestXML
// arrives. The overlay virtual size must match the source image size (qemu
// mirror requires equal sizes). The source size is taken from the overlay
// slices placed in DestXML (not on dataStore), with MigratedVolumes /
// dataStore GetDiskInfo as fallbacks.
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

		backendPath, isBlock, err := dataStoreBackendPath(disk.Source.DataStore.Source)
		if err != nil {
			return fmt.Errorf("CBT overlay hook: volume %s: %w", volumeName, err)
		}

		overlaySize, err := sourceOverlaySize(vmi, volumeName, disk, backendPath)
		if err != nil {
			return fmt.Errorf("CBT overlay hook: volume %s: %w", volumeName, err)
		}

		overlayPath := cbt.GetQCOW2OverlayPath(vmi, volumeName)
		logger.Infof("Creating CBT overlay for migration target: volume=%s overlay=%s dataStore=%s block=%v size=%d",
			volumeName, overlayPath, backendPath, isBlock, overlaySize)

		if err := storage.CreateQCOW2Overlay(overlayPath, backendPath, isBlock, overlaySize); err != nil {
			return fmt.Errorf("CBT overlay hook: failed to create overlay for volume %s: %w", volumeName, err)
		}

		disk.Source.File = &libvirtxml.DomainDiskSourceFile{File: overlayPath}
		disk.Source.Block = nil
		// Slices were only a size carrier in DestXML; libvirt rejects them on dataStore
		// and they must not remain once the overlay has the correct virtual size.
		disk.Source.Slices = nil
		disk.Source.DataStore.Source.Slices = nil
	}

	return nil
}

// sourceOverlaySize returns the virtual size the target CBT overlay must have so
// it matches the source image for block mirror.
func sourceOverlaySize(vmi *v1.VirtualMachineInstance, volumeName string, disk *libvirtxml.DomainDisk, backendPath string) (int64, error) {
	if size, ok := sizeFromSlices(disk.Source); ok {
		return size, nil
	}
	if disk.Source.DataStore != nil {
		if size, ok := sizeFromSlices(disk.Source.DataStore.Source); ok {
			return size, nil
		}
	}
	if size := sourceCapacityFromMigratedVolumes(vmi, volumeName); size > 0 {
		return size, nil
	}

	info, err := GetDiskInfo(backendPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get disk info for path %s: %w", backendPath, err)
	}
	if info == nil || info.VirtualSize <= 0 {
		return 0, fmt.Errorf("invalid virtual size %v for path %s", info, backendPath)
	}
	return info.VirtualSize, nil
}

func sizeFromSlices(source *libvirtxml.DomainDiskSource) (int64, bool) {
	if source == nil || source.Slices == nil || len(source.Slices.Slices) == 0 {
		return 0, false
	}
	sliceSize := uint64(source.Slices.Slices[0].Size)
	if sliceSize == 0 || sliceSize > math.MaxInt64 {
		return 0, false
	}
	return int64(sliceSize), true
}

func sourceCapacityFromMigratedVolumes(vmi *v1.VirtualMachineInstance, volumeName string) int64 {
	for i := range vmi.Status.MigratedVolumes {
		mv := &vmi.Status.MigratedVolumes[i]
		if mv.VolumeName != volumeName || mv.SourcePVCInfo == nil {
			continue
		}
		if capacity := storagetypes.GetDiskCapacity(mv.SourcePVCInfo); capacity != nil && *capacity > 0 {
			return *capacity
		}
	}
	return 0
}

func volumeNameFromDiskAlias(disk *libvirtxml.DomainDisk) string {
	if disk == nil || disk.Alias == nil {
		return ""
	}
	return api.UserAliasToName(disk.Alias.Name)
}

func dataStoreBackendPath(source *libvirtxml.DomainDiskSource) (path string, isBlock bool, err error) {
	if source == nil {
		return "", false, fmt.Errorf("empty dataStore source")
	}
	switch {
	case source.Block != nil && source.Block.Dev != "":
		return source.Block.Dev, true, nil
	case source.File != nil && source.File.File != "":
		return source.File.File, false, nil
	default:
		return "", false, fmt.Errorf("dataStore source has no file or block path")
	}
}
