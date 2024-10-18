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
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
)

type VolumeOption int

const (
	// Default option, just includes regular volumes found in the VM/VMI spec
	WithRegularVolumes VolumeOption = iota
	// Also adds the backend storage PVC
	WithBackendVolume
	// TODO: Add other options as needed
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

// needsBackendPVC checks if the backend PVC is needed based on the options passed
func needsBackendPVC(vmiSpec v1.VirtualMachineInstanceSpec, opts []VolumeOption) bool {
	for _, opt := range opts {
		if opt == WithBackendVolume {
			return backendstorage.IsBackendStorageNeededForVMI(&vmiSpec)
		}
	}
	return false
}

// GetVolumes returns all volumes of the passed object, empty if it's an unsupported object
func GetVolumes(obj interface{}, client kubecli.KubevirtClient, opts ...VolumeOption) ([]v1.Volume, error) {
	switch obj := obj.(type) {
	case *v1.VirtualMachine:
		return GetVirtualMachineVolumes(obj, client, opts...)
	case *snapshotv1.VirtualMachine:
		return GetSnapshotVirtualMachineVolumes(obj, client, opts...)
	case *v1.VirtualMachineInstance:
		return GetVirtualMachineInstanceVolumes(obj, opts...), nil
	default:
		return []v1.Volume{}, fmt.Errorf("unsupported object type: %T", obj)
	}
}

// GetVirtualMachineVolumes returns all volumes of a VM except the special ones based on volume options
func GetVirtualMachineVolumes(vm *v1.VirtualMachine, client kubecli.KubevirtClient, opts ...VolumeOption) ([]v1.Volume, error) {
	var err error
	vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: vm.Name}, Spec: vm.Spec.Template.Spec}
	if needsBackendPVC(vm.Spec.Template.Spec, opts) {
		if client == nil {
			return []v1.Volume{}, fmt.Errorf("no client provided")
		}
		vmi, err = client.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		if err != nil {
			return []v1.Volume{}, err
		}
	}
	return GetVirtualMachineInstanceVolumes(vmi, opts...), nil
}

// GetSnapshotVirtualMachineVolumes returns all volumes of a Snapshot VM except the special ones based on volume options
func GetSnapshotVirtualMachineVolumes(vm *snapshotv1.VirtualMachine, client kubecli.KubevirtClient, opts ...VolumeOption) ([]v1.Volume, error) {
	var err error
	vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: vm.Name}, Spec: vm.Spec.Template.Spec}
	if needsBackendPVC(vm.Spec.Template.Spec, opts) {
		if client == nil {
			return []v1.Volume{}, fmt.Errorf("no client provided")
		}
		vmi, err = client.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		if err != nil {
			return []v1.Volume{}, err
		}
	}
	return GetVirtualMachineInstanceVolumes(vmi, opts...), nil
}

// GetVirtualMachineInstanceVolumes returns all volumes of a VMI except the special ones based on volume options
func GetVirtualMachineInstanceVolumes(vmi *v1.VirtualMachineInstance, opts ...VolumeOption) []v1.Volume {
	var enumeratedVolumes []v1.Volume

	for _, volume := range vmi.Spec.Volumes {
		enumeratedVolumes = append(enumeratedVolumes, volume)
	}

	if needsBackendPVC(vmi.Spec, opts) {
		backendVolume := backendstorage.CurrentPVCName(vmi)
		if backendVolume != "" {
			enumeratedVolumes = append(enumeratedVolumes, *createPVCVolume(backendVolume))
		}
	}

	return enumeratedVolumes
}
