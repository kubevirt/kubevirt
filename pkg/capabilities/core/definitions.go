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

package core

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/hooks"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

// Capability constants - each represents a feature that may need validation or blocking
const (
	CapVsock CapabilityKey = iota
	CapVirtiofsStorage
	CapDownwardMetricsVolume
	CapDownwardMetricsDevice
	CapDeclarativeHotplugVolumes
	CapNUMAGuestMapping
	CapHostDevicesPassthrough
	CapHostDisk
	CapIgnitionSupport
	CapSidecarHooks
	CapPersistentReservation
	CapVideoConfig
	CapRebootPolicy
	CapReservedOverheadMemlock
)

var CapVsockDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.Devices.AutoattachVSOCK != nil && *vmi.Spec.Domain.Devices.AutoattachVSOCK {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("devices").Child("autoattachVSOCK")}
		}
		return nil
	},
}

var CapVirtiofsStorageDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.Devices.Filesystems == nil {
			return nil
		}

		var paths []*field.Path
		volumes := storagetypes.GetVolumesByName(&vmi.Spec)

		for i, fs := range vmi.Spec.Domain.Devices.Filesystems {
			volume, ok := volumes[fs.Name]
			if !ok {
				continue
			}

			if storagetypes.IsStorageVolume(volume) {
				paths = append(paths, field.NewPath("spec").Child("domain").Child("devices").Child("filesystems").Index(i))
			}
		}
		return paths
	},
}

var CapDownwardMetricsVolumeDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		var paths []*field.Path
		for i, volume := range vmi.Spec.Volumes {
			if volume.DownwardMetrics != nil {
				paths = append(paths, field.NewPath("spec").Child("volumes").Index(i).Child("downwardMetrics"))
			}
		}
		return paths
	},
}

var CapDownwardMetricsDeviceDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if downwardmetrics.HasDevice(&vmi.Spec) {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("devices").Child("downwardMetrics")}
		}
		return nil
	},
}

var CapDeclarativeHotplugVolumesDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		volumeNameMap := make(map[string]*v1.Volume)

		for i, volume := range vmi.Spec.Volumes {
			volumeNameMap[volume.Name] = &vmi.Spec.Volumes[i]
		}

		var paths []*field.Path
		for i, disk := range vmi.Spec.Domain.Devices.Disks {
			_, volumeExists := volumeNameMap[disk.Name]

			if !volumeExists && disk.CDRom != nil {
				paths = append(paths, field.NewPath("spec").Child("domain").Child("devices").Child("disks").Index(i))
			}
		}
		return paths
	},
}

var CapNUMAGuestMappingDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.CPU != nil &&
			vmi.Spec.Domain.CPU.NUMA != nil &&
			vmi.Spec.Domain.CPU.NUMA.GuestMappingPassthrough != nil {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("cpu").Child("numa").Child("guestMappingPassthrough")}
		}
		return nil
	},
}

var CapHostDevicesPassthroughDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.Devices.HostDevices != nil {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("devices").Child("hostDevices")}
		}
		return []*field.Path{}
	},
}

var CapHostDiskDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		var paths []*field.Path
		for i, volume := range vmi.Spec.Volumes {
			if volume.HostDisk != nil {
				paths = append(paths, field.NewPath("spec").Child("volumes").Index(i))
			}
		}
		return paths
	},
}

var CapRebootPolicyDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.RebootPolicy != nil {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("rebootPolicy")}
		}
		return nil
	},
}

var CapIgnitionSupportDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Annotations == nil {
			return nil
		}
		_, exists := vmi.Annotations[v1.IgnitionAnnotation]
		if exists && vmi.Annotations[v1.IgnitionAnnotation] != "" {
			return []*field.Path{field.NewPath("metadata").Child("annotations").Child(v1.IgnitionAnnotation)}
		}
		return nil
	},
}

var CapSidecarHooksDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Annotations == nil {
			return nil
		}
		_, exists := vmi.Annotations[hooks.HookSidecarListAnnotationName]
		if exists && vmi.Annotations[hooks.HookSidecarListAnnotationName] != "" {
			return []*field.Path{field.NewPath("metadata").Child("annotations").Child(hooks.HookSidecarListAnnotationName)}
		}
		return nil
	},
}

var CapReservedOverheadMemlockDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.ReservedOverhead != nil {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("memory").Child("reservedOverhead")}
		}
		return nil
	},
}
