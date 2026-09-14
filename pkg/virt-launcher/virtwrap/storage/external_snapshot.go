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
	"path/filepath"
	"syscall"

	"libvirt.org/go/libvirt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

const (
	overlayFilePrefix = "ovl-"
	overlayFileSuffix = ".qcow2"

	snapshotDiskExternal = "external"
	snapshotDiskSkipped  = "no"

	// appended to quiesce failures, which reach the user as vmSnapshot.status.error
	quiesceHint = "; snapshot with the Direct mode for a crash consistent snapshot"
)

// ExternalSnapshot redirects every snapshottable disk to a qcow2 overlay under
// overlayDir, in a single libvirt transaction. Returns once libvirt accepts it,
// the result is reported through the domain metadata cache. Idempotent.
func (m *StorageManager) ExternalSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) error {
	if m.MigrationInProgress() {
		return fmt.Errorf("failed to take an external snapshot, VMI is currently during migration")
	}

	if phase := m.claimSnapshotOverlay(); phase != "" {
		log.Log.Object(vmi).Infof("Not taking an external snapshot, the overlays are in phase %s", phase)
		return nil
	}

	go m.runExternalSnapshot(vmi, overlayDir)
	return nil
}

// CommitSnapshot starts a live block-commit of the overlays under overlayDir
// back into their base images, and returns as soon as it has been started.
// Completion is reported through the metadata cache. The controller re-issues
// the RPC every reconcile: a second call while a commit runs is a no-op, and a
// retry after a failed one resumes from whatever the domain is running on.
func (m *StorageManager) CommitSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) error {
	return fmt.Errorf("snapshot overlay commit is not implemented")
}

// claimSnapshotOverlay atomically claims the snapshot transaction.
// Returns an empty phase on success, or the phase that refused the claim.
func (m *StorageManager) claimSnapshotOverlay() (refusedBy api.SnapshotOverlayPhase) {
	m.metadataCache.SnapshotOverlay.WithSafeBlock(func(overlay *api.SnapshotOverlayMetadata, _ bool) {
		switch overlay.Phase {
		case "", api.SnapshotOverlaySnapshotFailed:
			now := metav1.Now()
			*overlay = api.SnapshotOverlayMetadata{
				Phase:          api.SnapshotOverlayInProgress,
				StartTimestamp: &now,
			}
		default:
			refusedBy = overlay.Phase
		}
	})
	return refusedBy
}

func (m *StorageManager) runExternalSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.virConn.LookupDomainByName(domName)
	if err != nil || dom == nil {
		// The transaction never reached libvirt, the disks are still on base
		m.failExternalSnapshot(vmi, api.SnapshotOverlaySnapshotFailed,
			fmt.Errorf("failed to look up domain %s: %v", domName, err))
		return
	}
	defer dom.Free()

	if err := m.takeExternalSnapshot(vmi, dom, overlayDir); err != nil {
		m.failExternalSnapshot(vmi, m.snapshotFailurePhase(vmi, dom, overlayDir), err)
		return
	}

	m.setSnapshotOverlayPhase(api.SnapshotOverlayReady, "")
	log.Log.Object(vmi).Info("External snapshot taken, all snapshottable disks are on their overlays")
}

func (m *StorageManager) takeExternalSnapshot(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir string) error {
	if err := verifyOverlayDirMounted(overlayDir); err != nil {
		return err
	}

	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		return fmt.Errorf("failed to get the disks of the domain: %w", err)
	}

	snapshotXML, err := buildSnapshotXML(vmi, disks, overlayDir)
	if err != nil {
		return err
	}
	log.Log.Object(vmi).V(3).Infof("External snapshot XML: %s", snapshotXML)

	if err := m.createSnapshotQuiesced(vmi, dom, snapshotXML); err != nil {
		return err
	}

	return verifyAllDisksOnOverlay(vmi, dom, overlayDir)
}

