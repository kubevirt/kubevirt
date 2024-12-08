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
package defaults

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
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

// Disable ACPI Feature by default on s390x, since it is not supported
func setS390xDefaultFeatures(spec *v1.VirtualMachineInstanceSpec) {
	featureStateDisabled := v1.FeatureState{Enabled: pointer.P[bool](false)}
	if spec.Domain.Features == nil {
		spec.Domain.Features = &v1.Features{
			ACPI: featureStateDisabled,
		}
	} else if spec.Domain.Features.ACPI.Enabled == nil {
		spec.Domain.Features.ACPI.Enabled = pointer.P[bool](false)
	}
}

// SetS390xDefaults is mutating function for mutating-webhook
func SetS390xDefaults(spec *v1.VirtualMachineInstanceSpec) {
	setDefaultS390xDisksBus(spec)
}

func IsS390X(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	return vmiSpec.Architecture == "s390x"
}
