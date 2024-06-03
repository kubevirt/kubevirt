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
 * Copyright The KubeVirt Authors
 *
 */

package volumemigration

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	k8sv1 "k8s.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

const InvalidUpdateErrMsg = "The volume can only be reverted to the previous version during the update"

// invalidVols includes the invalid volumes for the volume migration
type invalidVols struct {
	hotplugged []string
	fs         []string
	shareable  []string
	luns       []string
}

func (vols *invalidVols) errorMessage() error {
	var s strings.Builder
	if len(vols.hotplugged) < 1 && len(vols.fs) < 1 &&
		len(vols.shareable) < 1 && len(vols.luns) < 1 {
		return nil
	}
	s.WriteString("invalid volumes to update with migration:")
	if len(vols.hotplugged) > 0 {
		s.WriteString(fmt.Sprintf(" hotplugged: %v", vols.hotplugged))
	}
	if len(vols.fs) > 0 {
		s.WriteString(fmt.Sprintf(" filesystems: %v", vols.fs))
	}
	if len(vols.shareable) > 0 {
		s.WriteString(fmt.Sprintf(" shareable: %v", vols.shareable))
	}
	if len(vols.luns) > 0 {
		s.WriteString(fmt.Sprintf(" luns: %v", vols.luns))
	}

	return fmt.Errorf(s.String())
}

// updatedVolumesMapping returns a mapping with the volume names and the old claims that have been updated in the VM
func updatedVolumesMapping(vmi *virtv1.VirtualMachineInstance, vm *virtv1.VirtualMachine) map[string]string {
	updateVols := make(map[string]string)
	vmVols := make(map[string]string)
	// New volumes
	for _, v := range vm.Spec.Template.Spec.Volumes {
		if name := storagetypes.PVCNameFromVirtVolume(&v); name != "" {
			vmVols[v.Name] = name
		}
	}
	// Old volumes
	for _, v := range vmi.Spec.Volumes {
		name := storagetypes.PVCNameFromVirtVolume(&v)
		if name == "" {
			continue
		}
		if claim, ok := vmVols[v.Name]; ok && name != claim {
			updateVols[v.Name] = claim
		}
	}
	return updateVols
}

// ValidateVolumes checks that the volumes can be updated with the migration
func ValidateVolumes(vmi *virtv1.VirtualMachineInstance, vm *virtv1.VirtualMachine) error {
	var invalidVols invalidVols
	if vmi == nil {
		return fmt.Errorf("cannot validate the migrated volumes for an empty VMI")
	}
	if vm == nil {
		return fmt.Errorf("cannot validate the migrated volumes for an empty VM")
	}
	updatedVols := updatedVolumesMapping(vmi, vm)
	valid := true
	disks := storagetypes.GetDisksByName(&vmi.Spec)
	filesystems := storagetypes.GetFilesystemsFromVolumes(vmi)
	for _, v := range vm.Spec.Template.Spec.Volumes {
		_, ok := updatedVols[v.Name]
		if !ok {
			continue
		}

		// Hotplugged volumes
		if storagetypes.IsHotplugVolume(&v) {
			invalidVols.hotplugged = append(invalidVols.hotplugged, v.Name)
			valid = false
			continue
		}
		// Filesystems
		if _, ok := filesystems[v.Name]; ok {
			invalidVols.fs = append(invalidVols.fs, v.Name)
			valid = false
			continue
		}

		d, ok := disks[v.Name]
		if !ok {
			continue
		}

		// Shareable disks
		if d.Shareable != nil && *d.Shareable {
			invalidVols.shareable = append(invalidVols.shareable, v.Name)
			valid = false
			continue
		}

		// LUN disks
		if d.DiskDevice.LUN != nil {
			invalidVols.luns = append(invalidVols.luns, v.Name)
			valid = false
			continue
		}
	}
	if !valid {
		return invalidVols.errorMessage()
	}

	return nil
}

// VolumeMigrationCancel cancels the volume migraton
func VolumeMigrationCancel(clientset kubecli.KubevirtClient, vmi *virtv1.VirtualMachineInstance, vm *virtv1.VirtualMachine) (bool, error) {
	if !IsVolumeMigrating(vmi) || !changeMigratedVolumes(vmi, vm) {
		return false, nil
	}
	// A volumem migration can be canceled only if the original set of volumes is restored
	if revertedToOldVolumes(vmi, vm) {
		vmiCopy, err := PatchVMIVolumes(clientset, vmi, vm)
		if err != nil {
			return true, err
		}
		return true, cancelVolumeMigration(clientset, vmiCopy)
	}

	return true, fmt.Errorf(InvalidUpdateErrMsg)
}

