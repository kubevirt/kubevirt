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

package libvmi

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	defaultDiskSize = "1Gi"
)

// WithContainerDisk specifies the disk name and the name of the container image to be used.
func WithContainerDisk(diskName, imageName string, options ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio, options...))
		addVolume(vmi, newContainerVolume(diskName, imageName, ""))
	}
}

// WithContainerDiskAndPullPolicy specifies the disk name, the name of the container image and Pull Policy to be used.
func WithContainerDiskAndPullPolicy(diskName, imageName string, imagePullPolicy k8sv1.PullPolicy, diskOpts ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio, diskOpts...))
		addVolume(vmi, newContainerVolume(diskName, imageName, imagePullPolicy))
	}
}

// WithContainerSATADisk specifies the disk name and the name of the container image to be used.
func WithContainerSATADisk(diskName, imageName string, diskOpts ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusSATA, diskOpts...))
		addVolume(vmi, newContainerVolume(diskName, imageName, ""))
	}
}

// WithPersistentVolumeClaim specifies the name of the PersistentVolumeClaim to be used.
func WithPersistentVolumeClaim(diskName, pvcName string, diskOpts ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio, diskOpts...))
		addVolume(vmi, newPersistentVolumeClaimVolume(diskName, pvcName, false))
	}
}

// WithHotplugPersistentVolumeClaim specifies the name of the hotpluggable PersistentVolumeClaim to be used.
func WithHotplugPersistentVolumeClaim(diskName, pvcName string, diskOpts ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio, diskOpts...))
		addVolume(vmi, newPersistentVolumeClaimVolume(diskName, pvcName, true))
	}
}

// WithEphemeralPersistentVolumeClaim specifies the name of the Ephemeral.PersistentVolumeClaim to be used.
func WithEphemeralPersistentVolumeClaim(diskName, pvcName string, diskOpts ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusSATA, diskOpts...))
		addVolume(vmi, newEphemeralPersistentVolumeClaimVolume(diskName, pvcName))
	}
}

// WithDataVolume specifies the name of the DataVolume to be used.
func WithDataVolume(diskName, pvcName string, diskOpts ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio, diskOpts...))
		addVolume(vmi, newDataVolume(diskName, pvcName, false))
	}
}

// WithHotplugDataVolume specifies the name of the hotpluggable DataVolume to be used.
func WithHotplugDataVolume(diskName, pvcName string, diskOpts ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusSCSI, diskOpts...))
		addVolume(vmi, newDataVolume(diskName, pvcName, true))
	}
}

// WithEmptyDisk specifies the name of the EmptyDisk to be used.
func WithEmptyDisk(diskName string, bus v1.DiskBus, capacity resource.Quantity, diskOpts ...DiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, bus, diskOpts...))
		addVolume(vmi, newEmptyDisk(diskName, capacity))
	}
}

// WithCDRom specifies a CDRom drive backed by a PVC to be used.
func WithCDRom(cdRomName string, bus v1.DiskBus, claimName string) Option {
	return WithCDRomAndVolume(bus, newPersistentVolumeClaimVolume(cdRomName, claimName, false))
}

// WithCDRomAndVolume specifies a CDRom drive backed by given volume and given bus.
func WithCDRomAndVolume(bus v1.DiskBus, volume v1.Volume) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newCDRom(volume.Name, bus))
		addVolume(vmi, volume)
	}
}

// WithEmptyCDRom specifies a CDRom drive with no volume
func WithEmptyCDRom(bus v1.DiskBus, name string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newCDRom(name, bus))
	}
}

// WithEphemeralCDRom specifies a CDRom drive to be used.
func WithEphemeralCDRom(cdRomName string, bus v1.DiskBus, claimName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newCDRom(cdRomName, bus))
		addVolume(vmi, newContainerVolume(cdRomName, claimName, ""))
	}
}

// WithFilesystemPVC specifies a filesystem backed by a PVC to be used.
func WithFilesystemPVC(claimName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addFilesystem(vmi, newVirtiofsFilesystem(claimName))
		addVolume(vmi, newPersistentVolumeClaimVolume(claimName, claimName, false))
	}
}

// WithFilesystemDV specifies a filesystem backed by a DV to be used.
func WithFilesystemDV(dataVolumeName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addFilesystem(vmi, newVirtiofsFilesystem(dataVolumeName))
		addVolume(vmi, newDataVolume(dataVolumeName, dataVolumeName, false))
	}
}

func WithPersistentVolumeClaimLun(diskName, pvcName string, reservation bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newLun(diskName, reservation))
		addVolume(vmi, newPersistentVolumeClaimVolume(diskName, pvcName, false))
	}
}

