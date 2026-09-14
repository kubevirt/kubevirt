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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"libvirt.org/go/libvirt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
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

	// what <mirror> reports when the commit has drained the overlay and can pivot
	mirrorReady = "yes"

	// sized so a slow waiter never blocks the libvirt event loop
	blockJobEventBuffer = 16
)

// Timings of a block commit, variables so that tests can shorten them
var (
	// how long a commit runs before the guest is paused to help it converge
	commitConvergenceGracePeriod = 4 * time.Minute
	// the whole budget for one disk, the pause included
	commitReadyTimeout = 5 * time.Minute
	// the event subscription is volatile and lost on a libvirt reconnect, so
	// readiness is also polled
	commitReadyPollInterval = 15 * time.Second
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

// CommitSnapshot starts a live block commit of the overlays under overlayDir
// back into their base images. Returns once the commit has started, completion
// is reported through the domain metadata cache. Idempotent.
func (m *StorageManager) CommitSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) error {
	if m.MigrationInProgress() {
		return fmt.Errorf("failed to commit the snapshot overlays, VMI is currently during migration")
	}

	if claimed, phase := m.claimSnapshotCommit(); !claimed {
		log.Log.Object(vmi).Infof("Not committing the snapshot overlays, they are in phase %q", phase)
		return nil
	}

	go m.runCommitSnapshot(vmi, overlayDir)
	return nil
}

// claimSnapshotCommit atomically claims the commit. Returns false and the phase
// that refused it when the overlays are not in a committable state.
func (m *StorageManager) claimSnapshotCommit() (claimed bool, refusedBy api.SnapshotOverlayPhase) {
	m.metadataCache.SnapshotOverlay.WithSafeBlock(func(overlay *api.SnapshotOverlayMetadata, _ bool) {
		switch overlay.Phase {
		case api.SnapshotOverlayReady, api.SnapshotOverlayCommitFailed:
			overlay.Phase = api.SnapshotOverlayCommitting
			overlay.Message = ""
			claimed = true
		default:
			refusedBy = overlay.Phase
		}
	})
	return claimed, refusedBy
}

func (m *StorageManager) runCommitSnapshot(vmi *v1.VirtualMachineInstance, overlayDir string) {
	domName := api.VMINamespaceKeyFunc(vmi)
	dom, err := m.virConn.LookupDomainByName(domName)
	if err != nil || dom == nil {
		m.failCommitSnapshot(vmi, fmt.Errorf("failed to look up domain %s: %v", domName, err))
		return
	}
	defer dom.Free()

	if err := m.commitAllDisks(vmi, dom, overlayDir); err != nil {
		m.failCommitSnapshot(vmi, err)
		return
	}

	if err := verifyAllDisksOnBase(vmi, dom, overlayDir); err != nil {
		m.failCommitSnapshot(vmi, err)
		return
	}

	if err := removeOverlays(vmi, dom, overlayDir); err != nil {
		m.failCommitSnapshot(vmi, err)
		return
	}

	m.clearSnapshotOverlay()
	log.Log.Object(vmi).Info("Snapshot overlays committed, all snapshottable disks are back on their base images")
}

// commitAllDisks commits one disk at a time. Parallel jobs compete for the
// bandwidth the guest is dirtying the overlays with.
func (m *StorageManager) commitAllDisks(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir string) error {
	disks, err := currentSnapshottableDisks(vmi, dom)
	if err != nil {
		return err
	}

	events, err := subscribeBlockJobEvents(m.virConn)
	if err != nil {
		return err
	}
	defer events.close(vmi)

	for _, disk := range disks {
		if err := m.commitDisk(vmi, dom, disk, overlayDir, events.events); err != nil {
			return err
		}
	}
	return nil
}

func (m *StorageManager) commitDisk(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, disk api.Disk, overlayDir string, events <-chan libvirt.DomainEventBlockJob) error {
	target := disk.Target.Device
	if filepath.Dir(disk.Source.File) != overlayDir {
		log.Log.Object(vmi).V(3).Infof("Disk %s is already on its base image, nothing to commit", target)
		return nil
	}

	running, err := blockCommitRunning(dom, target)
	if err != nil {
		return err
	}
	if running {
		log.Log.Object(vmi).Infof("Disk %s already has a block commit running, waiting for it to converge", target)
	} else if err := dom.BlockCommit(target, "", "", 0, libvirt.DOMAIN_BLOCK_COMMIT_ACTIVE); err != nil {
		return fmt.Errorf("failed to start the block commit of disk %s: %w", target, err)
	}

	// Deferred, so the guest is resumed after the pivot on the way out and on
	// every error path too: leaving it paused is worse than a failed commit.
	var paused guestPause
	defer paused.resume(vmi, dom)

	commitStart := time.Now()
	if err := m.waitForCommitReady(vmi, dom, target, events, &paused); err != nil {
		return err
	}
	return pivotDisk(vmi, dom, target, overlayDir, commitStart)
}