func changeMigratedVolumes(vmi *virtv1.VirtualMachineInstance, vm *virtv1.VirtualMachine) bool {
	updatedVols := updatedVolumesMapping(vmi, vm)
	for _, migVol := range vmi.Status.MigratedVolumes {
		if _, ok := updatedVols[migVol.VolumeName]; ok {
			return true
		}
	}
	return false
}

// revertedToOldVolumes checks that all migrated volumes have been reverted from destination to the source volume
func revertedToOldVolumes(vmi *virtv1.VirtualMachineInstance, vm *virtv1.VirtualMachine) bool {
	updatedVols := updatedVolumesMapping(vmi, vm)
	for _, migVol := range vmi.Status.MigratedVolumes {
		if migVol.SourcePVCInfo == nil {
			// something wrong with the source volume
			return false
		}
		claim, ok := updatedVols[migVol.VolumeName]
		if !ok || migVol.SourcePVCInfo.ClaimName != claim {
			return false
		}
		delete(updatedVols, migVol.VolumeName)
	}
	// updatedVols should only include the source volumes and not additional volumes.
	return len(updatedVols) == 0
}

func cancelVolumeMigration(clientset kubecli.KubevirtClient, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil {
		return fmt.Errorf("vmi is empty")
	}
	log.Log.V(2).Object(vmi).Infof("Cancel volume migration")
	vmiConditions := controller.NewVirtualMachineInstanceConditionManager()
	vmiCopy := vmi.DeepCopy()
	vmiConditions.UpdateCondition(vmiCopy, &virtv1.VirtualMachineInstanceCondition{
		Type:               virtv1.VirtualMachineInstanceVolumesChange,
		LastTransitionTime: metav1.Now(),
		Status:             k8sv1.ConditionFalse,
	})
	vmiCopy.Status.MigratedVolumes = nil
	if equality.Semantic.DeepEqual(vmiCopy.Status, vmi.Status) {
		return nil
	}
	log.Log.V(2).Object(vmi).Infof("Patch VMI %s status to cancel the volume migration", vmi.Name)
	p, err := patch.New(
		patch.WithTest("/status/conditions", vmi.Status.Conditions),
		patch.WithReplace("/status/conditions", vmiCopy.Status.Conditions),
		patch.WithTest("/status/migratedVolumes", vmi.Status.MigratedVolumes),
		patch.WithReplace("/status/migratedVolumes", vmiCopy.Status.MigratedVolumes),
	).GeneratePayload()
	if err != nil {
		return err
	}
	_, err = clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, p, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed updating vmi condition: %v", err)
	}
	return nil
}

// IsVolumeMigrating checks the VMI condition for volume migration
func IsVolumeMigrating(vmi *virtv1.VirtualMachineInstance) bool {
	return controller.NewVirtualMachineInstanceConditionManager().HasConditionWithStatus(vmi,
		virtv1.VirtualMachineInstanceVolumesChange, k8sv1.ConditionTrue)
}

