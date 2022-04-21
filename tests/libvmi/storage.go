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
	kvirtv1 "kubevirt.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
)

// WithContainerImage specifies the name of the container image to be used.
func WithContainerImage(name string) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		diskName := "disk0"
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, newDisk(diskName, v1.DiskBusVirtio))
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, newContainerVolume(diskName, name))
	}
}

func addDisk(vmi *kvirtv1.VirtualMachineInstance, disk kvirtv1.Disk) {
	if !diskExists(vmi, disk) {
		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, disk)
	}
}

func addVolume(vmi *kvirtv1.VirtualMachineInstance, volume kvirtv1.Volume) {
	if !volumeExists(vmi, volume) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, volume)
	}
}

func getVolume(vmi *kvirtv1.VirtualMachineInstance, name string) *kvirtv1.Volume {
	for i := range vmi.Spec.Volumes {
		if vmi.Spec.Volumes[i].Name == name {
			return &vmi.Spec.Volumes[i]
		}
	}
	return nil
}

func diskExists(vmi *kvirtv1.VirtualMachineInstance, disk kvirtv1.Disk) bool {
	for _, d := range vmi.Spec.Domain.Devices.Disks {
		if d.Name == disk.Name {
			return true
		}
	}
	return false
}

func volumeExists(vmi *kvirtv1.VirtualMachineInstance, volume kvirtv1.Volume) bool {
	for _, v := range vmi.Spec.Volumes {
		if v.Name == volume.Name {
			return true
		}
	}
	return false
}

func newDisk(name string, bus v1.DiskBus) kvirtv1.Disk {
	return kvirtv1.Disk{
		Name: name,
		DiskDevice: kvirtv1.DiskDevice{
			Disk: &kvirtv1.DiskTarget{
				Bus: bus,
			},
		},
	}
}

func newVolume(name string) kvirtv1.Volume {
	return kvirtv1.Volume{Name: name}
}

func newContainerVolume(name, image string) kvirtv1.Volume {
	return kvirtv1.Volume{
		Name: name,
		VolumeSource: kvirtv1.VolumeSource{
			ContainerDisk: &kvirtv1.ContainerDiskSource{
				Image: image,
			},
		},
	}
}
