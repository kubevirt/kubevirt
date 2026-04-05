/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

//nolint:gocyclo
package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

func applyDiskPreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for diskIndex := range vmiSpec.Domain.Devices.Disks {
		vmiDisk := &vmiSpec.Domain.Devices.Disks[diskIndex]
		// If we don't have a target device defined default to a DiskTarget so we can apply preferences
		if vmiDisk.DiskDevice.Disk == nil && vmiDisk.DiskDevice.CDRom == nil && vmiDisk.DiskDevice.LUN == nil {
			vmiDisk.DiskDevice.Disk = &virtv1.DiskTarget{}
		}

		if vmiDisk.DiskDevice.Disk != nil {
			if preferenceSpec.Devices.PreferredDiskBus != "" && vmiDisk.DiskDevice.Disk.Bus == "" {
				vmiDisk.DiskDevice.Disk.Bus = preferenceSpec.Devices.PreferredDiskBus
			}

			if preferenceSpec.Devices.PreferredDiskBlockSize != nil && vmiDisk.BlockSize == nil {
				vmiDisk.BlockSize = preferenceSpec.Devices.PreferredDiskBlockSize.DeepCopy()
			}

			if preferenceSpec.Devices.PreferredDiskCache != "" && vmiDisk.Cache == "" {
				vmiDisk.Cache = preferenceSpec.Devices.PreferredDiskCache
			}

			if preferenceSpec.Devices.PreferredDiskIO != "" && vmiDisk.IO == "" {
				vmiDisk.IO = preferenceSpec.Devices.PreferredDiskIO
			}

			if preferenceSpec.Devices.PreferredDiskDedicatedIoThread != nil &&
				vmiDisk.DedicatedIOThread == nil &&
				vmiDisk.DiskDevice.Disk.Bus == virtv1.DiskBusVirtio {
				vmiDisk.DedicatedIOThread = pointer.P(*preferenceSpec.Devices.PreferredDiskDedicatedIoThread)
			}
		} else if vmiDisk.DiskDevice.CDRom != nil {
			if preferenceSpec.Devices.PreferredCdromBus != "" && vmiDisk.DiskDevice.CDRom.Bus == "" {
				vmiDisk.DiskDevice.CDRom.Bus = preferenceSpec.Devices.PreferredCdromBus
			}
		} else if vmiDisk.DiskDevice.LUN != nil {
			if preferenceSpec.Devices.PreferredLunBus != "" && vmiDisk.DiskDevice.LUN.Bus == "" {
				vmiDisk.DiskDevice.LUN.Bus = preferenceSpec.Devices.PreferredLunBus
			}
		}
	}
}
