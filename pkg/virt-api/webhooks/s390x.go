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
 * Copyright the KubeVirt Authors.
 *
 */

/*
 * s390x utilities are in the webhooks package because they are used both
 * by validation and mutation webhooks.
 */
package webhooks

import (
	v1 "kubevirt.io/api/core/v1"
)

func setDefaultS390xDisksBus(spec *v1.VirtualMachineInstanceSpec) {
	bus := v1.DiskBusVirtio

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

// SetS390xDefaults is mutating function for mutating-webhook
func SetS390xDefaults(spec *v1.VirtualMachineInstanceSpec) {
	setDefaultS390xDisksBus(spec)
}
