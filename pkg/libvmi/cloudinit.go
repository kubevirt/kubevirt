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

	k8scorev1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

const cloudInitDiskName = "disk1"

// WithCloudInitNoCloudUserData adds cloud-init no-cloud user data.
func WithCloudInitNoCloudUserData(data string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitNoCloud(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		volume.CloudInitNoCloud.UserData = data
		volume.CloudInitNoCloud.UserDataBase64 = ""
	}
}

// WithCloudInitNoCloudEncodedUserData adds cloud-init no-cloud base64-encoded user data
func WithCloudInitNoCloudEncodedUserData(data string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitNoCloud(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		encodedData := base64.StdEncoding.EncodeToString([]byte(data))
		volume.CloudInitNoCloud.UserData = ""
		volume.CloudInitNoCloud.UserDataBase64 = encodedData
	}
}

// WithCloudInitNoCloudNetworkData adds cloud-init no-cloud network data.
func WithCloudInitNoCloudNetworkData(data string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitNoCloud(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		volume.CloudInitNoCloud.NetworkData = data
		volume.CloudInitNoCloud.NetworkDataBase64 = ""
		volume.CloudInitNoCloud.NetworkDataSecretRef = nil
	}
}

// WithCloudInitNoCloudEncodedNetworkData adds cloud-init no-cloud base64 encoded network data.
func WithCloudInitNoCloudEncodedNetworkData(networkData string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitNoCloud(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		volume.CloudInitNoCloud.NetworkDataBase64 = base64.StdEncoding.EncodeToString([]byte(networkData))
		volume.CloudInitNoCloud.NetworkData = ""
		volume.CloudInitNoCloud.NetworkDataSecretRef = nil
	}
}

// WithCloudInitNoCloudNetworkDataSecretName adds cloud-init no-cloud network data from secret.
func WithCloudInitNoCloudNetworkDataSecretName(secretName string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitNoCloud(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		volume.CloudInitNoCloud.NetworkDataSecretRef = &k8scorev1.LocalObjectReference{Name: secretName}
		volume.CloudInitNoCloud.NetworkData = ""
		volume.CloudInitNoCloud.NetworkDataBase64 = ""
	}
}

// WithCloudInitConfigDriveUserData adds cloud-init config-drive user data.
func WithCloudInitConfigDriveUserData(data string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitConfigDrive(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		volume.CloudInitConfigDrive.UserData = data
		volume.CloudInitConfigDrive.UserDataBase64 = ""
	}
}

// WithCloudInitConfigDriveEncodedUserData adds cloud-init config-drive with encoded user data.
func WithCloudInitConfigDriveEncodedUserData(data string) Option {
	return func(vmi *v1.VirtualMachineInstance) {
		addDiskVolumeWithCloudInitConfigDrive(vmi, cloudInitDiskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, cloudInitDiskName)
		volume.CloudInitConfigDrive.UserData = ""
		volume.CloudInitConfigDrive.UserDataBase64 = base64.StdEncoding.EncodeToString([]byte(data))
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
