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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package migration

import (
	"context"
	"fmt"

	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	virtv1 "kubevirt.io/api/core/v1"
	virtstorage "kubevirt.io/api/storage/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

type VolumeMigrationUpdater interface {
	UpdateVMIWithMigrationVolumes(vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstance, error)
	UpdateVMWithMigrationVolumes(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
}

type volumeMigrationUpdater struct {
	clientset kubecli.KubevirtClient
}

func NewVolumeMigrationUpdater(clientset kubecli.KubevirtClient) VolumeMigrationUpdater {
	return &volumeMigrationUpdater{clientset: clientset}
}

func (c *volumeMigrationUpdater) UpdateVMIWithMigrationVolumes(vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachineInstance, error) {
	vmiCopy := vmi.DeepCopy()
	if err := replaceSourceVolswithDestinationVolVMI(vmiCopy); err != nil {
		return nil, err
	}
	if equality.Semantic.DeepEqual(vmi, vmiCopy) {
		return vmiCopy, nil
	}
	if _, err := c.clientset.VirtualMachineInstance(vmi.ObjectMeta.Namespace).Update(context.Background(), vmiCopy); err != nil {
		return nil, fmt.Errorf("failed updating migrated disks: %v", err)
	}
	return vmiCopy, nil
}

func (c *volumeMigrationUpdater) UpdateVMWithMigrationVolumes(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
	vmCopy := vm.DeepCopy()
	if err := replaceSourceVolswithDestinationVolVM(vmCopy, vmi); err != nil {
		return nil, err
	}
	if equality.Semantic.DeepEqual(vm, vmCopy) {
		return vmCopy, nil
	}
	if _, err := c.clientset.VirtualMachine(vm.ObjectMeta.Namespace).Update(context.Background(), vmCopy); err != nil {
		return nil, fmt.Errorf("failed updating migrated disks: %v", err)
	}
	return vmCopy, nil
}

func replaceSourceVolswithDestinationVolVMI(vmi *virtv1.VirtualMachineInstance) error {
	// Collect migrated volumes that need to be update
	replaceVol := make(map[string]string)
	for _, v := range vmi.Status.MigratedVolumes {
		if v.MigrationPhase == nil || v.SourcePVCInfo == nil || v.DestinationPVCInfo == nil {
			continue
		}
		if *v.MigrationPhase != virtstorage.VolumeMigrationPhaseSucceeded {
			continue
		}
		replaceVol[v.SourcePVCInfo.ClaimName] = v.DestinationPVCInfo.ClaimName
	}

	for i, v := range vmi.Spec.Volumes {
		claim := storagetypes.PVCNameFromVirtVolume(&v)
		if claim == "" {
			continue
		}

		if dest, ok := replaceVol[claim]; ok {
			switch {
			case v.VolumeSource.PersistentVolumeClaim != nil:
				vmi.Spec.Volumes[i].VolumeSource.PersistentVolumeClaim.ClaimName = dest
			case v.VolumeSource.DataVolume != nil:
				vmi.Spec.Volumes[i].VolumeSource.PersistentVolumeClaim = &virtv1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
						ClaimName: dest,
					},
				}
				vmi.Spec.Volumes[i].VolumeSource.DataVolume = nil
			}
			delete(replaceVol, claim)
		}
	}
	if len(replaceVol) != 0 {
		return fmt.Errorf("failed to replace the source volumes with the destination volumes in the VMI")
	}

	return nil
}

func deleteDVTemplateFromSpec(vm *virtv1.VirtualMachine, dv string) {
	for i, dvSpec := range vm.Spec.DataVolumeTemplates {
		if dvSpec.ObjectMeta.Name != dv {
			continue
		}
		length := len(vm.Spec.DataVolumeTemplates)
		switch {
		case length == 1:
			vm.Spec.DataVolumeTemplates = []virtv1.DataVolumeTemplateSpec{}
		case i == length-1:
			vm.Spec.DataVolumeTemplates = vm.Spec.DataVolumeTemplates[:i-1]
		default:
			vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates[:i],
				vm.Spec.DataVolumeTemplates[i+1:]...)
		}
	}
}

func replaceSourceVolswithDestinationVolVM(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	migrateVolMap := make(map[string]string)
	volVmi := make(map[string]bool)
	if vmi == nil {
		return nil
	}
	for _, v := range vmi.Status.MigratedVolumes {
		if v.MigrationPhase == nil || v.SourcePVCInfo == nil || v.DestinationPVCInfo == nil {
			continue
		}
		if *v.MigrationPhase != virtstorage.VolumeMigrationPhaseSucceeded {
			continue
		}
		migrateVolMap[v.SourcePVCInfo.ClaimName] = v.DestinationPVCInfo.ClaimName
	}

	for _, v := range vmi.Spec.Volumes {
		if name := storagetypes.PVCNameFromVirtVolume(&v); name != "" {
			volVmi[name] = true
		}
	}
	for k, v := range vm.Spec.Template.Spec.Volumes {
		if name := storagetypes.PVCNameFromVirtVolume(&v); name != "" {
			// The volume to update in the VM needs to be one of the migrate
			// volume AND already have been changed in the VMI spec
			repName, okMig := migrateVolMap[name]
			_, okVMI := volVmi[name]
			if okMig && okVMI {
				switch {
				case v.VolumeSource.PersistentVolumeClaim != nil:
					vm.Spec.Template.Spec.Volumes[k].VolumeSource.PersistentVolumeClaim.ClaimName = repName
				case v.VolumeSource.DataVolume != nil:
					vm.Spec.Template.Spec.Volumes[k].VolumeSource.PersistentVolumeClaim = &virtv1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: k8score.PersistentVolumeClaimVolumeSource{
							ClaimName: repName,
						},
					}
					vm.Spec.Template.Spec.Volumes[k].VolumeSource.DataVolume = nil
					deleteDVTemplateFromSpec(vm, name)
				}
				delete(migrateVolMap, name)
			}
		}
	}

	if len(migrateVolMap) != 0 {
		return fmt.Errorf("failed to replace the source volumes with the destination volumes in the VM")
	}

	return nil
}
