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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package backendstorage

import (
	"context"
	"fmt"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const (
	PVCPrefix = "persistent-state-for-"
	PVCSize   = "10Mi"
)

func PVCForVMI(vmi *corev1.VirtualMachineInstance) string {
	return PVCPrefix + vmi.Name
}

func HasPersistentTPMDevice(vmiSpec *corev1.VirtualMachineInstanceSpec) bool {
	if vmiSpec.Domain.Devices.TPM != nil &&
		vmiSpec.Domain.Devices.TPM.Persistent != nil &&
		*vmiSpec.Domain.Devices.TPM.Persistent {
		return true
	}

	return false
}

func isBackendStorageNeededForVMI(vmi *corev1.VirtualMachineInstance) bool {
	return HasPersistentTPMDevice(&vmi.Spec)
}

func IsBackendStorageNeededForVM(vm *corev1.VirtualMachine) bool {
	if vm.Spec.Template == nil {
		return false
	}
	return HasPersistentTPMDevice(&vm.Spec.Template.Spec)
}

func CreateIfNeeded(vmi *corev1.VirtualMachineInstance, clusterConfig *virtconfig.ClusterConfig, client kubecli.KubevirtClient) error {
	if !isBackendStorageNeededForVMI(vmi) {
		return nil
	}

	_, err := client.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), PVCForVMI(vmi), metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !errors.IsNotFound(err) {
		return err
	}

	modeFile := v1.PersistentVolumeFilesystem
	storageClass := clusterConfig.GetVMStateStorageClass()
	if storageClass == "" {
		return fmt.Errorf("backend VM storage requires a backend storage class defined in the custom resource")
	}
	ownerReferences := vmi.OwnerReferences
	if len(vmi.OwnerReferences) == 0 {
		// If the VMI has no owner, then it did not originate from a VM.
		// In that case, we tie the PVC to the VMI, rendering it quite useless since it wont actually persist.
		// The alternative is to remove this `if` block, allowing the PVC to persist after the VMI is deleted.
		// However, that would pose security and littering concerns.
		ownerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(vmi, corev1.VirtualMachineInstanceGroupVersionKind),
		}
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            PVCForVMI(vmi),
			OwnerReferences: ownerReferences,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteMany},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: resource.MustParse(PVCSize)},
			},
			StorageClassName: &storageClass,
			VolumeMode:       &modeFile,
		},
	}

	_, err = client.CoreV1().PersistentVolumeClaims(vmi.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		return nil
	}

	return err
}
