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
	"errors"
	"fmt"
	"sort"

	"github.com/openshift/library-go/pkg/build/naming"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	validation "k8s.io/apimachinery/pkg/util/validation"
	v1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"

	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
)

type VolumeOption int

const (
	// Default option, just includes regular volumes found in the VM/VMI spec
	WithRegularVolumes VolumeOption = iota
	// Includes backend storage PVC
	WithBackendVolume
	// Includes all volumes
	WithAllVolumes
)

var ErrNoBackendPVC = fmt.Errorf("no backend PVC when there should be one")

// GetVolumes returns all volumes of the passed object, empty if it's an unsupported object
func GetVolumes(obj interface{}, client kubecli.KubevirtClient, opts ...VolumeOption) ([]v1.Volume, error) {
	switch obj := obj.(type) {
	case *v1.VirtualMachine:
		return GetVirtualMachineVolumes(obj, client, opts...)
	case *snapshotv1.VirtualMachine:
		return GetSnapshotVirtualMachineVolumes(obj, client, opts...)
	case *v1.VirtualMachineInstance:
		return GetVirtualMachineInstanceVolumes(obj, opts...)
	default:
		return []v1.Volume{}, fmt.Errorf("unsupported object type: %T", obj)
	}
}

// GetVirtualMachineVolumes returns all volumes of a VM except the special ones based on volume options
func GetVirtualMachineVolumes(vm *v1.VirtualMachine, client kubecli.KubevirtClient, opts ...VolumeOption) ([]v1.Volume, error) {
	return getVolumes(vm, vm.Spec.Template.Spec, client, opts...)
}

// GetSnapshotVirtualMachineVolumes returns all volumes of a Snapshot VM except the special ones based on volume options
func GetSnapshotVirtualMachineVolumes(vm *snapshotv1.VirtualMachine, client kubecli.KubevirtClient, opts ...VolumeOption) ([]v1.Volume, error) {
	return getVolumes(vm, vm.Spec.Template.Spec, client, opts...)
}

// GetVirtualMachineInstanceVolumes returns all volumes of a VMI except the special ones based on volume options
func GetVirtualMachineInstanceVolumes(vmi *v1.VirtualMachineInstance, opts ...VolumeOption) ([]v1.Volume, error) {
	return getVolumes(vmi, vmi.Spec, nil, opts...)
}

func getVolumes(obj metav1.Object, vmiSpec v1.VirtualMachineInstanceSpec, client kubecli.KubevirtClient, opts ...VolumeOption) ([]v1.Volume, error) {
	var enumeratedVolumes []v1.Volume

	if needsRegularVolumes(vmiSpec, opts) {
		for _, volume := range vmiSpec.Volumes {
			enumeratedVolumes = append(enumeratedVolumes, volume)
		}
	}

	if needsBackendPVC(vmiSpec, opts) {
		backendVolumeName, err := getBackendPVCName(obj, client)
		if err != nil {
			return enumeratedVolumes, err
		}
		if backendVolumeName != "" {
			enumeratedVolumes = append(enumeratedVolumes, *createBackendPVCVolume(backendVolumeName, obj.GetName()))
		}
	}

	return enumeratedVolumes, nil
}

func getBackendPVCName(obj metav1.Object, client kubecli.KubevirtClient) (string, error) {
	switch obj := obj.(type) {
	case *v1.VirtualMachineInstance:
		return backendstorage.CurrentPVCName(obj), nil
	default:
		// TODO: This could be way more simpler if the backend PVC name was accessible from the VM spec/status.
		// Refactor this once the backend PVC is more accessible.
		if client == nil {
			return "", fmt.Errorf("no client provided")
		}
		pvcs, err := client.CoreV1().PersistentVolumeClaims(obj.GetNamespace()).List(context.Background(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", backendstorage.PVCPrefix, obj.GetName()),
		})
		if err != nil {
			return "", err
		}
		switch len(pvcs.Items) {
		case 1:
			return pvcs.Items[0].Name, nil
		case 0:
			return "", ErrNoBackendPVC
		default:
			pvc, err := getNewestNonTerminatingPVC(pvcs.Items)
			if err != nil {
				return "", fmt.Errorf("no non-terminating PVC found")
			}
			return pvc.Name, nil
		}
	}
}

func needsBackendPVC(vmiSpec v1.VirtualMachineInstanceSpec, opts []VolumeOption) bool {
	for _, opt := range opts {
		if opt == WithBackendVolume || opt == WithAllVolumes {
			return backendstorage.IsBackendStorageNeededForVMI(&vmiSpec)
		}
	}
	return false
}

func needsRegularVolumes(vmiSpec v1.VirtualMachineInstanceSpec, opts []VolumeOption) bool {
	if len(opts) == 0 {
		return true
	}

	for _, opt := range opts {
		if opt == WithRegularVolumes || opt == WithAllVolumes {
			return true
		}
	}
	return false
}

func createBackendPVCVolume(pvcName, vmName string) *v1.Volume {
	return &v1.Volume{
		Name: BackendPVCVolumeName(vmName),
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

// BackendPVCVolumeName return the name of the volume that will be arbitrarily used to represent
// the backend PVC during volume enumeration.
func BackendPVCVolumeName(vmName string) string {
	return naming.GetName(backendstorage.PVCPrefix, vmName, validation.DNS1035LabelMaxLength)
}

// Helper function to select the newest non-terminating PVC
func getNewestNonTerminatingPVC(pvcs []k8sv1.PersistentVolumeClaim) (*k8sv1.PersistentVolumeClaim, error) {
	nonTerminatingPVCs := []k8sv1.PersistentVolumeClaim{}

	for _, pvc := range pvcs {
		if pvc.ObjectMeta.DeletionTimestamp == nil {
			nonTerminatingPVCs = append(nonTerminatingPVCs, pvc)
		}
	}

	if len(nonTerminatingPVCs) == 0 {
		return nil, fmt.Errorf("no non-terminating PVCs found")
	}

	sort.Slice(nonTerminatingPVCs, func(i, j int) bool {
		return nonTerminatingPVCs[i].CreationTimestamp.After(nonTerminatingPVCs[j].CreationTimestamp.Time)
	})

	return &nonTerminatingPVCs[0], nil
}

func IsErrNoBackendPVC(err error) bool {
	return errors.Is(err, ErrNoBackendPVC)
}
