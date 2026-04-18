/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package types

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "kubevirt.io/api/core/v1"
)

func IsStorageVolume(volume *v1.Volume) bool {
	return volume.PersistentVolumeClaim != nil || volume.DataVolume != nil
}

func IsDeclarativeHotplugVolume(vol *v1.Volume) bool {
	if vol == nil {
		return false
	}
	volSrc := vol.VolumeSource
	if volSrc.PersistentVolumeClaim != nil && volSrc.PersistentVolumeClaim.Hotpluggable {
		return true
	}
	if volSrc.DataVolume != nil && volSrc.DataVolume.Hotpluggable {
		return true
	}

	return false
}

func IsHotpluggableVolumeSource(vol *v1.Volume) bool {
	return IsStorageVolume(vol) ||
		vol.MemoryDump != nil
}

func IsHotplugVolume(vol *v1.Volume) bool {
	if vol == nil {
		return false
	}
	if IsDeclarativeHotplugVolume(vol) {
		return true
	}
	volSrc := vol.VolumeSource
	if volSrc.MemoryDump != nil && volSrc.MemoryDump.PersistentVolumeClaimVolumeSource.Hotpluggable {
		return true
	}

	return false
}

func IsUtilityVolume(vmi *v1.VirtualMachineInstance, volumeName string) bool {
	for _, utilityVolume := range vmi.Spec.UtilityVolumes {
		if utilityVolume.Name == volumeName {
			return true
		}
	}
	return false
}

func GetHotplugVolumes(vmi *v1.VirtualMachineInstance, virtlauncherPod *k8sv1.Pod) []*v1.Volume {
	hotplugVolumes := make([]*v1.Volume, 0)
	podVolumes := virtlauncherPod.Spec.Volumes
	vmiVolumes := vmi.Spec.Volumes

	podVolumeMap := make(map[string]k8sv1.Volume)
	for _, podVolume := range podVolumes {
		podVolumeMap[podVolume.Name] = podVolume
	}
	for _, vmiVolume := range vmiVolumes {
		if _, ok := podVolumeMap[vmiVolume.Name]; !ok && IsHotpluggableVolumeSource(&vmiVolume) {
			hotplugVolumes = append(hotplugVolumes, vmiVolume.DeepCopy())
		}
	}

	// Also include utility volumes, converting them to regular volumes
	for _, utilityVolume := range vmi.Spec.UtilityVolumes {
		if _, ok := podVolumeMap[utilityVolume.Name]; !ok {
			volume := &v1.Volume{
				Name: utilityVolume.Name,
				VolumeSource: v1.VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: utilityVolume.PersistentVolumeClaimVolumeSource,
						Hotpluggable:                      true,
					},
				},
			}
			hotplugVolumes = append(hotplugVolumes, volume)
		}
	}

	return hotplugVolumes
}

func GetVolumesByName(vmiSpec *v1.VirtualMachineInstanceSpec) map[string]*v1.Volume {
	volumes := map[string]*v1.Volume{}
	for _, vol := range vmiSpec.Volumes {
		volumes[vol.Name] = vol.DeepCopy()
	}
	return volumes
}

func GetFilesystemsFromVolumes(vmi *v1.VirtualMachineInstance) map[string]*v1.Filesystem {
	fs := map[string]*v1.Filesystem{}

	for _, f := range vmi.Spec.Domain.Devices.Filesystems {
		fs[f.Name] = f.DeepCopy()
	}

	return fs
}

func IsMigratedVolume(name string, vmi *v1.VirtualMachineInstance) bool {
	for _, v := range vmi.Status.MigratedVolumes {
		if v.VolumeName == name {
			return true
		}
	}
	return false
}

func GetTotalSizeMigratedVolumes(vmi *v1.VirtualMachineInstance) *resource.Quantity {
	size := int64(0)
	for _, v := range vmi.Status.MigratedVolumes {
		if v.SourcePVCInfo == nil {
			continue
		}
		if s := GetDiskCapacity(v.SourcePVCInfo); s != nil {
			size += *s
		}
	}

	return resource.NewQuantity(size, resource.BinarySI)
}
