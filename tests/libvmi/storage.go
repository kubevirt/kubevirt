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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package libvmi

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libstorage"
)

const (
	waitDiskTemplateError         = "waiting on new disk to appear in template"
	waitVolumeTemplateError       = "waiting on new volume to appear in template"
	waitVolumeRequestProcessError = "waiting on all VolumeRequests to be processed"
)

// WithContainerImage specifies the name of the container image to be used.
func WithContainerImage(name string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		diskName := "disk0"
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, newDisk(diskName, v1.DiskBusVirtio))
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, newContainerVolume(diskName, name))
	}
}

func addDisk(vmi *v1.VirtualMachineInstance, disk v1.Disk) {
	if !diskExists(vmi, disk) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)
	}
}

func addVolume(vmi *v1.VirtualMachineInstance, volume v1.Volume) {
	if !volumeExists(vmi, volume) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, volume)
	}
}

func getVolume(vmi *v1.VirtualMachineInstance, name string) *v1.Volume {
	for i := range vmi.Spec.Volumes {
		if vmi.Spec.Volumes[i].Name == name {
			return &vmi.Spec.Volumes[i]
		}
	}
	return nil
}

func diskExists(vmi *v1.VirtualMachineInstance, disk v1.Disk) bool {
	for _, d := range vmi.Spec.Domain.Devices.Disks {
		if d.Name == disk.Name {
			return true
		}
	}
	return false
}

func volumeExists(vmi *v1.VirtualMachineInstance, volume v1.Volume) bool {
	for _, v := range vmi.Spec.Volumes {
		if v.Name == volume.Name {
			return true
		}
	}
	return false
}

func newDisk(name string, bus v1.DiskBus) v1.Disk {
	return v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	}
}

func newVolume(name string) v1.Volume {
	return v1.Volume{Name: name}
}

func newContainerVolume(name, image string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image: image,
			},
		},
	}
}

func VerifyVolumeAndDiskVMAdded(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	log.Log.Infof("Checking %d volumes", len(volumeNames))
	Eventually(func() error {
		updatedVM, err := virtClient.VirtualMachine(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
		if err != nil {
			return err
		}

		if len(updatedVM.Status.VolumeRequests) > 0 {
			return fmt.Errorf(waitVolumeRequestProcessError)
		}

		foundVolume := 0
		foundDisk := 0

		for _, volume := range updatedVM.Spec.Template.Spec.Volumes {
			if _, ok := nameMap[volume.Name]; ok {
				foundVolume++
			}
		}
		for _, disk := range updatedVM.Spec.Template.Spec.Domain.Devices.Disks {
			if _, ok := nameMap[disk.Name]; ok {
				foundDisk++
			}
		}

		if foundDisk != len(volumeNames) {
			return fmt.Errorf(waitDiskTemplateError)
		}
		if foundVolume != len(volumeNames) {
			return fmt.Errorf(waitVolumeTemplateError)
		}

		return nil
	}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func VerifyVolumeAndDiskVMIAdded(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	Eventually(func() error {
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &metav1.GetOptions{})
		if err != nil {
			return err
		}

		foundVolume := 0
		foundDisk := 0

		for _, volume := range updatedVMI.Spec.Volumes {
			if _, ok := nameMap[volume.Name]; ok {
				foundVolume++
			}
		}
		for _, disk := range updatedVMI.Spec.Domain.Devices.Disks {
			if _, ok := nameMap[disk.Name]; ok {
				foundDisk++
			}
		}

		if foundDisk != len(volumeNames) {
			return fmt.Errorf(waitDiskTemplateError)
		}
		if foundVolume != len(volumeNames) {
			return fmt.Errorf(waitVolumeTemplateError)
		}

		return nil
	}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func AddVolumeAndVerify(virtClient kubecli.KubevirtClient, storageClass string, vm *v1.VirtualMachine, addVMIOnly bool) string {
	dv := libstorage.NewRandomBlankDataVolume(vm.Namespace, storageClass, "64Mi", k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeFilesystem)
	_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(dv.Namespace).Create(context.Background(), dv, metav1.CreateOptions{})
	Expect(err).To(BeNil())
	Eventually(ThisDV(dv), 240).Should(HaveSucceeded())
	volumeSource := &v1.HotplugVolumeSource{
		DataVolume: &v1.DataVolumeSource{
			Name: dv.Name,
		},
	}
	addVolumeName := "test-volume-" + rand.String(12)
	addVolumeOptions := &v1.AddVolumeOptions{
		Name: addVolumeName,
		Disk: &v1.Disk{
			DiskDevice: v1.DiskDevice{
				Disk: &v1.DiskTarget{
					Bus: v1.DiskBusSCSI,
				},
			},
			Serial: addVolumeName,
		},
		VolumeSource: volumeSource,
	}

	if addVMIOnly {
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(vm.Namespace).AddVolume(vm.Name, addVolumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
	} else {
		Eventually(func() error {
			return virtClient.VirtualMachine(vm.Namespace).AddVolume(vm.Name, addVolumeOptions)
		}, 3*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
		VerifyVolumeAndDiskVMAdded(virtClient, vm, addVolumeName)
	}

	vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(vm.Name, &metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	VerifyVolumeAndDiskVMIAdded(virtClient, vmi, addVolumeName)

	return addVolumeName
}
