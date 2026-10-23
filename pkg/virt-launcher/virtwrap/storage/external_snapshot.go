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
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"syscall"
	"time"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const (
	overlayFilePrefix = "ovl-"
	overlayFileSuffix = ".qcow2"

	snapshotDiskExternal = "external"
	snapshotDiskSkipped  = "no"

	externalSnapshotFreezeTimeout = 10 * time.Second

	// appended to quiesce failures, which reach the user as vmSnapshot.status.error
	quiesceHint = "; snapshot with the Direct mode for a crash consistent snapshot"
)

// ExternalSnapshot redirects the disks of volumes to qcow2 overlays under
// overlayDir in a single libvirt transaction. It returns once libvirt accepts the
// transaction and reports the result through the domain metadata cache. Idempotent.
func (m *StorageManager) ExternalSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string, volumes []string) error {
	if m.MigrationInProgress() {
		return fmt.Errorf("failed to take an external snapshot, VMI is currently during migration")
	}

	reference, err := expectedOverlayReference(vmi, overlayDir)
	if err != nil {
		return err
	}
	if phase := m.claimSnapshotOverlay(*reference); phase != "" {
		log.Log.Object(vmi).Infof("Not taking an external snapshot, the overlays are in phase %s", phase)
		return nil
	}

	go m.runExternalSnapshot(vmi, overlayDir, volumes)
	return nil
}

// CommitSnapshot starts a live block commit of the overlays under overlayDir
// back into their base images. It returns once the commit has started and
// reports completion through the domain metadata cache. Idempotent.
func (m *StorageManager) CommitSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) error {
	return fmt.Errorf("snapshot overlay commit is not implemented")
}

// claimSnapshotOverlay atomically claims the snapshot transaction.
// Returns an empty phase on success, or the phase that refused the claim.
func (m *StorageManager) claimSnapshotOverlay(reference storagetypes.SnapshotOverlayReference) (refusedBy api.SnapshotOverlayPhase) {
	m.metadataCache.SnapshotOverlay.WithSafeBlock(func(overlay *api.SnapshotOverlayMetadata, _ bool) {
		switch overlay.Phase {
		case "", api.SnapshotOverlaySnapshotFailed, api.SnapshotOverlayCommitted:
			if overlay.Phase != "" && overlay.ContentUID == reference.ContentUID && overlay.ScratchPVCUID == reference.ScratchPVCUID {
				refusedBy = overlay.Phase
				return
			}
			*overlay = api.SnapshotOverlayMetadata{
				ContentUID:    reference.ContentUID,
				ScratchPVCUID: reference.ScratchPVCUID,
				Phase:         api.SnapshotOverlayPreparing,
			}
		default:
			refusedBy = overlay.Phase
		}
	})
	return refusedBy
}

func (m *StorageManager) runExternalSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string, volumes []string) {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.virConn.LookupDomainByName(domName)
	if err != nil || dom == nil {
		// Without a domain observation, existing overlay state is unknown.
		m.failExternalSnapshot(vmi, api.SnapshotOverlayCommitFailed,
			fmt.Errorf("failed to look up domain %s: %v", domName, err))
		return
	}
	defer dom.Free()

	if baseVerified, err := m.takeExternalSnapshot(vmi, dom, overlayDir, volumes); err != nil {
		phase := api.SnapshotOverlaySnapshotFailed
		if !baseVerified {
			phase = m.snapshotFailurePhase(vmi, dom, overlayDir)
		}
		m.failExternalSnapshot(vmi, phase, err)
		return
	}

	m.setSnapshotOverlayPhase(api.SnapshotOverlayReady, "")
	log.Log.Object(vmi).Info("External snapshot taken, all snapshottable disks are on their overlays")
}

func (m *StorageManager) takeExternalSnapshot(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir string, volumes []string) (baseVerified bool, err error) {
	if err := verifyOverlayDirMounted(overlayDir); err != nil {
		return false, err
	}

	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		return false, fmt.Errorf("failed to get the disks of the domain: %w", err)
	}

	selected := snapshottableDisks(volumes, disks)
	if len(selected) == 0 {
		return false, fmt.Errorf("no snapshottable domain disks")
	}
	for _, disk := range selected {
		if onOverlay(disk, overlayDir) {
			return false, fmt.Errorf("disk %s is already on an overlay that this snapshot did not create", disk.Target.Device)
		}
	}
	// No disk is on scratch yet, so a failure from here on leaves every disk on
	// its base image. Existing overlays are still never overwritten.
	if err := emptyOverlayScratch(overlayDir); err != nil {
		return true, err
	}
	snapshotXML, err := buildSnapshotXML(vmi, disks, selected, overlayDir)
	if err != nil {
		return true, err
	}
	log.Log.Object(vmi).V(3).Infof("External snapshot XML: %s", snapshotXML)

	if err := m.createSnapshotQuiesced(vmi, dom, overlayDir, snapshotXML); err != nil {
		return false, err
	}

	return false, verifyAllDisksOnOverlay(dom, volumes, overlayDir)
}

