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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package common

import (
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

func GetHotplugVolumes(vmi *v1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod) []*v1.Volume {
	hotplugVolumes := make([]*v1.Volume, 0)
	podVolumes := virtlauncherPod.Spec.Volumes
	vmiVolumes := vmi.Spec.Volumes

	podVolumeMap := make(map[string]k8sv1.Volume)
	for _, podVolume := range podVolumes {
		podVolumeMap[podVolume.Name] = podVolume
	}
	for _, vmiVolume := range vmiVolumes {
		if _, ok := podVolumeMap[vmiVolume.Name]; !ok && (vmiVolume.DataVolume != nil || vmiVolume.PersistentVolumeClaim != nil || vmiVolume.MemoryDump != nil) {
			hotplugVolumes = append(hotplugVolumes, vmiVolume.DeepCopy())
		}
	}
	return hotplugVolumes
}