// PatchVMIStatusWithMigratedVolumes patches the VMI status with the source and destination volume information during the volume migration
func PatchVMIStatusWithMigratedVolumes(clientset kubecli.KubevirtClient, pvcStore cache.Store, vmi *virtv1.VirtualMachineInstance, vm *virtv1.VirtualMachine) error {
	if len(vmi.Status.MigratedVolumes) > 0 {
		return nil
	}
	var migVolsInfo []virtv1.StorageMigratedVolumeInfo
	oldVols := make(map[string]string)
	for _, v := range vmi.Spec.Volumes {
		if pvcName := storagetypes.PVCNameFromVirtVolume(&v); pvcName != "" {
			oldVols[v.Name] = pvcName
		}
	}
	for _, v := range vm.Spec.Template.Spec.Volumes {
		claim := storagetypes.PVCNameFromVirtVolume(&v)
		if claim == "" {
			continue
		}
		oldClaim, ok := oldVols[v.Name]
		if !ok {
			continue
		}
		if oldClaim == claim {
			continue
		}
		oldPvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vmi.Namespace, oldClaim, pvcStore)
		if err != nil {
			return err
		}
		pvc, err := storagetypes.GetPersistentVolumeClaimFromCache(vmi.Namespace, claim, pvcStore)
		if err != nil {
			return err
		}
		var oldVolMode *k8sv1.PersistentVolumeMode
		var volMode *k8sv1.PersistentVolumeMode
		if oldPvc != nil && oldPvc.Spec.VolumeMode != nil {
			oldVolMode = oldPvc.Spec.VolumeMode
		}
		if pvc != nil && pvc.Spec.VolumeMode != nil {
			volMode = pvc.Spec.VolumeMode
		}
		migVolsInfo = append(migVolsInfo, virtv1.StorageMigratedVolumeInfo{
			VolumeName: v.Name,
			DestinationPVCInfo: &virtv1.PersistentVolumeClaimInfo{
				ClaimName:  claim,
				VolumeMode: volMode,
			},
			SourcePVCInfo: &virtv1.PersistentVolumeClaimInfo{
				ClaimName:  oldClaim,
				VolumeMode: oldVolMode,
			},
		})
	}
	if equality.Semantic.DeepEqual(migVolsInfo, vmi.Status.MigratedVolumes) {
		return nil
	}
	patch, err := patch.New(
		patch.WithTest("/status/migratedVolumes", vmi.Status.MigratedVolumes),
		patch.WithReplace("/status/migratedVolumes", migVolsInfo),
	).GeneratePayload()
	if err != nil {
		return err
	}
	vmi, err = clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	return err
}

// PatchVMIVolumes replaces the VMI volumes with the migrated volumes
func PatchVMIVolumes(clientset kubecli.KubevirtClient, vmi *virtv1.VirtualMachineInstance, vm *virtv1.VirtualMachine) (*virtv1.VirtualMachineInstance, error) {
	log.Log.V(2).Object(vmi).Infof("Patch VMI volumes")
	migVols := make(map[string]bool)
	vmiCopy := vmi.DeepCopy()
	if len(vmi.Status.MigratedVolumes) == 0 {
		return vmiCopy, nil
	}
	vmVols := storagetypes.GetVolumesByName(&vm.Spec.Template.Spec)
	for _, migVol := range vmi.Status.MigratedVolumes {
		migVols[migVol.VolumeName] = true
	}
	for i, v := range vmi.Spec.Volumes {
		if _, ok := migVols[v.Name]; ok {
			if vol, ok := vmVols[v.Name]; ok {
				vmiCopy.Spec.Volumes[i] = *vol
			}
		}
	}
	if equality.Semantic.DeepEqual(vmi.Spec.Volumes, vmiCopy.Spec.Volumes) {
		return vmiCopy, nil
	}
	patch, err := patch.New(
		patch.WithTest("/spec/volumes", vmi.Spec.Volumes),
		patch.WithReplace("/spec/volumes", vmiCopy.Spec.Volumes),
	).GeneratePayload()
	if err != nil {
		return nil, err
	}
	return clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
}

// CanVolumesUpdateMigration checks if the VMI can be update with the volume migration. For example, for certain VMs, the migration is not allowed for
// other reasons then the storage
func CanVolumesUpdateMigration(vmi *virtv1.VirtualMachineInstance) bool {
	if vmi == nil {
		return false
	}
	if len(vmi.Status.MigratedVolumes) == 0 {
		return false
	}
	// Check if there are other reasons rather than the DisksNotLiveMigratable
	for _, cond := range vmi.Status.Conditions {
		if cond.Type == virtv1.VirtualMachineInstanceIsMigratable &&
			cond.Status == k8sv1.ConditionFalse &&
			cond.Reason != virtv1.VirtualMachineInstanceReasonDisksNotMigratable {
			log.Log.Object(vmi).Errorf("cannot migrate the volumes as the VMI isn't migratable: %s", cond.Reason)
			return false
		}
	}
	// Check that all RWO volumes will be copied
	volMigMap := make(map[string]bool)
	for _, v := range vmi.Status.MigratedVolumes {
		volMigMap[v.VolumeName] = true
	}
	for _, v := range vmi.Status.VolumeStatus {
		if v.PersistentVolumeClaimInfo == nil {
			continue
		}
		_, ok := volMigMap[v.Name]
		if storagetypes.IsReadWriteOnceAccessMode(v.PersistentVolumeClaimInfo.AccessModes) && !ok {
			log.Log.Object(vmi).Errorf("cannot migrate the VM. The volume %s is RWO and not included in the migration volumes", v.Name)
			return false
		}
	}
	return true
}