// createSnapshotQuiesced freezes the guest, creates the external snapshot,
// and thaws
func (m *StorageManager) createSnapshotQuiesced(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, snapshotXML string) error {
	quiesceStart := time.Now()
	if err := m.freezeForSnapshot(vmi, dom); err != nil {
		return err
	}
	frozenAt := time.Now()
	log.Log.Object(vmi).Infof("Guest filesystems frozen for the external snapshot, quiescing took %s", frozenAt.Sub(quiesceStart))

	defer func() {
		if err := m.UnfreezeVMI(vmi); err != nil {
			log.Log.Object(vmi).Reason(err).Error("Failed to thaw the guest filesystems after the external snapshot")
			return
		}
		log.Log.Object(vmi).Infof("Guest filesystems thawed, frozen for %s", time.Since(frozenAt))
	}()

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
	if !guestAgentConnected(vmi) {
		return fmt.Errorf("cannot take an external snapshot without a connected guest agent, the guest filesystems cannot be quiesced" + quiesceHint)
	}

	if err := m.FreezeVMI(vmi, 0); err != nil {
		return fmt.Errorf("failed to freeze the guest filesystems, which an external snapshot requires: %w"+quiesceHint, err)
	}
	return nil
}

func guestAgentConnected(vmi *v1.VirtualMachineInstance) bool {
	for _, condition := range vmi.Status.Conditions {
		if condition.Type == v1.VirtualMachineInstanceAgentConnected {
			return condition.Status == k8sv1.ConditionTrue
		}
	}
	return false
}

// snapshotFailurePhase returns SnapshotFailed if all disks are back on their
// base images, CommitFailed otherwise
func (m *StorageManager) snapshotFailurePhase(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir string) api.SnapshotOverlayPhase {
	if err := verifyAllDisksOnBase(vmi, dom, overlayDir); err != nil {
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
func buildSnapshotXML(vmi *v1.VirtualMachineInstance, disks []api.Disk, overlayDir string) (string, error) {
	snapshotted := map[string]bool{}
	for _, disk := range snapshottableDisks(vmi, disks) {
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
			snapshotDisk.Source = &api.SnapshotDiskSource{File: overlayPath(overlayDir, target)}
		}
		domainSnapshot.SnapshotDisks.Disks = append(domainSnapshot.SnapshotDisks.Disks, snapshotDisk)
	}

	snapshotXML, err := xml.Marshal(domainSnapshot)
	if err != nil {
		return "", fmt.Errorf("failed to marshal the snapshot XML: %w", err)
	}
	return string(snapshotXML), nil
}

// snapshottableDisks returns the disks to redirect, matching
// isVolumeSnapshottable in the snapshot controller. Keyed on the volume, so the
// scratch volume is excluded: utility volumes are not in vmi.Spec.Volumes
func snapshottableDisks(vmi *v1.VirtualMachineInstance, disks []api.Disk) []api.Disk {
	snapshottable := map[string]bool{}
	for _, volume := range vmi.Spec.Volumes {
		if volume.VolumeSource.PersistentVolumeClaim != nil ||
			volume.VolumeSource.DataVolume != nil ||
			volume.VolumeSource.MemoryDump != nil {
			snapshottable[volume.Name] = true
		}
	}

	var snapshottableDisks []api.Disk
	for _, disk := range disks {
		if disk.Alias == nil || disk.Target.Device == "" {
			continue
		}
		if snapshottable[converter.GetVolumeNameByDisk(disk)] {
			snapshottableDisks = append(snapshottableDisks, disk)
		}
	}
	return snapshottableDisks
}

// verifyAllDisksOnOverlay checks the domain, not the filesystem. libvirt creates
// the overlay files before it switches the domain over
func verifyAllDisksOnOverlay(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir string) error {
	disks, err := currentSnapshottableDisks(vmi, dom)
	if err != nil {
		return err
	}
	for _, disk := range disks {
		if expected := overlayPath(overlayDir, disk.Target.Device); disk.Source.File != expected {
			return fmt.Errorf("disk %s is on %q, expected it to be on its overlay %q",
				disk.Target.Device, disk.Source.File, expected)
		}
	}
	return nil
}

// verifyAllDisksOnBase checks that no disk is being written through an overlay
func verifyAllDisksOnBase(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir string) error {
	disks, err := currentSnapshottableDisks(vmi, dom)
	if err != nil {
		return err
	}
	for _, disk := range disks {
		if filepath.Dir(disk.Source.File) == overlayDir {
			return fmt.Errorf("disk %s is still on its overlay %q", disk.Target.Device, disk.Source.File)
		}
	}
	return nil
}

func currentSnapshottableDisks(vmi *v1.VirtualMachineInstance, dom cli.VirDomain) ([]api.Disk, error) {
	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		return nil, fmt.Errorf("failed to get the disks of the domain: %w", err)
	}
	return snapshottableDisks(vmi, disks), nil
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

func overlayPath(overlayDir, target string) string {
	return filepath.Join(overlayDir, overlayFilePrefix+target+overlayFileSuffix)
}
