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
	"time"

	"golang.org/x/sys/unix"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	osdisk "kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/storage/cbt"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
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
	if cbt.IsClaimOverlayPath(overlayPath) {
		if err := ensureClaimDir(filepath.Dir(overlayPath)); err != nil {
			return fmt.Errorf("failed to prepare QCOW2 overlay directory for %s: %w", overlayPath, err)
		}
	}
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

func DeleteQCOW2Overlay(disk api.Disk) error {
	if !DiskHasDataStore(&disk) || disk.Source.File == "" {
		return nil
	}
	target := disk.Source.File
	if cbt.IsClaimOverlayPath(target) {
		target = filepath.Dir(target)
		if err := requireDir(filepath.Dir(target)); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to delete QCOW2 overlay %s: %w", target, err)
		}
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("failed to delete QCOW2 overlay %s: %w", target, err)
	}
	log.Log.Infof("QCOW2 overlay %s deleted", target)
	return nil
}

func ensureClaimDir(dir string) error {
	for _, d := range []string{filepath.Dir(dir), dir} {
		// #nosec G301: qemu has to traverse the directory.
		if err := os.Mkdir(d, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if err := requireDir(d); err != nil {
			return err
		}
	}
	return nil
}

func requireDir(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
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
		if !cbt.IsCBTEligibleVolume(vmi, &volume) || isPendingHotplug(&volume, c) {
			logger.V(3).Infof("SKIP Creating QCOW2 overlay for %s", volume.Name)
			continue
		}

		overlayPath, err := cbt.GetQCOW2OverlayPath(vmi, volumeName)
		if err != nil {
			return err
		}
		logger.V(3).Infof("QCOW2 overlay path is %s", overlayPath)

		hotplugStatus, isHotplug := c.HotplugVolumes[volumeName]

		if !shouldCreateQCOW2Overlay(vmi, isHotplug, hotplugStatus.Phase) {
			applyCBTMap[volumeName] = overlayPath
			continue
		}

		isBlock := c.IsBlockPVC[volumeName] || c.IsBlockDV[volumeName]
		imagePath := volumepath.Image(volumeName, isBlock, isHotplug)

		if err := CreateQCOW2Overlay(overlayPath, imagePath); err != nil {
			return err
		}
		applyCBTMap[volumeName] = overlayPath
	}

	c.ApplyCBT = applyCBTMap
	return nil
}

func isPendingHotplug(volume *v1.Volume, c *convertertypes.ConverterContext) bool {
	if !storagetypes.IsHotplugVolume(volume) {
		return false
	}
	return !storagetypes.IsHotplugVolumeAttachable(c.HotplugVolumes[volume.Name].Phase)
}

func ApplyChangedBlockTrackingForMigration(vmi *v1.VirtualMachineInstance, c *convertertypes.ConverterContext) error {
	applyCBTMap := make(map[string]string)
	for _, volume := range vmi.Spec.Volumes {
		if !cbt.IsCBTEligibleVolume(vmi, &volume) || isPendingHotplug(&volume, c) {
			continue
		}
		overlayPath, err := cbt.GetQCOW2OverlayPath(vmi, volume.Name)
		if err != nil {
			return err
		}
		applyCBTMap[volume.Name] = overlayPath
	}
	c.ApplyCBT = applyCBTMap
	return nil
}
