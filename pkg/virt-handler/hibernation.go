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

package virthandler

import (
	"fmt"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/hibernation"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	"kubevirt.io/kubevirt/pkg/storage/utilityvolume"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// saveStartTimeout bounds the time from the commit to a save until
// virt-launcher accepts it. Past it, a save virt-launcher never started is
// reported failed, which ends the hibernation and keeps the guest running.
const saveStartTimeout = 2 * time.Minute

// saveTarget returns the path virt-launcher saves the state of vmi to and its
// file name, if the VM controller armed a save: a save target utility volume
// is in the spec of vmi and mounted in its pod.
func saveTarget(vmi *v1.VirtualMachineInstance) (path, fileName string, ok bool) {
	for _, volume := range vmi.Spec.UtilityVolumes {
		if !hibernation.IsSaveTargetVolume(volume) || !utilityvolume.Attached(vmi, volume.Name) {
			continue
		}
		fileName = hibernation.FileNameFor(volume.ClaimName)
		return filepath.Join(hotplugdisk.GetVolumeMountDir(volume.Name), fileName), fileName, true
	}
	return "", "", false
}

// armedFile returns the file name of the save the VM controller armed on vmi, if any.
func armedFile(vmi *v1.VirtualMachineInstance) string {
	for _, volume := range vmi.Spec.UtilityVolumes {
		if hibernation.IsSaveTargetVolume(volume) {
			return hibernation.FileNameFor(volume.ClaimName)
		}
	}
	return ""
}

// saveState performs the save the VM controller armed on vmi, in two steps.
// It first commits to the save by reporting it Saving on the VMI status, which
// the status update of this sync persists with the resource version the
// armed volume was seen with. A cancel of the VM controller unplugs the volume
// in a patch conditional on the resource version, so either the commit or the
// cancel fails. Only a committed save is then handed to virt-launcher, which
// saves in the background and never retries a failed save.
func (c *VirtualMachineController) saveState(vmi *v1.VirtualMachineInstance) error {
	path, fileName, ok := saveTarget(vmi)
	if !ok {
		return nil
	}
	status := vmi.Status.SavedState
	if status == nil || !hibernation.IsSavePhase(status.Phase) || status.FileName != fileName {
		// The status of a VMI resumed from a saved state reports its restore,
		// and a VMI whose earlier hibernation failed reports that save:
		// neither is the save of this volume.
		now := metav1.Now()
		vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
			Phase:          v1.SavedStateSaving,
			FileName:       fileName,
			StartTimestamp: &now,
		}
		c.logger.Object(vmi).Infof("Committing to save the state of the VirtualMachineInstance to %s", path)
		return nil
	}
	if status.Phase != v1.SavedStateSaving || c.saveAccepted(vmi, fileName) {
		return nil
	}
	client, err := c.launcherClients.GetVerifiedLauncherClient(vmi)
	if err == nil {
		c.logger.Object(vmi).Infof("Saving the state of the VirtualMachineInstance to %s", path)
		if err = client.SaveVirtualMachine(vmi, path); err == nil {
			return nil
		}
	}
	if cmdclient.IsUnimplemented(err) {
		failSave(status, "virt-launcher does not support saving the state, it predates hibernation")
		return nil
	}
	if status.StartTimestamp != nil && time.Since(status.StartTimestamp.Time) > saveStartTimeout {
		failSave(status, fmt.Sprintf("virt-launcher did not start saving the state within %s: %v", saveStartTimeout, err))
		return nil
	}
	return fmt.Errorf("failed to save the state: %v", err)
}

func failSave(status *v1.VirtualMachineInstanceSavedStateStatus, message string) {
	now := metav1.Now()
	status.Phase = v1.SavedStateSaveFailed
	status.Message = message
	status.EndTimestamp = &now
}

// saveAccepted reports whether virt-launcher took the save to fileName, as
// its domain metadata shows: a save that runs is never reported failed
// because the RPC asking for it failed.
func (c *VirtualMachineController) saveAccepted(vmi *v1.VirtualMachineInstance, fileName string) bool {
	if c.BaseController == nil || c.domainStore == nil {
		return false
	}
	domain, exists, _, err := c.getDomainFromCache(controller.VirtualMachineInstanceKey(vmi))
	if err != nil || !exists || domain == nil {
		return false
	}
	metadata := domain.Spec.Metadata.KubeVirt.SavedState
	return metadata != nil && metadata.Operation == api.SavedStateOperationSave && metadata.FileName == fileName
}