// waitForCommitReady blocks until the block commit of one disk has drained its
// overlay, driven by block job events with a domain XML poll as a fallback
func (m *StorageManager) waitForCommitReady(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, target string, events <-chan libvirt.DomainEventBlockJob, paused *guestPause) error {
	// The job may already be ready on a retry, with no event left to come
	ready, err := blockCommitReady(dom, target)
	if err != nil {
		return err
	}
	if ready {
		return nil
	}

	start := time.Now()
	poll := time.NewTicker(commitReadyPollInterval)
	defer poll.Stop()
	forceConvergence := time.NewTimer(commitConvergenceGracePeriod)
	defer forceConvergence.Stop()
	deadline := time.NewTimer(commitReadyTimeout)
	defer deadline.Stop()

	for {
		select {
		case event := <-events:
			if event.Disk != target {
				continue
			}
			switch event.Status {
			case libvirt.DOMAIN_BLOCK_JOB_READY, libvirt.DOMAIN_BLOCK_JOB_COMPLETED:
				return nil
			case libvirt.DOMAIN_BLOCK_JOB_FAILED, libvirt.DOMAIN_BLOCK_JOB_CANCELED:
				return fmt.Errorf("the block commit of disk %s ended with status %d", target, event.Status)
			}
		case <-poll.C:
			// Skip the XML parse while the job still reports bytes outstanding
			if info, err := dom.GetBlockJobInfo(target, 0); err == nil && info != nil && info.End > 0 && info.Cur < info.End {
				log.Log.Object(vmi).Infof("The block commit of disk %s has copied %d of %d MiB after %s",
					target, info.Cur/storagetypes.MiB, info.End/storagetypes.MiB, time.Since(start).Round(time.Second))
				continue
			}

			ready, err := blockCommitReady(dom, target)
			if err != nil {
				return err
			}
			if ready {
				return nil
			}
		case <-forceConvergence.C:
			if err := paused.forceConvergence(vmi, dom, target); err != nil {
				return err
			}
		case <-deadline.C:
			return fmt.Errorf("the block commit of disk %s did not converge within %v", target, commitReadyTimeout)
		}
	}
}

// guestPause tracks whether this commit is the thing that paused the guest, so
// that a guest somebody else paused is never resumed underneath them.
type guestPause struct {
	paused bool
}

func (p *guestPause) forceConvergence(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, target string) error {
	if p.paused {
		return nil
	}

	state, _, err := dom.GetState()
	if err != nil {
		return fmt.Errorf("failed to get the state of the domain: %w", err)
	}
	if state != libvirt.DOMAIN_RUNNING {
		return nil
	}

	log.Log.Object(vmi).Infof("The block commit of disk %s is not converging, pausing the guest to let it finish", target)
	if err := dom.Suspend(); err != nil {
		return fmt.Errorf("failed to pause the guest to converge the block commit of disk %s: %w", target, err)
	}
	p.paused = true
	return nil
}

func (p *guestPause) resume(vmi *v1.VirtualMachineInstance, dom cli.VirDomain) {
	if !p.paused {
		return
	}
	if err := dom.Resume(); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to resume the guest after the block commit")
		return
	}
	p.paused = false
	log.Log.Object(vmi).Info("Guest resumed after the block commit")
}

// pivotDisk ends the block job and switches the domain back to its base image
func pivotDisk(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, target, overlayDir string, commitStart time.Time) error {
	if err := dom.BlockJobAbort(target, libvirt.DOMAIN_BLOCK_JOB_ABORT_PIVOT); err != nil {
		// The job can pivot itself before this call, so check the domain rather
		// than parse the error
		if onBase, baseErr := diskOnBase(dom, target, overlayDir); baseErr == nil && onBase {
			log.Log.Object(vmi).Infof("Disk %s is already on its base image, the block commit had pivoted itself", target)
			return nil
		}
		return fmt.Errorf("failed to pivot disk %s onto its base image: %w", target, err)
	}

	log.Log.Object(vmi).Infof("Disk %s pivoted onto its base image after %s", target, time.Since(commitStart).Round(time.Millisecond))
	return nil
}

