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

package storage

import (
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type DeviceNamer struct {
	ExistingNameMap map[string]string
	UsedDeviceMap   map[string]string
}

func NewDeviceNamer(volumeStatuses []v1.VolumeStatus, disks []v1.Disk) map[string]DeviceNamer {
	prefixMap := make(map[string]DeviceNamer)
	volumeTargetMap := make(map[string]string)
	for _, volumeStatus := range volumeStatuses {
		if volumeStatus.Target != "" {
			volumeTargetMap[volumeStatus.Name] = volumeStatus.Target
		}
	}

	for _, disk := range disks {
		var prefix string
		switch {
		case disk.Disk != nil:
			prefix = getPrefixFromBus(disk.Disk.Bus)
		case disk.LUN != nil:
			prefix = getPrefixFromBus(disk.LUN.Bus)
		case disk.CDRom != nil:
			prefix = getPrefixFromBus(disk.CDRom.Bus)
		default:
			continue
		}

		if _, ok := prefixMap[prefix]; !ok {
			prefixMap[prefix] = DeviceNamer{
				ExistingNameMap: make(map[string]string),
				UsedDeviceMap:   make(map[string]string),
			}
		}
		namer := prefixMap[prefix]
		if _, ok := volumeTargetMap[disk.Name]; ok {
			namer.ExistingNameMap[disk.Name] = volumeTargetMap[disk.Name]
			namer.UsedDeviceMap[volumeTargetMap[disk.Name]] = disk.Name
		}
	}
	return prefixMap
}

func (n *DeviceNamer) getExistingVolumeValue(key string) (string, bool) {
	if _, ok := n.ExistingNameMap[key]; ok {
		return n.ExistingNameMap[key], true
	}
	return "", false
}

func (n *DeviceNamer) getExistingTargetValue(key string) (string, bool) {
	if _, ok := n.UsedDeviceMap[key]; ok {
		return n.UsedDeviceMap[key], true
	}
	return "", false
}

func MakeDeviceName(diskName string, bus v1.DiskBus, prefixMap map[string]DeviceNamer) (string, int) {
	prefix := getPrefixFromBus(bus)
	if _, ok := prefixMap[prefix]; !ok {
		// This should never happen since the prefix map is populated from all disks.
		prefixMap[prefix] = DeviceNamer{
			ExistingNameMap: make(map[string]string),
			UsedDeviceMap:   make(map[string]string),
		}
	}
	deviceNamer := prefixMap[prefix]
	if name, ok := deviceNamer.getExistingVolumeValue(diskName); ok {
		for i := 0; i < 26*26*26; i++ {
			calculatedName := FormatDeviceName(prefix, i)
			if calculatedName == name {
				return name, i
			}
		}
		log.Log.Error("Unable to determine index of device")
		return name, 0
	}
	// Name not found yet, generate next new one.
	for i := 0; i < 26*26*26; i++ {
		name := FormatDeviceName(prefix, i)
		if _, ok := deviceNamer.getExistingTargetValue(name); !ok {
			deviceNamer.ExistingNameMap[diskName] = name
			deviceNamer.UsedDeviceMap[name] = diskName
			return name, i
		}
	}
	return "", 0
}

// port of http://elixir.free-electrons.com/linux/v4.15/source/drivers/scsi/sd.c#L3211
func FormatDeviceName(prefix string, index int) string {
	base := int('z' - 'a' + 1)
	name := ""

	for index >= 0 {
		name = string(rune('a'+(index%base))) + name
		index = (index / base) - 1
	}
	return prefix + name
}

func getPrefixFromBus(bus v1.DiskBus) string {
	switch bus {
	case v1.DiskBusVirtio:
		return "vd"
	case v1.DiskBusSATA, v1.DiskBusSCSI, v1.DiskBusUSB:
		return "sd"
	default:
		log.Log.Errorf("Unrecognized bus '%s'", bus)
		return ""
	}
}

func GetVolumeNameByDisk(disk api.Disk) string {
	return disk.Alias.GetName()
}

// GetVolumeNameByTarget returns the volume name associated to the device target in the domain (e.g vda)
func GetVolumeNameByTarget(domain *api.Domain, target string) string {
	for _, d := range domain.Spec.Devices.Disks {
		if d.Target.Device == target {
			return GetVolumeNameByDisk(d)
		}
	}
	return ""
}