// updateSavedStateFromDomain reports the save of the domain state, or its
// restore from a saved state, on the VMI status.
func updateSavedStateFromDomain(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	if domain == nil {
		return
	}
	metadata := domain.Spec.Metadata.KubeVirt.SavedState
	status := vmi.Status.SavedState

	switch {
	case status != nil && status.Phase == v1.SavedStateSaving:
		updateSaveStatus(status, domain, metadata)
	case metadata != nil && metadata.Operation == api.SavedStateOperationSave && metadata.FileName == armedFile(vmi) &&
		(status == nil || status.FileName != metadata.FileName):
		// The status of the save was lost: rebuild it from virt-launcher.
		vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
			Phase:          v1.SavedStateSaving,
			FileName:       metadata.FileName,
			StartTimestamp: metadata.StartTimestamp,
		}
		updateSaveStatus(vmi.Status.SavedState, domain, metadata)
	case vmi.Spec.SavedStateSource != nil && (status == nil || status.Phase == v1.SavedStateRestoring):
		if metadata == nil || metadata.Operation != api.SavedStateOperationRestore || !metadata.Completed ||
			metadata.FileName != vmi.Spec.SavedStateSource.FileName {
			return
		}
		vmi.Status.SavedState = restoreStatus(metadata)
	}
}

func updateSaveStatus(status *v1.VirtualMachineInstanceSavedStateStatus, domain *api.Domain, metadata *api.SavedStateMetadata) {
	if metadata != nil && (metadata.Operation != api.SavedStateOperationSave || metadata.FileName != status.FileName) {
		// The domain in the cache predates the save request: it reports the
		// restore the VMI resumed with, or an earlier save.
		return
	}
	if metadata != nil {
		if metadata.StartTimestamp != nil {
			status.StartTimestamp = metadata.StartTimestamp
		}
		if metadata.Completed && metadata.Failed {
			status.Phase = v1.SavedStateSaveFailed
			status.Message = metadata.FailureReason
			status.EndTimestamp = metadata.EndTimestamp
			return
		}
	}
	// libvirt stops a domain with this reason only once its state is saved.
	if domain.Status.Status == api.Shutoff && domain.Status.Reason == api.ReasonSaved {
		status.Phase = v1.SavedStateSaved
		if metadata != nil && metadata.EndTimestamp != nil {
			status.EndTimestamp = metadata.EndTimestamp
		} else {
			now := metav1.Now()
			status.EndTimestamp = &now
		}
	}
}

func restoreStatus(metadata *api.SavedStateMetadata) *v1.VirtualMachineInstanceSavedStateStatus {
	status := &v1.VirtualMachineInstanceSavedStateStatus{
		Phase:          v1.SavedStateRestored,
		FileName:       metadata.FileName,
		StartTimestamp: metadata.StartTimestamp,
		EndTimestamp:   metadata.EndTimestamp,
	}
	if metadata.Failed {
		status.Phase = v1.SavedStateRestoreFailed
		status.Message = metadata.FailureReason
	}
	return status
}

// restoreReady reports whether the domain of vmi can be started. A VMI that
// resumes from a saved state commits to the restore once its saved state
// source volume is mounted, by reporting it Restoring on the VMI status, and
// starts the domain only once that is persisted: the VM controller may stop
// a VMI that did not commit, keeping the saved state, while the guest of a
// VMI that committed may run.
func (c *VirtualMachineController) restoreReady(vmi *v1.VirtualMachineInstance) bool {
	source := vmi.Spec.SavedStateSource
	if source == nil || vmi.Status.SavedState != nil {
		return true
	}
	if !c.sourceMounted(vmi) {
		return false
	}
	now := metav1.Now()
	vmi.Status.SavedState = &v1.VirtualMachineInstanceSavedStateStatus{
		Phase:          v1.SavedStateRestoring,
		FileName:       source.FileName,
		StartTimestamp: &now,
	}
	c.logger.Object(vmi).Info("Committing to resume the VirtualMachineInstance from its saved state")
	return false
}

// sourceMounted reports whether the saved state source of vmi is mounted in
// its pod, as reported, or as mounted by this sync.
func (c *VirtualMachineController) sourceMounted(vmi *v1.VirtualMachineInstance) bool {
	if utilityvolume.Attached(vmi, hibernation.SourceVolumeName) {
		return true
	}
	if c.hotplugVolumeMounter == nil {
		return false
	}
	for _, volumeStatus := range vmi.Status.VolumeStatus {
		if volumeStatus.Name != hibernation.SourceVolumeName || volumeStatus.HotplugVolume == nil {
			continue
		}
		mounted, err := c.hotplugVolumeMounter.IsMounted(vmi, volumeStatus.Name, volumeStatus.HotplugVolume.AttachPodUID)
		return err == nil && mounted
	}
	return false
}

// restoreFailed reports whether resuming vmi from its saved state failed.
// virt-launcher never boots such a domain, so the VMI fails.
func restoreFailed(vmi *v1.VirtualMachineInstance, domain *api.Domain) bool {
	source := vmi.Spec.SavedStateSource
	if source == nil {
		return false
	}
	if domain != nil {
		switch domain.Status.Status {
		case api.Shutoff, api.Crashed, api.NoState:
		default:
			return false
		}
	}
	if status := vmi.Status.SavedState; status != nil && status.Phase == v1.SavedStateRestoreFailed {
		return true
	}
	if domain == nil {
		return false
	}
	metadata := domain.Spec.Metadata.KubeVirt.SavedState
	return metadata != nil && metadata.Operation == api.SavedStateOperationRestore && metadata.FileName == source.FileName &&
		metadata.Completed && metadata.Failed
}
