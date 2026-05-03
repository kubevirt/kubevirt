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

package capabilities

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/hooks"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
)

// Capability constants - each represents a feature that may need validation or blocking
const (
	CapVsock                     CapabilityKey = "domain.devices.vsock"
	CapVirtiofsStorage           CapabilityKey = "domain.devices.filesystems.virtiofs-storage"
	CapDownwardMetricsVolume     CapabilityKey = "volumes.downwardmetrics"
	CapDownwardMetricsDevice     CapabilityKey = "domain.devices.downwardmetrics"
	CapDeclarativeHotplugVolumes CapabilityKey = "domain.devices.disks.declarative-hotplug"
	CapNUMAGuestMapping          CapabilityKey = "domain.cpu.numa.guest-mapping-passthrough"
	CapHostDevicesPassthrough    CapabilityKey = "domain.devices.hostdevices"
	CapHostDisk                  CapabilityKey = "volumes.hostdisk"
	CapIgnitionSupport           CapabilityKey = "metadata.annotations.ignition"
	CapSidecarHooks              CapabilityKey = "metadata.annotations.hooks.sidecars"
	CapPersistentReservation     CapabilityKey = "domain.devices.disks.lun.persistent-reservation"
	CapVideoConfig               CapabilityKey = "domain.devices.video"
	CapRebootPolicy              CapabilityKey = "domain.rebootpolicy"
	CapReservedOverheadMemlock   CapabilityKey = "domain.memory.reserved-overhead"
)

// Define CapVsock capability
var CapVsockDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.Devices.AutoattachVSOCK != nil && *vmi.Spec.Domain.Devices.AutoattachVSOCK {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("devices").Child("autoattachVSOCK")}
		}
		return nil
	},
}

// Define VirtioFS Storage capability - filesystems with storage volumes (PVC/DataVolume)
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

// Define Downward Metrics Volume capability
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

// Define Downward Metrics Device capability
var CapDownwardMetricsDeviceDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if downwardmetrics.HasDevice(&vmi.Spec) {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("devices").Child("downwardMetrics")}
		}
		return nil
	},
}

// Define Declarative Hotplug Volumes capability - empty CD-ROM disks
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

// Define NUMA Guest Mapping capability
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

// Define Host Devices Passthrough capability
var CapHostDevicesPassthroughDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.Devices.HostDevices != nil {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("devices").Child("hostDevices")}
		}
		return []*field.Path{}
	},
}

// Define Host Disk capability
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

// Define Persistent Reservation capability
var CapPersistentReservationDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		// Find all disks with persistent reservation
		var paths []*field.Path
		for i, disk := range vmi.Spec.Domain.Devices.Disks {
			if disk.LUN != nil && disk.LUN.Reservation {
				paths = append(paths, field.NewPath("spec").Child("domain").Child("devices").Child("disks").Index(i).Child("lun").Child("reservation"))
			}
		}
		return paths
	},
}

// Define Video Config capability
var CapVideoConfigDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.Devices.Video != nil {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("devices").Child("video")}
		}
		return nil
	},
}

// Define Reboot Policy capability
var CapRebootPolicyDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.RebootPolicy != nil {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("rebootPolicy")}
		}
		return nil
	},
}

// Define Ignition Support capability
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

// Define Sidecar Hooks capability
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

// Define Reserved Overhead Memlock capability
var CapReservedOverheadMemlockDef = Capability{
	GetRequiredFields: func(vmi *v1.VirtualMachineInstance) []*field.Path {
		if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.ReservedOverhead != nil {
			return []*field.Path{field.NewPath("spec").Child("domain").Child("memory").Child("reservedOverhead")}
		}
		return nil
	},
}
