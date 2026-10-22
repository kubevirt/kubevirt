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
	"io/fs"
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

	if _, err := osdisk.QEMUImg(ctx, extraFiles, "create", "-f", "qcow2",
		"-o", fmt.Sprintf("data_file=%s,data_file_raw=on,cluster_size=%d", scratchPath, overlayClusterSize),
		tmpPath, strconv.FormatInt(info.VirtualSize, 10)); err != nil {
		return err
	}
	if _, err := osdisk.QEMUImg(ctx, extraFiles, "amend", "-f", "qcow2", "-o", "data_file="+imagePath, tmpPath); err != nil {
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
	if err := os.Remove(target); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("failed to delete QCOW2 overlay %s: %w", target, err)
	}
	log.Log.Infof("QCOW2 overlay %s deleted", target)
	return nil
}

// CleanupChangedBlockTrackingOverlays removes all QCOW2 overlays under the CBT
// directory when CBT is not active on the VMI. Used after CBT disable when the
// backend PVC is retained (e.g. TPM/EFI). Disabled is the common case and may
// run on every SyncVMI until the directory is empty; nil/Undefined are effectively
// no-ops when no overlays exist.
//
// domainSpec is the live domain after syncDisks. If any disk source under the
// CBT dir is still referenced, wipe is skipped (not an error) so SyncVMI cannot
// wedge on a state-machine bug that cleared VMI CBT status while QEMU still has
// overlays open. Orphans are retried on a later SyncVMI once the domain no
// longer references them.
func CleanupChangedBlockTrackingOverlays(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) error {
	// Defensive: callers already gate on !HasCBTStateEnabled; keep the guard so
	// this helper stays safe if invoked alone.
	if cbt.HasCBTStateEnabled(vmi.Status.ChangedBlockTracking) {
		return nil
	}
	cbtDir := cbt.PathForCBT(vmi)
	if liveDomainReferencesCBTOverlays(domainSpec, cbtDir) {
		log.Log.Object(vmi).Warningf("skipping CBT overlay cleanup: live domain still references paths under %s", cbtDir)
		return nil
	}
	return reconcileQCOW2OverlaysInDir(cbtDir, nil)
}

// ReconcileChangedBlockTrackingOverlays is a GC pass for overlays whose volumes
// are no longer on the VMI. Hot-unplug deletes the matching overlay via
// DeleteQCOW2Overlay during syncDisks; this reconciler catches orphans that miss
// that path (and removes leftover *.qcow2.tmp files). Must run after disk detach
// so QEMU is not still referencing a path that is about to be unlinked.
//
// The keep-set combines:
//  1. Paths from GetQCOW2OverlayPath for each eligible volume (whatever layout
//     that helper currently uses), and
//  2. Disk source paths under the CBT dir still referenced by the live domain.
//
// The live-domain set matters across launcher upgrades / migrations: the running
// domain may still use an older overlay layout than this launcher would compute
// (see #19404 / #19409). Never unlink a file QEMU still has open.
//
// domainSpec must reflect the domain after syncDisks (re-read), not the pre-detach
// snapshot — otherwise GC would keep overlays for disks that were just detached.
func ReconcileChangedBlockTrackingOverlays(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) error {
	cbtDir, desiredPaths := buildCBTOverlayKeepPaths(vmi, domainSpec)
	return reconcileQCOW2OverlaysInDir(cbtDir, desiredPaths)
}

// buildCBTOverlayKeepPaths returns the CBT directory and the set of overlay
// paths that must not be deleted: eligible VMI volumes plus any live-domain
// disk sources still under that directory.
func buildCBTOverlayKeepPaths(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) (string, map[string]struct{}) {
	cbtDir := cbt.PathForCBT(vmi)
	desiredPaths := make(map[string]struct{})
	for _, volume := range vmi.Spec.Volumes {
		if !cbt.IsCBTEligibleVolume(&volume) {
			continue
		}
		desiredPaths[cbt.GetQCOW2OverlayPath(vmi, volume.Name)] = struct{}{}
	}
	addLiveDomainCBTOverlayPaths(desiredPaths, cbtDir, domainSpec)
	return cbtDir, desiredPaths
}

// addLiveDomainCBTOverlayPaths keeps any disk Source.File under cbtDir that the
// live domain still references, so GC cannot unlink an overlay QEMU has open.
func addLiveDomainCBTOverlayPaths(desiredPaths map[string]struct{}, cbtDir string, domainSpec *api.DomainSpec) {
	// Writing to a nil map panics; callers always pass a non-nil map.
	if domainSpec == nil || desiredPaths == nil {
		return
	}
	for _, disk := range domainSpec.Devices.Disks {
		path := disk.Source.File
		if path == "" || !isPathUnderDir(path, cbtDir) {
			continue
		}
		desiredPaths[path] = struct{}{}
	}
}

func liveDomainReferencesCBTOverlays(domainSpec *api.DomainSpec, cbtDir string) bool {
	if domainSpec == nil {
		return false
	}
	for _, disk := range domainSpec.Devices.Disks {
		if path := disk.Source.File; path != "" && isPathUnderDir(path, cbtDir) {
			return true
		}
	}
	return false
}

func isPathUnderDir(path, dir string) bool {
	cleanPath := filepath.Clean(path)
	cleanDir := filepath.Clean(dir)
	rel, err := filepath.Rel(cleanDir, cleanPath)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func reconcileQCOW2OverlaysInDir(cbtDir string, desiredPaths map[string]struct{}) error {
	err := filepath.WalkDir(cbtDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".qcow2") && !strings.HasSuffix(name, ".qcow2.tmp") {
			return nil
		}
		// Only delete regular files — never follow or remove symlinks/devices.
		info, infoErr := d.Info()
		if infoErr != nil {
			if errors.Is(infoErr, os.ErrNotExist) {
				return nil
			}
			return infoErr
		}
		if !info.Mode().IsRegular() {
			log.Log.Warningf("skipping non-regular CBT path %s (mode %s)", path, info.Mode())
			return nil
		}
		if _, keep := desiredPaths[path]; keep {
			return nil
		}
		// Safe after domain detach: guest data lives in the raw data_file, not this metadata overlay.
		if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("failed to delete orphaned QCOW2 overlay %s: %w", path, removeErr)
		}
		log.Log.Infof("QCOW2 overlay %s deleted (volume no longer in CBT set)", path)
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
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

func ApplyChangedBlockTrackingForMigration(vmi *v1.VirtualMachineInstance, c *convertertypes.ConverterContext) error {
	applyCBTMap := make(map[string]string)
	for _, volume := range vmi.Spec.Volumes {
		if cbt.IsCBTEligibleVolume(&volume) {
			applyCBTMap[volume.Name] = cbt.GetQCOW2OverlayPath(vmi, volume.Name)
		}
	}
	c.ApplyCBT = applyCBTMap
	return nil
}
