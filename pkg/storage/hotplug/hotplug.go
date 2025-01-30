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

package hotplug

import (
	"context"

	"k8s.io/apimachinery/pkg/api/equality"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

func HandleDeclarativeVolumes(client kubecli.KubevirtClient, vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	// TEMP as to not conflict with subresource
	if len(vm.Status.VolumeRequests) > 0 {
		return nil
	}

	if vm.Spec.UpdateVolumesStrategy != nil && *vm.Spec.UpdateVolumesStrategy == virtv1.UpdateVolumesStrategyMigration {
		// Are there some cases we can proceed?
		return nil
	}

	if err := patchHotplugVolumes(client, vm, vmi); err != nil {
		log.Log.Object(vm).Errorf("failed to update hotplug volumes for vmi:%v", err)
		return err
	}

	return nil
}

func patchHotplugVolumes(client kubecli.KubevirtClient, vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	if vmi == nil || !vmi.IsRunning() {
		return nil
	}

	newVmiVolumes := append(filterHotplugVMIVolumes(vm, vmi), getNewHotplugVMVolumes(vm, vmi)...)
	newVmiDisks := append(filterHotplugVMIDisks(vm, vmi, newVmiVolumes), getNewHotplugVMDisks(vm, vmi, newVmiVolumes)...)

	if equality.Semantic.DeepEqual(vmi.Spec.Volumes, newVmiVolumes) &&
		equality.Semantic.DeepEqual(vmi.Spec.Domain.Devices.Disks, newVmiDisks) {
		log.Log.Object(vm).V(3).Info("No hotplug volumes to patch")
		return nil
	}

	patchSet := patch.New(
		patch.WithTest("/spec/volumes", vmi.Spec.Volumes),
		patch.WithTest("/spec/domain/devices/disks", vmi.Spec.Domain.Devices.Disks),
	)

	if len(vmi.Spec.Volumes) > 0 {
		patchSet.AddOption(patch.WithReplace("/spec/volumes", newVmiVolumes))
	} else {
		patchSet.AddOption(patch.WithAdd("/spec/volumes", newVmiVolumes))
	}

	if len(vmi.Spec.Domain.Devices.Disks) > 0 {
		patchSet.AddOption(patch.WithReplace("/spec/domain/devices/disks", newVmiDisks))
	} else {
		patchSet.AddOption(patch.WithAdd("/spec/domain/devices/disks", newVmiDisks))
	}

	patchBytes, err := patchSet.GeneratePayload()
	if err != nil {
		return err
	}

	_, err = client.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, v1.PatchOptions{})
	if err != nil {
		return err
	}

	return nil
}

func filterHotplugVMIVolumes(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) []virtv1.Volume {
	var volumes []virtv1.Volume
	vmVolumesByName := storagetypes.GetVolumesByName(&vm.Spec.Template.Spec)

	// remove any volumes missing/changed in the VM spec
	for _, vmiVolume := range vmi.Spec.Volumes {
		if storagetypes.IsDeclarativeHotplugVolume(&vmiVolume) {
			vmVolume, exists := vmVolumesByName[vmiVolume.Name]
			if !exists {
				// volume not in VM spec, remove it
				log.Log.Object(vm).Infof("Removing hotplug volume %s from VMI, no longer in VM", vmiVolume.Name)
				continue
			}

			// volume changed in VM spec - remove it to be re-added with new values later
			if storagetypes.IsDeclarativeHotplugVolume(vmVolume) && !equality.Semantic.DeepEqual(vmVolume, &vmiVolume) {
				log.Log.Object(vm).Infof("Removing hotplug volume %s from VMI, volume changed", vmiVolume.Name)
				continue
			}
		}

		volumes = append(volumes, *vmiVolume.DeepCopy())
	}

	return volumes
}

func getNewHotplugVMVolumes(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) []virtv1.Volume {
	var volumes []virtv1.Volume
	vmiVolumesByName := storagetypes.GetVolumesByName(&vmi.Spec)

	var volumesWithStatus = make(map[string]struct{})
	for _, vs := range vmi.Status.VolumeStatus {
		volumesWithStatus[vs.Name] = struct{}{}
	}

	for _, vmVolume := range vm.Spec.Template.Spec.Volumes {
		if storagetypes.IsDeclarativeHotplugVolume(&vmVolume) {
			_, vmiVolumeExists := vmiVolumesByName[vmVolume.Name]

			// vmi will report status on volume after removed from spec
			// if in process of hot unplugging
			_, vmiVolumeHasStatus := volumesWithStatus[vmVolume.Name]

			if !vmiVolumeExists && !vmiVolumeHasStatus {
				log.Log.Object(vm).Infof("Adding hotplug volume %s to VMI", vmVolume.Name)
				volumes = append(volumes, *vmVolume.DeepCopy())
			}
		}
	}

	return volumes
}

func volumesByName(volumes []virtv1.Volume) map[string]*virtv1.Volume {
	volumeMap := make(map[string]*virtv1.Volume)
	for _, v := range volumes {
		volumeMap[v.Name] = v.DeepCopy()
	}
	return volumeMap
}

func filterHotplugVMIDisks(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, vmiNewVolumes []virtv1.Volume) []virtv1.Disk {
	var disks []virtv1.Disk
	vmiNewVolumesByName := volumesByName(vmiNewVolumes)
	vmDisksByName := storagetypes.GetDisksByName(&vm.Spec.Template.Spec)

	for _, vmiDisk := range vmi.Spec.Domain.Devices.Disks {
		_, vmDiskExists := vmDisksByName[vmiDisk.Name]
		_, vmiVolumeExists := vmiNewVolumesByName[vmiDisk.Name]

		if !vmDiskExists || !vmiVolumeExists {
			continue
		}

		disks = append(disks, *vmiDisk.DeepCopy())
	}

	return disks
}

func getNewHotplugVMDisks(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance, vmiNewVolumes []virtv1.Volume) []virtv1.Disk {
	var disks []virtv1.Disk
	vmiNewVolumesByName := volumesByName(vmiNewVolumes)
	vmiDisksByName := storagetypes.GetDisksByName(&vmi.Spec)

	for _, vmDisk := range vm.Spec.Template.Spec.Domain.Devices.Disks {
		vmVolume, vmVolumeExists := vmiNewVolumesByName[vmDisk.Name]
		_, vmiDiskExists := vmiDisksByName[vmDisk.Name]

		if vmVolumeExists && storagetypes.IsDeclarativeHotplugVolume(vmVolume) && !vmiDiskExists {
			log.Log.Object(vm).Infof("Adding hotplug disk %s to VMI", vmDisk.Name)
			disks = append(disks, *vmDisk.DeepCopy())
		}
	}

	return disks
}
