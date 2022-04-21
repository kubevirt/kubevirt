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

	kvirtv1 "kubevirt.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"
)

// WithCloudInitNoCloudUserData adds cloud-init no-cloud user data.
func WithCloudInitNoCloudUserData(data string, b64Encoding bool) Option {
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		diskName := "disk1"
		addDiskVolumeWithCloudInitNoCloud(vmi, diskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, diskName)
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
	return func(vmi *kvirtv1.VirtualMachineInstance) {
		diskName := "disk1"
		addDiskVolumeWithCloudInitNoCloud(vmi, diskName, v1.DiskBusVirtio)

		volume := getVolume(vmi, diskName)
		if b64Encoding {
			encodedData := base64.StdEncoding.EncodeToString([]byte(data))
			volume.CloudInitNoCloud.NetworkDataBase64 = encodedData
		} else {
			volume.CloudInitNoCloud.NetworkData = data
		}
	}
}

func addDiskVolumeWithCloudInitNoCloud(vmi *kvirtv1.VirtualMachineInstance, diskName string, bus v1.DiskBus) {
	addDisk(vmi, newDisk(diskName, bus))
	v := newVolume(diskName)
	setCloudInitNoCloud(&v, &kvirtv1.CloudInitNoCloudSource{})
	addVolume(vmi, v)
}

func setCloudInitNoCloud(volume *kvirtv1.Volume, source *kvirtv1.CloudInitNoCloudSource) {
	volume.VolumeSource = kvirtv1.VolumeSource{CloudInitNoCloud: source}
}