// createSnapshotQuiesced freezes the guest, creates the external snapshot,
// and thaws
func (m *StorageManager) createSnapshotQuiesced(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir, snapshotXML string) error {
	quiesceStart := time.Now()
	if err := m.freezeForSnapshot(vmi, dom); err != nil {
		return err
	}
	frozenAt := time.Now()
	log.Log.Object(vmi).Infof("Guest filesystems frozen for the external snapshot, quiescing took %s", frozenAt.Sub(quiesceStart))

	defer func() {
		if err := m.UnfreezeVMI(vmi); err != nil {
			log.Log.Object(vmi).Reason(err).Error("Failed to thaw the guest filesystems after the external snapshot")
			go m.scheduleSafetyVMIUnfreeze(vmi, externalSnapshotFreezeTimeout)
			return
		}
		log.Log.Object(vmi).Infof("Guest filesystems thawed, frozen for %s", time.Since(frozenAt))
	}()
	thawTimer := time.AfterFunc(externalSnapshotFreezeTimeout, func() {
		if err := m.UnfreezeVMI(vmi); err != nil {
			log.Log.Object(vmi).Reason(err).Error("External snapshot safety thaw failed")
		}
	})
	defer thawTimer.Stop()

	if err := verifyOverlayDirMounted(overlayDir); err != nil {
		return err
	}
	snapshot, err := dom.CreateSnapshotXML(snapshotXML,
		libvirt.DOMAIN_SNAPSHOT_CREATE_DISK_ONLY|
			libvirt.DOMAIN_SNAPSHOT_CREATE_ATOMIC|
			libvirt.DOMAIN_SNAPSHOT_CREATE_NO_METADATA)
	if err != nil {
		return fmt.Errorf("failed to create the external snapshot: %w", err)
	}
	if snapshot != nil {
		snapshot.Free()
	}
	if time.Since(quiesceStart) >= externalSnapshotFreezeTimeout {
		return fmt.Errorf("external snapshot exceeded its guest filesystem freeze timeout")
	}
	return nil
}

// freezeForSnapshot freezes the guest filesystems via the guest agent
func (m *StorageManager) freezeForSnapshot(vmi *v1.VirtualMachineInstance, dom cli.VirDomain) error {
	state, _, err := dom.GetState()
	if err != nil {
		return fmt.Errorf("failed to get the state of the domain: %w", err)
	}
	if state == libvirt.DOMAIN_PAUSED {
		return fmt.Errorf("cannot take an external snapshot of a paused domain, its filesystems cannot be quiesced" + quiesceHint)
	}
	if err := m.FreezeVMI(vmi, 0); err != nil {
		return fmt.Errorf("failed to freeze the guest filesystems, which an external snapshot requires: %w"+quiesceHint, err)
	}
	return nil
}

// snapshotFailurePhase returns SnapshotFailed if all disks are back on their
// base images, CommitFailed otherwise
func (m *StorageManager) snapshotFailurePhase(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir string) api.SnapshotOverlayPhase {
	_, err := expectedOverlayReference(vmi, overlayDir)
	if err == nil {
		err = verifyOverlayDirMounted(overlayDir)
	}
	if err == nil {
		err = verifyAllDisksOnBase(dom, overlayDir)
	}
	if err != nil {
		log.Log.Object(vmi).Reason(err).Warning("Cannot confirm that the disks are back on their base images")
		return api.SnapshotOverlayCommitFailed
	}
	return api.SnapshotOverlaySnapshotFailed
}

func (m *StorageManager) failExternalSnapshot(vmi *v1.VirtualMachineInstance, phase api.SnapshotOverlayPhase, cause error) {
	log.Log.Object(vmi).Reason(cause).Errorf("External snapshot failed, reporting %s", phase)
	m.setSnapshotOverlayPhase(phase, cause.Error())
}

func (m *StorageManager) setSnapshotOverlayPhase(phase api.SnapshotOverlayPhase, message string) {
	m.metadataCache.SnapshotOverlay.WithSafeBlock(func(overlay *api.SnapshotOverlayMetadata, _ bool) {
		overlay.Phase = phase
		overlay.Message = message
	})
}

