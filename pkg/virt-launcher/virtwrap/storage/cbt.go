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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/storage/volumepath"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

const (
	overlayClusterSize = 1 << 18 // 256KiB
	stdioFdCount       = 3
)

func DiskHasDataStore(disk *api.Disk) bool {
	return disk != nil && disk.Source.DataStore != nil
}

func IsChangedBlockTrackingEnabled(vmi *v1.VirtualMachineInstance) bool {
	return cbt.CompareCBTState(vmi.Status.ChangedBlockTracking, v1.ChangedBlockTrackingEnabled)
}

func shouldCreateQCOW2Overlay(vmi *v1.VirtualMachineInstance, isHotplug bool, hotplugPhase v1.VolumePhase) bool {
	if cbt.CBTStateInitializing(vmi.Status.ChangedBlockTracking) {
		return true
	}

	if !IsChangedBlockTrackingEnabled(vmi) {
		return false
	}

	// For hotplug volumes with CBT enabled, only create overlay when mounted but not yet in domain.
	// Once VolumeReady, the volume is in the domain and the overlay was already created.
	return isHotplug && hotplugPhase == v1.HotplugVolumeMounted
}

var CreateQCOW2Overlay = createQCOW2Overlay

// XXX: Use keep_data_file once QEMU 11 lands.
func createQCOW2Overlay(overlayPath, imagePath string) error {
	if fi, err := os.Lstat(overlayPath); err == nil {
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("QCOW2 overlay %s is not a regular file", overlayPath)
		}
		log.Log.V(3).Infof("overlay %s already exists", overlayPath)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to check QCOW2 overlay %s existence: %w", overlayPath, err)
	}

	info, err := osdisk.GetDiskInfo(imagePath)
	if err != nil {
		return fmt.Errorf("failed to get image info for image %q: %w", imagePath, err)
	}

	fd, err := unix.MemfdCreate("cbt-scratch", unix.MFD_CLOEXEC)
	if err != nil {
		return fmt.Errorf("failed to create scratch data file: %w", err)
	}
	scratch := os.NewFile(uintptr(fd), "cbt-scratch")
	defer scratch.Close()

	tmpOverlay := overlayPath + ".tmp"
	if err := os.Remove(tmpOverlay); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove stale QCOW2 overlay %s: %w", tmpOverlay, err)
	}
	// #nosec G302: the overlay is shared between virt-launcher and qemu.
	tmp, err := os.OpenFile(tmpOverlay, os.O_RDWR|os.O_CREATE|os.O_EXCL|unix.O_NOFOLLOW, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create QCOW2 overlay %s: %w", tmpOverlay, err)
	}
	defer os.Remove(tmpOverlay)
	defer tmp.Close()

	var extraFiles []*os.File
	extraFiles = append(extraFiles, scratch)
	scratchPath := fmt.Sprintf("/proc/self/fd/%d", stdioFdCount+len(extraFiles)-1)
	extraFiles = append(extraFiles, tmp)
	tmpPath := fmt.Sprintf("/proc/self/fd/%d", stdioFdCount+len(extraFiles)-1)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := osdisk.QEMUImg(ctx, extraFiles, "create", "-f", "qcow2",
		"-o", fmt.Sprintf("data_file=%s,data_file_raw=on,cluster_size=%d", scratchPath, overlayClusterSize),
		tmpPath, strconv.FormatInt(info.VirtualSize, 10)); err != nil {
		return err
	}
	if err := osdisk.QEMUImg(ctx, extraFiles, "amend", "-f", "qcow2", "-o", "data_file="+imagePath, tmpPath); err != nil {
		return err
	}
	if err := os.Rename(tmpOverlay, overlayPath); err != nil {
		return fmt.Errorf("failed to move QCOW2 overlay into place at %s: %w", overlayPath, err)
	}

	log.Log.Infof("QCOW2 overlay %s created successfully", overlayPath)
	return nil
}

func DeleteQCOW2Overlay(vmi *v1.VirtualMachineInstance, volumeName string) error {
	if !cbt.HasCBTStateEnabled(vmi.Status.ChangedBlockTracking) {
		return nil
	}

	overlayPath := cbt.GetQCOW2OverlayPath(vmi, volumeName)
	if err := os.Remove(overlayPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to delete QCOW2 overlay %s for volume %s: %w", overlayPath, volumeName, err)
	}
	log.Log.Infof("QCOW2 overlay %s deleted for unplugged volume %s", overlayPath, volumeName)
	return nil
}

// CleanupChangedBlockTrackingOverlays removes all QCOW2 overlays under the CBT
// directory when CBT is not active on the VMI. Used after CBT disable when the
// backend PVC is retained (e.g. TPM/EFI). Disabled is the common case and may
// run on every SyncVMI until the directory is empty; nil/Undefined are effectively
// no-ops when no overlays exist. Callers must ensure the domain is not using
// these overlays (CBT disable requires a restart).
func CleanupChangedBlockTrackingOverlays(vmi *v1.VirtualMachineInstance) error {
	// Mirror SyncVMI: only wipe when CBT is not Initializing/Enabled.
	if cbt.HasCBTStateEnabled(vmi.Status.ChangedBlockTracking) {
		return nil
	}
	return reconcileQCOW2Overlays(vmi, nil)
}