func WithHostDisk(diskName, path string, diskType v1.HostDiskType, opts ...HostDiskOption) Option {
	var capacity string
	if diskType == v1.HostDiskExistsOrCreate {
		capacity = defaultDiskSize
	}
	return WithHostDiskAndCapacity(diskName, path, diskType, capacity, opts...)
}

func WithHostDiskAndCapacity(diskName, path string, diskType v1.HostDiskType, capacity string, opts ...HostDiskOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDisk(vmi, newDisk(diskName, v1.DiskBusVirtio))
		addVolume(vmi, newHostDisk(diskName, path, diskType, capacity, opts...))
	}
}

type HostDiskOption func(v *v1.Volume)

func WithSharedHostDisk(shared bool) HostDiskOption {
	return func(v *v1.Volume) {
		v.HostDisk.Shared = pointer.P(shared)
	}
}

// WithIOThreadsPolicy sets the WithIOThreadPolicy parameter
func WithIOThreadsPolicy(policy v1.IOThreadsPolicy) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.IOThreadsPolicy = &policy
	}
}

func WithIOThreads(iothreads v1.DiskIOThreads) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Domain.IOThreads = pointer.P(iothreads)
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

func addFilesystem(vmi *v1.VirtualMachineInstance, filesystem v1.Filesystem) {
	if filesystemExists(vmi, filesystem) {
		panic(fmt.Errorf("filesystem %s already exists", filesystem.Name))
	}

	vmi.Spec.Domain.Devices.Filesystems = append(vmi.Spec.Domain.Devices.Filesystems, filesystem)
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

func filesystemExists(vmi *v1.VirtualMachineInstance, filesystem v1.Filesystem) bool {
	for _, f := range vmi.Spec.Domain.Devices.Filesystems {
		if f.Name == filesystem.Name {
			return true
		}
	}
	return false
}

type DiskOption func(vm *v1.Disk)

func newDisk(name string, bus v1.DiskBus, opts ...DiskOption) v1.Disk {
	d := v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: bus,
			},
		},
	}
	for _, f := range opts {
		f(&d)
	}

	return d
}

func WithDedicatedIOThreads(enabled bool) DiskOption {
	return func(d *v1.Disk) {
		d.DedicatedIOThread = pointer.P(enabled)
	}
}

func WithSerial(serial string) DiskOption {
	return func(d *v1.Disk) {
		d.Serial = serial
	}
}

func newCDRom(name string, bus v1.DiskBus) v1.Disk {
	return v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			CDRom: &v1.CDRomTarget{
				Bus: bus,
			},
		},
	}
}

func newVirtiofsFilesystem(name string) v1.Filesystem {
	return v1.Filesystem{
		Name:     name,
		Virtiofs: &v1.FilesystemVirtiofs{},
	}
}

func newLun(name string, reservation bool) v1.Disk {
	return v1.Disk{
		Name: name,
		DiskDevice: v1.DiskDevice{
			LUN: &v1.LunTarget{
				Bus:         v1.DiskBusSCSI,
				Reservation: reservation,
			},
		},
	}
}

func newVolume(name string) v1.Volume {
	return v1.Volume{Name: name}
}

func newContainerVolume(name, image string, imagePullPolicy k8sv1.PullPolicy) v1.Volume {
	container := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			ContainerDisk: &v1.ContainerDiskSource{
				Image: image,
			},
		},
	}
	if imagePullPolicy != "" {
		container.ContainerDisk.ImagePullPolicy = imagePullPolicy
	}
	return container
}

func newPersistentVolumeClaimVolume(name, claimName string, hotpluggable bool) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
				Hotpluggable: hotpluggable,
			},
		},
	}
}

func newEphemeralPersistentVolumeClaimVolume(name, claimName string) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			Ephemeral: &v1.EphemeralVolumeSource{
				PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: claimName,
				},
			},
		},
	}
}

func newDataVolume(name, dataVolumeName string, hotpluggable bool) v1.Volume {
	return v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			DataVolume: &v1.DataVolumeSource{
				Name:         dataVolumeName,
				Hotpluggable: hotpluggable,
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

func newHostDisk(name, path string, diskType v1.HostDiskType, capacity string, opts ...HostDiskOption) v1.Volume {
	hostDisk := v1.HostDisk{
		Path: path,
		Type: diskType,
	}

	// Set capacity if provided
	if capacity != "" {
		hostDisk.Capacity = resource.MustParse(capacity)
	}

	v := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			HostDisk: &hostDisk,
		},
	}

	for _, f := range opts {
		f(&v)
	}

	return v
}

func WithSupplementalPoolThreadCount(count uint32) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.IOThreads == nil {
			vmi.Spec.Domain.IOThreads = &v1.DiskIOThreads{}
		}
		vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount = pointer.P(count)
	}
}
