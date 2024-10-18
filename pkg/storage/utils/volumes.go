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

package utils

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
)

const (
	FLAG_INCLUDE_ALL = 0
	FLAG_EXCLUDE_EFI = 1 << iota
	FLAG_EXCLUDE_TPM
	// TODO: Add other flags as needed (KERNEL_BOOT?)
)

func createPVCVolume(pvcName string) *v1.Volume {
	return &v1.Volume{
		Name: pvcName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
					ReadOnly:  false,
				},
			},
		},
	}
}

// GetVolumes returns all volumes of the passed object, nil if it's an unsupported object
func GetVolumes(obj interface{}, excludeFlags int) []v1.Volume {
	switch obj := obj.(type) {
	case *v1.VirtualMachine:
		return GetVirtualMachineVolumes(obj, excludeFlags)
	case *snapshotv1.VirtualMachine:
		return GetSnapshotVirtualMachineVolumes(obj, excludeFlags)
	case *v1.VirtualMachineInstance:
		return GetVirtualMachineInstanceVolumes(obj, excludeFlags)
	default:
		return []v1.Volume{}
	}
}

// GetVirtualMachineVolumes returns all volumes of a VM except the special ones based on the exclude flags
func GetVirtualMachineVolumes(vm *v1.VirtualMachine, excludeFlags int) []v1.Volume {
	return GetVirtualMachineInstanceVolumes(&v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: vm.Name}, Spec: vm.Spec.Template.Spec}, excludeFlags)
}

// GetSnapshotVirtualMachineVolumes returns all volumes of a Snapshot VM except the special ones based on the exclude flags
func GetSnapshotVirtualMachineVolumes(vm *snapshotv1.VirtualMachine, excludeFlags int) []v1.Volume {
	return GetVirtualMachineInstanceVolumes(&v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: vm.Name}, Spec: vm.Spec.Template.Spec}, excludeFlags)
}

// GetVirtualMachineInstanceVolumes returns all volumes of a VMI except the special ones based on the exclude flags
func GetVirtualMachineInstanceVolumes(vmi *v1.VirtualMachineInstance, excludeFlags int) []v1.Volume {
	var enumeratedVolumes []v1.Volume

	for _, volume := range vmi.Spec.Volumes {
		enumeratedVolumes = append(enumeratedVolumes, volume)
	}

	if (backendstorage.HasPersistentEFI(&vmi.Spec) && excludeFlags&FLAG_EXCLUDE_EFI == 0) ||
		(backendstorage.HasPersistentTPMDevice(&vmi.Spec) && excludeFlags&FLAG_EXCLUDE_TPM == 0) {
		backendVolume := backendstorage.PVCForVMI(vmi)
		if backendVolume != "" {
			enumeratedVolumes = append(enumeratedVolumes, *createPVCVolume(backendVolume))
		}
	}

	return enumeratedVolumes
}
