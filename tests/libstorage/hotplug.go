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

package libstorage

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func AttachmentPodName(vmi *v1.VirtualMachineInstance) string {
	for _, volume := range vmi.Status.VolumeStatus {
		if volume.HotplugVolume != nil {
			return volume.HotplugVolume.AttachPodName
		}
	}
	return ""
}

func VerifyVolumeStatus(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, expectedPhase v1.VolumePhase, expectedCache v1.DriverCache, expectTarget bool, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	Eventually(func() error {
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		foundVolume := 0
		for _, volumeStatus := range updatedVMI.Status.VolumeStatus {
			if _, ok := nameMap[volumeStatus.Name]; ok && volumeStatus.HotplugVolume != nil {
				if expectTarget && volumeStatus.Target == "" {
					continue
				}

				if volumeStatus.Phase == expectedPhase {
					foundVolume++
				}
			}
		}

		if foundVolume != len(volumeNames) {
			return fmt.Errorf("waiting on volume statuses for volumes to be in phase %s (found %d, expected %d)", expectedPhase, foundVolume, len(volumeNames))
		}

		// Verify disk cache mode in spec if specified
		if expectedCache != "" {
			for _, disk := range updatedVMI.Spec.Domain.Devices.Disks {
				if _, ok := nameMap[disk.Name]; ok && disk.Cache != expectedCache {
					return fmt.Errorf("expected disk cache mode is %s, but %s in actual", expectedCache, string(disk.Cache))
				}
			}
		}

		return nil
	}, 360*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
}

func GetVolumeTargetPaths(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, requireHotplug bool, volumeNames ...string) []string {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	res := make([]string, 0)
	updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	for _, volumeStatus := range updatedVMI.Status.VolumeStatus {
		if _, ok := nameMap[volumeStatus.Name]; ok && volumeStatus.HotplugVolume != nil {
			Expect(volumeStatus.Target).ToNot(BeEmpty())
			res = append(res, fmt.Sprintf("/dev/%s", volumeStatus.Target))
		}
	}
	return res
}

func VerifyVolumeAndDiskInVMISpec(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, volumeNames ...string) {
	nameMap := make(map[string]bool)
	for _, volumeName := range volumeNames {
		nameMap[volumeName] = true
	}
	Eventually(func() error {
		updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
			return fmt.Errorf("waiting on VMI disks to be added (found %d, expected %d)", foundDisk, len(volumeNames))
		}
		if foundVolume != len(volumeNames) {
			return fmt.Errorf("waiting on VMI volumes to be added (found %d, expected %d)", foundVolume, len(volumeNames))
		}

		return nil
	}, 90*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
}

func WaitForHotplugToComplete(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, volumeName, dvName string, added bool) {
	Eventually(func() error {
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		found := false
		for _, volume := range vmi.Spec.Volumes {
			if volume.Name == volumeName && volume.DataVolume != nil && volume.DataVolume.Name == dvName {
				found = true
				break
			}
		}
		if found != added {
			return fmt.Errorf("volume %s in VM %s not in expected state %t, spec not updated", volumeName, vm.Name, added)
		}
		found = false
		for _, vs := range vmi.Status.VolumeStatus {
			if vs.Name == volumeName {
				if added && vs.Phase == v1.VolumeReady {
					return nil
				}
				if !added {
					found = true
					break
				}
			}
		}
		if !added && !found {
			return nil
		}
		return fmt.Errorf("volume %s in VM %s not in expected state %t, status not updated", volumeName, vm.Name, added)
	}, 240*time.Second, 2*time.Second).Should(Succeed())
}

func AddHotplugDiskAndVolume(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, volumeName, dvName string) *v1.VirtualMachine {
	vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	vmCpy := vm.DeepCopy()
	disks := vmCpy.Spec.Template.Spec.Domain.Devices.Disks
	volumes := vmCpy.Spec.Template.Spec.Volumes
	disks = append(disks, v1.Disk{
		Name: volumeName,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: v1.DiskBusSCSI,
			},
		},
	})
	volumes = append(volumes, v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name:         dvName,
				Hotpluggable: true,
			},
		},
	})

	patchObj := patch.New()
	patchObj.AddOption(patch.WithReplace("/spec/template/spec/domain/devices/disks", disks))
	patchObj.AddOption(patch.WithReplace("/spec/template/spec/volumes", volumes))
	patchBytes, err := patchObj.GeneratePayload()
	Expect(err).ToNot(HaveOccurred())

	vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())

	return vm
}

func RemoveHotplugDiskAndVolume(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine, volumeName string) *v1.VirtualMachine {
	vm, err := virtClient.VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	var disks []v1.Disk
	for _, disk := range vm.Spec.Template.Spec.Domain.Devices.Disks {
		if disk.Name != volumeName {
			disks = append(disks, disk)
		}
	}
	var volumes []v1.Volume
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.Name != volumeName {
			volumes = append(volumes, volume)
		}
	}

	patchObj := patch.New()
	if len(disks) == 0 {
		patchObj.AddOption(patch.WithRemove("/spec/template/spec/domain/devices/disks"))
	} else {
		patchObj.AddOption(patch.WithAdd("/spec/template/spec/domain/devices/disks", disks))
	}
	if len(volumes) == 0 {
		patchObj.AddOption(patch.WithRemove("/spec/template/spec/volumes"))
	} else {
		patchObj.AddOption(patch.WithAdd("/spec/template/spec/volumes", volumes))
	}

	patchBytes, err := patchObj.GeneratePayload()
	Expect(err).ToNot(HaveOccurred())
	vm, err = virtClient.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	Expect(err).ToNot(HaveOccurred())
	return vm
}