// removeOverlays deletes the overlay files once every disk is back on its base
// image. The pre-start hook commits any overlay it finds on the next boot.
func removeOverlays(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, overlayDir string) error {
	disks, err := currentSnapshottableDisks(vmi, dom)
	if err != nil {
		return err
	}

	for _, disk := range disks {
		path := overlayPath(overlayDir, disk.Target.Device)
		if err := removeOverlayFile(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove the overlay %s: %w", path, err)
		}
	}
	return nil
}

var removeOverlayFile = os.Remove

// clearSnapshotOverlay resets the transaction state so the VMI can be
// snapshotted again
func (m *StorageManager) clearSnapshotOverlay() {
	m.metadataCache.SnapshotOverlay.WithSafeBlock(func(overlay *api.SnapshotOverlayMetadata, _ bool) {
		*overlay = api.SnapshotOverlayMetadata{}
	})
}

func (m *StorageManager) failCommitSnapshot(vmi *v1.VirtualMachineInstance, cause error) {
	log.Log.Object(vmi).Reason(cause).Error("Snapshot overlay commit failed, reporting CommitFailed")
	m.setSnapshotOverlayPhase(api.SnapshotOverlayCommitFailed, cause.Error())
}

// blockJobEventStream subscribes to block job events for one commit
type blockJobEventStream struct {
	conn           cli.Connection
	registrationID int
	events         chan libvirt.DomainEventBlockJob
}

func subscribeBlockJobEvents(conn cli.Connection) (*blockJobEventStream, error) {
	stream := &blockJobEventStream{
		conn:   conn,
		events: make(chan libvirt.DomainEventBlockJob, blockJobEventBuffer),
	}

	registrationID, err := conn.VolatileDomainEventBlockJobRegister(
		func(_ *libvirt.Connect, _ *libvirt.Domain, event *libvirt.DomainEventBlockJob) {
			// runs on the libvirt event loop, which must never block
			select {
			case stream.events <- *event:
			default:
			}
		})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to block job events: %w", err)
	}

	stream.registrationID = registrationID
	return stream, nil
}

func (s *blockJobEventStream) close(vmi *v1.VirtualMachineInstance) {
	if err := s.conn.DomainEventDeregister(s.registrationID); err != nil {
		log.Log.Object(vmi).Reason(err).Error("Failed to unsubscribe from block job events")
	}
}

// blockCommitRunning reports whether the disk already has this commit running,
// and errors on a disk busy with a different block job
func blockCommitRunning(dom cli.VirDomain, target string) (bool, error) {
	info, err := dom.GetBlockJobInfo(target, 0)
	if err != nil {
		return false, fmt.Errorf("failed to get the block job info of disk %s: %w", target, err)
	}
	if info == nil || info.Type == libvirt.DOMAIN_BLOCK_JOB_TYPE_UNKNOWN {
		return false, nil
	}
	if info.Type != libvirt.DOMAIN_BLOCK_JOB_TYPE_ACTIVE_COMMIT {
		return false, fmt.Errorf("disk %s is busy with a block job of type %d", target, info.Type)
	}
	return true, nil
}

func blockCommitReady(dom cli.VirDomain, target string) (bool, error) {
	disk, err := domainDisk(dom, target)
	if err != nil {
		return false, err
	}
	return disk.Mirror != nil && disk.Mirror.Ready == mirrorReady, nil
}

func diskOnBase(dom cli.VirDomain, target, overlayDir string) (bool, error) {
	disk, err := domainDisk(dom, target)
	if err != nil {
		return false, err
	}
	return filepath.Dir(disk.Source.File) != overlayDir, nil
}

func domainDisk(dom cli.VirDomain, target string) (api.Disk, error) {
	disks, err := util.GetAllDomainDisks(dom)
	if err != nil {
		return api.Disk{}, fmt.Errorf("failed to get the disks of the domain: %w", err)
	}
	for _, disk := range disks {
		if disk.Target.Device == target {
			return disk, nil
		}
	}
	return api.Disk{}, fmt.Errorf("disk %s is no longer attached to the domain", target)
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
