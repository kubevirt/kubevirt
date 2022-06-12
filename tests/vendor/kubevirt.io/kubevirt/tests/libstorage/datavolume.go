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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package libstorage

import (
	"context"

	"github.com/onsi/ginkgo/v2"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/resource"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v13 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/tests/util"
)

func NewRandomBlockDataVolumeWithRegistryImport(imageUrl, namespace string, accessMode v1.PersistentVolumeAccessMode) *v1beta1.DataVolume {
	sc, exists := GetRWOBlockStorageClass()
	if accessMode == v1.ReadWriteMany {
		sc, exists = GetRWXBlockStorageClass()
	}
	if !exists {
		ginkgo.Skip("Skip test when Block storage is not present")
	}
	return NewRandomDataVolumeWithRegistryImportInStorageClass(imageUrl, namespace, sc, accessMode, v1.PersistentVolumeBlock)
}

func NewRandomDataVolumeWithRegistryImport(imageUrl, namespace string, accessMode v1.PersistentVolumeAccessMode) *v1beta1.DataVolume {
	sc, exists := GetRWOFileSystemStorageClass()
	if accessMode == v1.ReadWriteMany {
		sc, exists = GetRWXFileSystemStorageClass()
	}
	if !exists {
		ginkgo.Skip("Skip test when Filesystem storage is not present")
	}
	return NewRandomDataVolumeWithRegistryImportInStorageClass(imageUrl, namespace, sc, accessMode, v1.PersistentVolumeFilesystem)
}

func newDataVolume(namespace, storageClass string, size string, accessMode v1.PersistentVolumeAccessMode, volumeMode v1.PersistentVolumeMode, dataVolumeSource v1beta1.DataVolumeSource) *v1beta1.DataVolume {
	name := "test-datavolume-" + rand.String(12)
	quantity, err := resource.ParseQuantity(size)
	util.PanicOnError(err)
	dataVolume := &v1beta1.DataVolume{
		ObjectMeta: v12.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.DataVolumeSpec{
			Source: &dataVolumeSource,
			PVC: &v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{accessMode},
				VolumeMode:  &volumeMode,
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"storage": quantity,
					},
				},
				StorageClassName: &storageClass,
			},
		},
	}

	dataVolume.TypeMeta = v12.TypeMeta{
		APIVersion: "cdi.kubevirt.io/v1beta1",
		Kind:       "DataVolume",
	}

	return dataVolume
}

func NewRandomDataVolumeWithRegistryImportInStorageClass(imageUrl, namespace, storageClass string, accessMode v1.PersistentVolumeAccessMode, volumeMode v1.PersistentVolumeMode) *v1beta1.DataVolume {
	size := "512Mi"
	dataVolumeSource := v1beta1.DataVolumeSource{
		Registry: &v1beta1.DataVolumeSourceRegistry{
			URL: &imageUrl,
		},
	}
	return newDataVolume(namespace, storageClass, size, accessMode, volumeMode, dataVolumeSource)
}

func NewRandomBlankDataVolume(namespace, storageClass, size string, accessMode v1.PersistentVolumeAccessMode, volumeMode v1.PersistentVolumeMode) *v1beta1.DataVolume {
	dataVolumeSource := v1beta1.DataVolumeSource{
		Blank: &v1beta1.DataVolumeBlankImage{},
	}
	return newDataVolume(namespace, storageClass, size, accessMode, volumeMode, dataVolumeSource)
}

func NewRandomDataVolumeWithPVCSource(sourceNamespace, sourceName, targetNamespace string, accessMode v1.PersistentVolumeAccessMode) *v1beta1.DataVolume {
	sc, exists := GetRWOFileSystemStorageClass()
	if accessMode == v1.ReadWriteMany {
		sc, exists = GetRWXFileSystemStorageClass()
	}
	if !exists {
		ginkgo.Skip("Skip test when Filesystem storage is not present")
	}
	return newRandomDataVolumeWithPVCSourceWithStorageClass(sourceNamespace, sourceName, targetNamespace, sc, "1Gi", accessMode)
}

func newRandomDataVolumeWithPVCSourceWithStorageClass(sourceNamespace, sourceName, targetNamespace, storageClass, size string, accessMode v1.PersistentVolumeAccessMode) *v1beta1.DataVolume {
	dataVolumeSource := v1beta1.DataVolumeSource{
		PVC: &v1beta1.DataVolumeSourcePVC{
			Namespace: sourceNamespace,
			Name:      sourceName,
		},
	}
	volumeMode := v1.PersistentVolumeFilesystem
	return newDataVolume(targetNamespace, storageClass, size, accessMode, volumeMode, dataVolumeSource)
}

func AddDataVolumeDisk(vmi *v13.VirtualMachineInstance, diskName, dataVolumeName string) *v13.VirtualMachineInstance {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v13.Disk{
		Name: diskName,
		DiskDevice: v13.DiskDevice{
			Disk: &v13.DiskTarget{
				Bus: v13.DiskBusVirtio,
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v13.Volume{
		Name: diskName,
		VolumeSource: v13.VolumeSource{
			DataVolume: &v13.DataVolumeSource{
				Name: dataVolumeName,
			},
		},
	})

	return vmi
}

func AddDataVolumeTemplate(vm *v13.VirtualMachine, dataVolume *v1beta1.DataVolume) {
	dvt := &v13.DataVolumeTemplateSpec{}

	dvt.Spec = *dataVolume.Spec.DeepCopy()
	dvt.ObjectMeta = *dataVolume.ObjectMeta.DeepCopy()

	vm.Spec.DataVolumeTemplates = append(vm.Spec.DataVolumeTemplates, *dvt)
}

func HasDataVolumeCRD() bool {
	virtClient, err := kubecli.GetKubevirtClient()
	util.PanicOnError(err)

	ext, err := clientset.NewForConfig(virtClient.Config())
	util.PanicOnError(err)

	_, err = ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "datavolumes.cdi.kubevirt.io", v12.GetOptions{})

	if err != nil {
		return false
	}
	return true
}

func HasCDI() bool {
	return HasDataVolumeCRD()
}