// buildSnapshotXML builds the domainsnapshot XML. Every disk must be listed,
// libvirt snapshots the ones it was not told about
func buildSnapshotXML(vmi *v1.VirtualMachineInstance, disks, selected []api.Disk, overlayDir string) (string, error) {
	snapshotted := map[string]bool{}
	for _, disk := range selected {
		snapshotted[disk.Target.Device] = true
	}

	domainSnapshot := &api.DomainSnapshot{
		// NO_METADATA, so libvirt keeps no snapshot object to collide with
		Name:          vmi.Name + "-overlay",
		SnapshotDisks: &api.SnapshotDisks{},
	}
	for _, disk := range disks {
		target := disk.Target.Device
		if target == "" {
			continue
		}

		snapshotDisk := api.SnapshotDisk{Name: target, Snapshot: snapshotDiskSkipped}
		if snapshotted[target] {
			snapshotDisk.Snapshot = snapshotDiskExternal
			snapshotDisk.Source = &api.SnapshotDiskSource{File: overlayPath(overlayDir, disk.Alias.GetName())}
		}
		domainSnapshot.SnapshotDisks.Disks = append(domainSnapshot.SnapshotDisks.Disks, snapshotDisk)
	}

	snapshotXML, err := xml.Marshal(domainSnapshot)
	if err != nil {
		return "", fmt.Errorf("failed to marshal the snapshot XML: %w", err)
	}
	return string(snapshotXML), nil
}

// snapshottableDisks returns the disks of the volumes the snapshot captures
func snapshottableDisks(volumes []string, disks []api.Disk) []api.Disk {
	var snapshottableDisks []api.Disk
	for _, disk := range disks {
		if disk.Alias != nil && disk.Target.Device != "" && slices.Contains(volumes, disk.Alias.GetName()) {
			snapshottableDisks = append(snapshottableDisks, disk)
		}
	}
	return snapshottableDisks
}

// verifyAllDisksOnOverlay checks the domain, not the filesystem. libvirt creates
// the overlay files before it switches the domain over
func verifyAllDisksOnOverlay(dom cli.VirDomain, volumes []string, overlayDir string) error {
	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		return fmt.Errorf("failed to get the disks of the domain: %w", err)
	}
	for _, disk := range snapshottableDisks(volumes, disks) {
		if expected := overlayPath(overlayDir, disk.Alias.GetName()); disk.Source.File != expected {
			return fmt.Errorf("disk %s is on %q, expected it to be on its overlay %q",
				disk.Target.Device, disk.Source.File, expected)
		}
	}
	return nil
}

// verifyAllDisksOnBase checks that no disk is being written through an overlay
func verifyAllDisksOnBase(dom cli.VirDomain, overlayDir string) error {
	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		return fmt.Errorf("failed to get the disks of the domain: %w", err)
	}
	for _, disk := range disks {
		if onOverlay(disk, overlayDir) {
			return fmt.Errorf("disk %s is still on its overlay %q", disk.Target.Device, disk.Source.File)
		}
	}
	return nil
}

// verifyOverlayDirMounted checks that overlayDir is a mount point and not the
// container filesystem
var verifyOverlayDirMounted = func(overlayDir string) error {
	var dirStat, parentStat syscall.Stat_t
	if err := syscall.Stat(overlayDir, &dirStat); err != nil {
		return fmt.Errorf("failed to stat the overlay directory %s: %w", overlayDir, err)
	}
	if err := syscall.Stat(filepath.Dir(overlayDir), &parentStat); err != nil {
		return fmt.Errorf("failed to stat the parent of the overlay directory %s: %w", overlayDir, err)
	}
	if dirStat.Dev == parentStat.Dev {
		return fmt.Errorf("the overlay directory %s is not a mount point, the scratch volume is not attached", overlayDir)
	}
	return nil
}

// onOverlay reports a disk written through its overlay on the scratch volume
func onOverlay(disk api.Disk, overlayDir string) bool {
	return disk.Alias != nil && disk.Source.File == overlayPath(overlayDir, disk.Alias.GetName())
}

func overlayPath(overlayDir, volumeName string) string {
	return filepath.Join(overlayDir, overlayFilePrefix+volumeName+overlayFileSuffix)
}

var overlayVolumeMountDir = hotplugdisk.GetVolumeMountDir

func expectedOverlayReference(vmi *v1.VirtualMachineInstance, overlayDir string) (*storagetypes.SnapshotOverlayReference, error) {
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return nil, err
	}
	if reference == nil {
		return nil, fmt.Errorf("external snapshot transaction reference is missing")
	}
	if overlayDir == "" || overlayScratchDir(vmi) != overlayDir {
		return nil, fmt.Errorf("expected scratch utility volume is not attached")
	}
	return reference, nil
}

func overlayScratchDir(vmi *v1.VirtualMachineInstance) string {
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil || reference == nil {
		return ""
	}
	name := storagetypes.ScratchPVCName(reference.ContentUID)
	for _, volume := range vmi.Spec.UtilityVolumes {
		if volume.Name == name && volume.ClaimName == name && volume.Type != nil && *volume.Type == v1.SnapshotOverlay {
			return filepath.Clean(overlayVolumeMountDir(volume.Name))
		}
	}
	return ""
}

func emptyOverlayScratch(overlayDir string) error {
	entries, err := os.ReadDir(overlayDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == "lost+found" && entry.IsDir() {
			// Filesystem-owned; the unprivileged launcher cannot read it.
			continue
		}
		return fmt.Errorf("scratch directory is not empty; refusing to overwrite snapshot overlays")
	}
	return nil
}
