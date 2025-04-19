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
	"kubevirt.io/kubevirt/pkg/libvmi/cloudinit"

	v1 "kubevirt.io/api/core/v1"
)

const CloudInitDiskName = "cloudinitdisk"

// WithCloudInitNoCloud adds cloud-init no-cloud sources.
func WithCloudInitNoCloud(opts ...cloudinit.NoCloudOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitNoCloud(vmi, CloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, CloudInitDiskName)

		for _, f := range opts {
			f(volume.CloudInitNoCloud)
		}
	}
}

// WithCloudInitConfigDrive adds cloud-init config-drive sources.
func WithCloudInitConfigDrive(opts ...cloudinit.ConfigDriveOption) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitConfigDrive(vmi, CloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, CloudInitDiskName)

		for _, f := range opts {
			f(volume.CloudInitConfigDrive)
		}
	}
}

func addDiskVolumeWithCloudInitConfigDrive(vmi *v1.VirtualMachineInstance, diskName string, bus v1.DiskBus) {
	addDisk(vmi, newDisk(diskName, bus))
	v := newVolume(diskName)
	v.VolumeSource = v1.VolumeSource{CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{}}
	addVolume(vmi, v)
}

func addDiskVolumeWithCloudInitNoCloud(vmi *v1.VirtualMachineInstance, diskName string, bus v1.DiskBus) {
	addDisk(vmi, newDisk(diskName, bus))
	v := newVolume(diskName)
	setCloudInitNoCloud(&v, &v1.CloudInitNoCloudSource{})
	addVolume(vmi, v)
}

func setCloudInitNoCloud(volume *v1.Volume, source *v1.CloudInitNoCloudSource) {
	volume.VolumeSource = v1.VolumeSource{CloudInitNoCloud: source}
}

func GetCloudInitVolume(vmi *v1.VirtualMachineInstance) *v1.Volume {
	return getVolume(vmi, CloudInitDiskName)
}
