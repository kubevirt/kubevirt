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

func HasPersistentTPMDevice(vmi *corev1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.TPM != nil &&
		vmi.Spec.Domain.Devices.TPM.Persistent != nil &&
		*vmi.Spec.Domain.Devices.TPM.Persistent {
		return true
	}

	return false
}

func isBackendStorageNeeded(vmi *corev1.VirtualMachineInstance) bool {
	return HasPersistentTPMDevice(vmi)
}

func CreateIfNeeded(vmi *corev1.VirtualMachineInstance, clusterConfig *virtconfig.ClusterConfig, client kubecli.KubevirtClient) (bool, error) {
	if !isBackendStorageNeeded(vmi) {
		return true, nil
	}

	_, err := client.CoreV1().PersistentVolumeClaims(vmi.Namespace).Get(context.Background(), PVCPrefix+vmi.Name, metav1.GetOptions{})
	if err == nil {
		return true, nil
	}
	if !errors.IsNotFound(err) {
		return false, err
	}

	modeFile := v1.PersistentVolumeFilesystem
	storageClass := clusterConfig.GetVMStateStorageClass()
	if storageClass == "" {
		return false, fmt.Errorf("backend VM storage requires a backend storage class defined in the custom resource")
	}
	pvc := &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name: PVCPrefix + vmi.Name,
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
	if err != nil {
		return false, err
	}

	return true, nil
}
