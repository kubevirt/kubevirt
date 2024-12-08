/* Licensed under the Apache License, Version 2.0 (the "License");
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
 * Copyright 2021
 *
 */
package defaults

import (
	v1 "kubevirt.io/api/core/v1"
)

func setDefaultAmd64DisksBus(spec *v1.VirtualMachineInstanceSpec) {
	// Setting SATA as the default bus since it is typically supported out of the box by
	// guest operating systems (we support only q35 and therefore IDE is not supported)
	// TODO: consider making this OS-specific (VIRTIO for linux, SATA for others)
	bus := v1.DiskBusSATA

	for i := range spec.Domain.Devices.Disks {
		disk := &spec.Domain.Devices.Disks[i].DiskDevice

		if disk.Disk != nil && disk.Disk.Bus == "" {
			disk.Disk.Bus = bus
		}
		if disk.CDRom != nil && disk.CDRom.Bus == "" {
			disk.CDRom.Bus = bus
		}
		if disk.LUN != nil && disk.LUN.Bus == "" {
			disk.LUN.Bus = bus
		}
	}
}

// SetAmd64Defaults is mutating function for mutating-webhook
func SetAmd64Defaults(spec *v1.VirtualMachineInstanceSpec) {
	setDefaultAmd64DisksBus(spec)
}
