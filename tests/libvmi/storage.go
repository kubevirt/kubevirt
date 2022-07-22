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
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

// WithContainerImage specifies the name of the container image to be used.
func WithContainerImage(name string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		diskName := "disk0"
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio))
		addVolume(vmi, newContainerVolume(diskName, name))
	}
}

// WithPersistentVolumeClaim specifies the name of the PersistentVolumeClaim to be used.
func WithPersistentVolumeClaim(diskName, pvcName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio))
		addVolume(vmi, newPersistentVolumeClaimVolume(diskName, pvcName))
	}
}

// WithDataVolume specifies the name of the DataVolume to be used.
func WithDataVolume(diskName, pvcName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio))
		addVolume(vmi, newDataVolume(diskName, pvcName))
	}
}

// WithEmptyDisk specifies the name of the EmptyDisk to be used.
func WithEmptyDisk(diskName string, bus v1.DiskBus, capacity resource.Quantity) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, bus))
		addVolume(vmi, newEmptyDisk(diskName, capacity))
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

func newPersistentVolumeClaimVolume(name, claimName string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				}},
		},
	}
}

func newDataVolume(name, dataVolumeName string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name: dataVolumeName,
			},
		},
	}
}

func newEmptyDisk(name string, capacity resource.Quantity) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			EmptyDisk: &v1.EmptyDiskSource{
				Capacity: capacity,
			},
		},
	}
}
