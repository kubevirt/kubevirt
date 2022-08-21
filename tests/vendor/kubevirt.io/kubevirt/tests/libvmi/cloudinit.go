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
	"encoding/base64"

	v1 "kubevirt.io/api/core/v1"
)

const cloudInitDiskName = "disk1"

// WithCloudInitNoCloudUserData adds cloud-init no-cloud user data.
func WithCloudInitNoCloudUserData(data string, b64Encoding bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitNoCloud(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		if b64Encoding {
			encodedData := base64.StdEncoding.EncodeToString([]byte(data))
			volume.CloudInitNoCloud.UserData = ""
			volume.CloudInitNoCloud.UserDataBase64 = encodedData
		} else {
			volume.CloudInitNoCloud.UserData = data
			volume.CloudInitNoCloud.UserDataBase64 = ""
		}
	}
}

// WithCloudInitNoCloudNetworkData adds cloud-init no-cloud network data.
func WithCloudInitNoCloudNetworkData(data string, b64Encoding bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitNoCloud(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		if b64Encoding {
			encodedData := base64.StdEncoding.EncodeToString([]byte(data))
			volume.CloudInitNoCloud.NetworkDataBase64 = encodedData
		} else {
			volume.CloudInitNoCloud.NetworkData = data
		}
	}
}

// WithCloudInitConfigDriveData adds cloud-init config-drive user data.
func WithCloudInitConfigDriveData(data string, b64Encoding bool) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitConfigDrive(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		if b64Encoding {
			encodedData := base64.StdEncoding.EncodeToString([]byte(data))
			volume.CloudInitConfigDrive.UserData = ""
			volume.CloudInitConfigDrive.UserDataBase64 = encodedData
		} else {
			volume.CloudInitConfigDrive.UserData = data
			volume.CloudInitConfigDrive.UserDataBase64 = ""
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
