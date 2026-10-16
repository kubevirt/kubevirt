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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

var overlayDiskInfo = osdisk.GetDiskInfo
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

// RecoverOverlays validates the complete transaction before changing any disk.
// Remaining overlays are matched by volume name and validated before startup.
func (m *StorageManager) RecoverOverlays(vmi *v1.VirtualMachineInstance, domain *api.Domain) (bool, error) {
	if vmi.Annotations[v1.PendingOverlayRecoveryAnnotation] == "" {
		return false, nil
	}
	dir := overlayScratchDir(vmi)
	reference, err := expectedOverlayReference(vmi, dir)
	if err != nil {
		return false, err
	}
	if err := verifyOverlayDirMounted(dir); err != nil {
		return false, err
	}
	if err := recoverOverlayDisks(vmi, domain, dir); err != nil {
		return false, err
	}
	m.metadataCache.SnapshotOverlay.WithSafeBlock(func(overlay *api.SnapshotOverlayMetadata, _ bool) {
		overlay.ContentUID, overlay.ScratchPVCUID = reference.ContentUID, reference.ScratchPVCUID
		overlay.Phase = api.SnapshotOverlayCommitting
		overlay.Message = "recovering an unfinished external snapshot transaction"
	})
	log.Log.Object(vmi).Info("Starting the domain with its validated snapshot overlay recovery state")
	// Even an empty scratch needs to publish matching completion.
	return true, nil
}

// recoverOverlayDisks validates the remaining overlays before changing any disk.
// Without retirement records, an absent overlay is treated as already committed.
func recoverOverlayDisks(vmi *v1.VirtualMachineInstance, domain *api.Domain, dir string) error {
	disks := snapshottableDisks(vmi, domain.Spec.Devices.Disks)
	if len(disks) == 0 {
		return fmt.Errorf("no snapshottable disks in recovery")
	}
	sources := make(map[string]string, len(disks))
	for _, disk := range disks {
		name := converter.GetVolumeNameByDisk(disk)
		path := overlayPath(dir, name)
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("cannot stat overlay %s: %w", path, err)
		}
		diskInfo, err := overlayDiskInfo(path)
		if err != nil {
			return fmt.Errorf("cannot read overlay %s: %w", path, err)
		}
		if disk.Driver == nil || disk.Driver.Type == "" {
			return fmt.Errorf("volume %s has no disk driver", name)
		}
		if diskInfo == nil || diskInfo.Format != "qcow2" || diskInfo.BackingFile != disk.Source.File {
			return fmt.Errorf("overlay %s has unexpected format or backing file: %+v, expected %s", path, diskInfo, disk.Source.File)
		}
		sources[name] = path
	}
	for i := range domain.Spec.Devices.Disks {
		disk := &domain.Spec.Devices.Disks[i]
		if disk.Alias == nil {
			continue
		}
		source, ok := sources[converter.GetVolumeNameByDisk(*disk)]
		if !ok {
			continue
		}
		disk.BackingStore = &api.BackingStore{Type: "file", Format: &api.BackingStoreFormat{Type: disk.Driver.Type}, Source: &api.DiskSource{File: disk.Source.File}}
		disk.Driver.Type = "qcow2"
		disk.Source.File = source
	}
	return nil
}

func (m *StorageManager) CommitRecoveredOverlays(vmi *v1.VirtualMachineInstance) {
	overlay, _ := m.metadataCache.SnapshotOverlay.Load()
	if overlay.Phase != api.SnapshotOverlayCommitting {
		return
	}
	dir := overlayScratchDir(vmi)
	if dir == "" {
		return
	}
	go m.runCommitSnapshot(vmi, dir)
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
