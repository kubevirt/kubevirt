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

package snapshotoverlay

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	hotplugdisk "kubevirt.io/kubevirt/pkg/storage/hotplug-disk"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// ScratchDir returns where the scratch volume of the VMI's External snapshot is
// mounted. The volume has to be attached to the VMI: the launcher is handed a
// path, and an unknown name would let the caller pick an arbitrary directory.
func ScratchDir(vmi *v1.VirtualMachineInstance, volumeName string) (string, error) {
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return "", err
	}
	if reference == nil || volumeName != storagetypes.ScratchPVCName(reference.ContentUID) {
		return "", fmt.Errorf("snapshot overlay request has no matching transaction")
	}
	for _, utilityVolume := range vmi.Spec.UtilityVolumes {
		if utilityVolume.Name == volumeName {
			return hotplugdisk.GetVolumeMountDir(volumeName), nil
		}
	}

	return "", fmt.Errorf("%q is not a utility volume of this VirtualMachineInstance", volumeName)
}

// UpdateVMIStatus reports the launcher's overlay metadata as the
// OverlaySnapshotActive condition and snapshotOverlayPVCUID of the VMI.
func UpdateVMIStatus(vmi *v1.VirtualMachineInstance, domain *api.Domain, condManager *controller.VirtualMachineInstanceConditionManager) {
	if domain == nil {
		// the condition's absence means the overlays were committed, so a domain
		// that could not be read must not clear it
		return
	}

	// the launcher zeroes the metadata rather than dropping the element
	overlay := domain.Spec.Metadata.KubeVirt.SnapshotOverlay
	vmi.Status.SnapshotOverlayPVCUID = ""
	if overlay == nil || overlay.Phase == "" {
		condManager.RemoveCondition(vmi, v1.VirtualMachineInstanceOverlaySnapshotActive)
		return
	}
	expected, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err == nil && expected != nil && overlay.ContentUID == expected.ContentUID && overlay.ScratchPVCUID == expected.ScratchPVCUID {
		vmi.Status.SnapshotOverlayPVCUID = overlay.ScratchPVCUID
	}

	status := k8sv1.ConditionTrue
	if overlay.Phase == api.SnapshotOverlayCommitted || overlay.Phase == api.SnapshotOverlaySnapshotFailed {
		// the disks are back on their base images
		status = k8sv1.ConditionFalse
	}

	condManager.UpdateCondition(vmi, &v1.VirtualMachineInstanceCondition{
		Type:               v1.VirtualMachineInstanceOverlaySnapshotActive,
		Status:             status,
		Reason:             string(overlay.Phase),
		Message:            overlay.Message,
		LastTransitionTime: metav1.Now(),
	})
}

// OverlayPhase returns the reason of the OverlaySnapshotActive condition, which
// also decides its status, or "" when the VMI has none.
func OverlayPhase(vmi *v1.VirtualMachineInstance) string {
	condition := controller.NewVirtualMachineInstanceConditionManager().GetCondition(vmi, v1.VirtualMachineInstanceOverlaySnapshotActive)
	if condition == nil {
		return ""
	}
	return condition.Reason
}

// OverlaysSettled reports whether the VMI reports the overlays on the scratch
// PVC with this UID as committed, or as never taken.
func OverlaysSettled(vmi *v1.VirtualMachineInstance, scratchPVCUID types.UID) bool {
	phase := OverlayPhase(vmi)
	return scratchPVCUID != "" && vmi.Status.SnapshotOverlayPVCUID == scratchPVCUID &&
		(phase == v1.VirtualMachineInstanceReasonOverlayCommitted || phase == v1.VirtualMachineInstanceReasonOverlaySnapshotFailed)
}

// HasUncommittedOverlays reports whether the VM is blocked by overlays a stopped
// launcher left uncommitted.
func HasUncommittedOverlays(vm *v1.VirtualMachine) bool {
	condition := controller.NewVirtualMachineConditionManager().GetCondition(vm, v1.VirtualMachineManualRecoveryRequired)
	return condition != nil && condition.Status == k8sv1.ConditionTrue && condition.Reason == storagetypes.UncommittedOverlaysReason
}

// SyncUncommittedOverlays sets ManualRecoveryRequired on a VM whose launcher
// stopped with overlays that may still hold writes. It reports cleanupPending
// while a scratch volume the launcher never used is still being removed.
func SyncUncommittedOverlays(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) (cleanupPending bool, err error) {
	if vm.DeletionTimestamp != nil || !storagetypes.HasSnapshotOverlayState(vmi) {
		return false, nil
	}
	launcherGone := vmi.IsFinal() || (vmi.IsWaitingForSync() && vmi.DeletionTimestamp != nil)
	// The VM controller drops its finalizer once the final VMI is recorded on the VM
	if !launcherGone || !controller.HasFinalizer(vmi, v1.VirtualMachineControllerFinalizer) {
		return false, nil
	}
	reference, err := storagetypes.GetOverlaySnapshotReference(vmi)
	if err != nil {
		return false, err
	}
	if reference == nil {
		return true, nil
	}
	if OverlaysSettled(vmi, reference.ScratchPVCUID) {
		return true, nil
	}
	controller.NewVirtualMachineConditionManager().UpdateCondition(vm, &v1.VirtualMachineCondition{
		Type:               v1.VirtualMachineManualRecoveryRequired,
		Status:             k8sv1.ConditionTrue,
		Reason:             storagetypes.UncommittedOverlaysReason,
		Message:            fmt.Sprintf("Uncommitted writes remain on scratch PVC `%s`. Startup is blocked. To keep the writes, commit the overlays manually, then annotate the VM with %s=%s. Otherwise restore a completed snapshot and accept losing writes newer than that snapshot.", storagetypes.ScratchPVCName(reference.ContentUID), storagetypes.OverlaysCommittedAnnotation, reference.ScratchPVCUID),
		LastTransitionTime: metav1.Now(),
	})
	return false, nil
}
