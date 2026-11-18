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
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var overlayDiskInfo = osdisk.GetDiskInfo

// RecoverOverlays starts the domain of a VMI replacing a crashed one on the
// overlays it left behind, so they can be committed once the domain runs.
func (m *StorageManager) RecoverOverlays(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	if vmi.Annotations[storagetypes.PendingOverlayRecoveryAnnotation] == "" {
		return nil
	}
	overlayDir := overlayScratchDir(vmi)
	reference, err := expectedOverlayReference(vmi, overlayDir)
	if err != nil {
		return err
	}
	if err := verifyOverlayDirMounted(overlayDir); err != nil {
		return err
	}
	if err := recoverOverlayDisks(domain, overlayDir); err != nil {
		return err
	}
	// Committing before the domain starts, so nothing captures the base images
	// the recovery is about to write
	m.metadataCache.SnapshotOverlay.WithSafeBlock(func(overlay *api.SnapshotOverlayMetadata, _ bool) {
		overlay.ContentUID, overlay.ScratchPVCUID = reference.ContentUID, reference.ScratchPVCUID
		overlay.Phase = api.SnapshotOverlayCommitting
		overlay.Message = "recovering the overlays of an interrupted external snapshot"
	})
	log.Log.Object(vmi).Infof("Starting the domain on the overlays left on scratch volume %s", overlayDir)
	return nil
}

// recoverOverlayDisks validates every remaining overlay before changing any
// disk. A missing overlay was committed: removal is synced at the pivot.
func recoverOverlayDisks(domain *api.Domain, overlayDir string) error {
	var disks []api.Disk
	for _, disk := range domain.Spec.Devices.Disks {
		if disk.Alias != nil {
			disks = append(disks, disk)
		}
	}
	overlays := make(map[string]string, len(disks))
	for _, disk := range disks {
		name := disk.Alias.GetName()
		path := overlayPath(overlayDir, name)
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("cannot stat overlay %s: %w", path, err)
		}
		base := disk.Source.File
		if disk.Type != "file" {
			base = disk.Source.Dev
		}
		info, err := overlayDiskInfo(path)
		if err != nil {
			return fmt.Errorf("cannot read overlay %s: %w", path, err)
		}
		if info.Format != "qcow2" || info.BackingFile != base || base == "" {
			return fmt.Errorf("overlay %s does not belong to volume %s: format %q, backing %q, expected %q", path, name, info.Format, info.BackingFile, base)
		}
		if disk.Driver == nil || disk.Driver.Type == "" {
			return fmt.Errorf("volume %s has no disk driver", name)
		}
		overlays[name] = path
	}
	if err := rejectStrayOverlays(overlayDir, disks); err != nil {
		return err
	}
	for i := range domain.Spec.Devices.Disks {
		disk := &domain.Spec.Devices.Disks[i]
		if disk.Alias == nil {
			continue
		}
		overlay, found := overlays[disk.Alias.GetName()]
		if !found {
			continue
		}
		base := disk.Source
		disk.BackingStore = &api.BackingStore{
			Type:   disk.Type,
			Format: &api.BackingStoreFormat{Type: disk.Driver.Type},
			Source: &base,
		}
		disk.Type = "file"
		disk.Source = api.DiskSource{File: overlay}
		disk.Driver.Type = "qcow2"
	}
	return nil
}

// rejectStrayOverlays refuses an overlay no disk would be started on, since its
// writes would be lost when the scratch volume is released
func rejectStrayOverlays(overlayDir string, disks []api.Disk) error {
	volumes := make(map[string]bool, len(disks))
	for _, disk := range disks {
		volumes[overlayPath(overlayDir, disk.Alias.GetName())] = true
	}
	overlays, err := filepath.Glob(filepath.Join(overlayDir, overlayFilePrefix+"*"+overlayFileSuffix))
	if err != nil {
		return err
	}
	for _, overlay := range overlays {
		if !volumes[overlay] {
			return fmt.Errorf("overlay %s belongs to no disk of the VMI", overlay)
		}
	}
	return nil
}

// CommitRecoveredOverlays starts the commit of overlays the domain was started on
func (m *StorageManager) CommitRecoveredOverlays(vmi *v1.VirtualMachineInstance) {
	if vmi.Annotations[storagetypes.PendingOverlayRecoveryAnnotation] == "" {
		return
	}
	overlay, _ := m.metadataCache.SnapshotOverlay.Load()
	if overlay.Phase != api.SnapshotOverlayCommitting {
		return
	}
	if overlayDir := overlayScratchDir(vmi); overlayDir != "" {
		go m.runCommitSnapshot(vmi, overlayDir)
	}
}
