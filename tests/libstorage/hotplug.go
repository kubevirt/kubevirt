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

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
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