// ReconcileChangedBlockTrackingOverlays is a GC pass for overlays whose volumes
// are no longer on the VMI. Hot-unplug deletes the matching overlay via
// DeleteQCOW2Overlay during syncDisks; this reconciler catches orphans that miss
// that path (and removes leftover *.qcow2.tmp files). Must run after disk detach
// so QEMU is not still referencing a path that is about to be unlinked. An empty
// desired set (no eligible volumes) removes every overlay.
func ReconcileChangedBlockTrackingOverlays(vmi *v1.VirtualMachineInstance) error {
	desiredVolumes := make(map[string]struct{})
	for _, volume := range vmi.Spec.Volumes {
		if cbt.IsCBTEligibleVolume(&volume) {
			desiredVolumes[volume.Name] = struct{}{}
		}
	}
	return reconcileQCOW2Overlays(vmi, desiredVolumes)
}

func reconcileQCOW2Overlays(vmi *v1.VirtualMachineInstance, desiredVolumes map[string]struct{}) error {
	return reconcileQCOW2OverlaysInDir(cbt.PathForCBT(vmi), desiredVolumes)
}

func reconcileQCOW2OverlaysInDir(cbtDir string, desiredVolumes map[string]struct{}) error {
	entries, err := os.ReadDir(cbtDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to list CBT overlays in %s: %w", cbtDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Also reclaim leftover createQCOW2Overlay temp files (*.qcow2.tmp).
		if strings.HasSuffix(name, ".qcow2.tmp") {
			overlayPath := filepath.Join(cbtDir, name)
			if err := os.Remove(overlayPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to delete stale QCOW2 overlay temp file %s: %w", overlayPath, err)
			}
			log.Log.Infof("stale QCOW2 overlay temp file %s deleted", overlayPath)
			continue
		}
		if !strings.HasSuffix(name, ".qcow2") {
			continue
		}
		volumeName := strings.TrimSuffix(name, ".qcow2")
		if _, keep := desiredVolumes[volumeName]; keep {
			continue
		}
		overlayPath := filepath.Join(cbtDir, name)
		// Safe after domain detach: guest data lives in the raw data_file, not this metadata overlay.
		if err := os.Remove(overlayPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to delete orphaned QCOW2 overlay %s: %w", overlayPath, err)
		}
		log.Log.Infof("QCOW2 overlay %s deleted (volume no longer in CBT set)", overlayPath)
	}
	return nil
}

func ApplyChangedBlockTracking(vmi *v1.VirtualMachineInstance, c *convertertypes.ConverterContext) error {
	logger := log.Log.Object(vmi)
	applyCBTMap := make(map[string]string)

	// create overlay for every disk supporting changedBlockTracking
	for _, volume := range vmi.Spec.Volumes {
		volumeName := volume.Name
		logger.V(3).Infof("Creating QCOW2 overlay for %+v", volume)
		if !cbt.IsCBTEligibleVolume(&volume) {
			logger.V(3).Infof("SKIP Creating QCOW2 overlay for %s", volume.Name)
			continue
		}

		overlayPath := cbt.GetQCOW2OverlayPath(vmi, volumeName)
		logger.V(3).Infof("QCOW2 overlay path is %s", overlayPath)

		hotplugStatus, isHotplug := c.HotplugVolumes[volumeName]

		if !shouldCreateQCOW2Overlay(vmi, isHotplug, hotplugStatus.Phase) {
			applyCBTMap[volumeName] = overlayPath
			continue
		}

		isBlock := c.IsBlockPVC[volumeName] || c.IsBlockDV[volumeName]
		imagePath := volumepath.Image(volumeName, isBlock, isHotplug)

		err := CreateQCOW2Overlay(overlayPath, imagePath)
		if err != nil {
			return err
		}
		applyCBTMap[volumeName] = overlayPath
	}

	c.ApplyCBT = applyCBTMap
	return nil
}

func isMigrationNewBackendStorage(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Status.MigrationState == nil {
		return false
	}
	sourcePVC := vmi.Status.MigrationState.SourcePersistentStatePVCName
	targetPVC := vmi.Status.MigrationState.TargetPersistentStatePVCName
	return sourcePVC != "" && targetPVC != "" && sourcePVC != targetPVC
}

func ApplyChangedBlockTrackingForMigration(vmi *v1.VirtualMachineInstance, c *convertertypes.ConverterContext) error {
	logger := log.Log.Object(vmi)
	applyCBTMap := make(map[string]string)

	for _, volume := range vmi.Spec.Volumes {
		if !cbt.IsCBTEligibleVolume(&volume) {
			continue
		}

		volumeName := volume.Name
		overlayPath := cbt.GetQCOW2OverlayPath(vmi, volumeName)

		if isMigrationNewBackendStorage(vmi) {
			_, isHotplug := c.HotplugVolumes[volumeName]
			isBlock := c.IsBlockPVC[volumeName] || c.IsBlockDV[volumeName]
			imagePath := volumepath.Image(volumeName, isBlock, isHotplug)

			logger.V(3).Infof("Creating CBT overlay for migration: %s -> %s (block=%v, hotplug=%v)", overlayPath, imagePath, isBlock, isHotplug)
			if err := CreateQCOW2Overlay(overlayPath, imagePath); err != nil {
				return fmt.Errorf("failed to create CBT overlay for volume %s: %v", volumeName, err)
			}
		} else {
			logger.V(3).Infof("Using existing CBT overlay for migration (RWX backend): %s", overlayPath)
		}

		applyCBTMap[volumeName] = overlayPath
	}

	c.ApplyCBT = applyCBTMap
	return nil
}
