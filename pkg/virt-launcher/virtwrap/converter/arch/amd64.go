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

package arch

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

// Ensure that there is a compile error should the struct not implement the archConverter interface anymore.
var _ = Converter(&converterAMD64{})

type converterAMD64 struct{}

func (converterAMD64) GetArchitecture() string {
	return amd64
}

func (converterAMD64) AddGraphicsDevice(_ *v1.VirtualMachineInstance, domain *api.Domain, isEFI bool) {
	// For AMD64 + EFI, use bochs. For BIOS, use VGA
	if isEFI {
		domain.Spec.Devices.Video = []api.Video{
			{
				Model: api.VideoModel{
					Type:  "bochs",
					Heads: pointer.P(graphicsDeviceDefaultHeads),
				},
			},
		}
	} else {
		domain.Spec.Devices.Video = []api.Video{
			{
				Model: api.VideoModel{
					Type:  "vga",
					Heads: pointer.P(graphicsDeviceDefaultHeads),
					VRam:  pointer.P(graphicsDeviceDefaultVRAM),
				},
			},
		}
	}
}

func (converterAMD64) ScsiController(model string, driver *api.ControllerDriver) api.Controller {
	return defaultSCSIController(model, driver)
}

func (converterAMD64) IsUSBNeeded(vmi *v1.VirtualMachineInstance) bool {
	for i := range vmi.Spec.Domain.Devices.Inputs {
		if vmi.Spec.Domain.Devices.Inputs[i].Bus == "usb" {
			return true
		}
	}

	for i := range vmi.Spec.Domain.Devices.Disks {
		disk := vmi.Spec.Domain.Devices.Disks[i].Disk

		if disk != nil && disk.Bus == v1.DiskBusUSB {
			return true
		}
	}

	if vmi.Spec.Domain.Devices.ClientPassthrough != nil {
		return true
	}

	if device.USBDevicesFound(vmi.Spec.Domain.Devices.HostDevices) {
		return true
	}

	return false
}

func (converterAMD64) SupportCPUHotplug() bool {
	return true
}

func (converterAMD64) IsSMBiosNeeded() bool {
	return true
}

func (converterAMD64) TransitionalModelType(useVirtioTransitional bool) string {
	return defaultTransitionalModelType(useVirtioTransitional)
}

func (converterAMD64) IsROMTuningSupported() bool {
	return true
}

func (converterAMD64) RequiresMPXCPUValidation() bool {
	return true
}

func (converterAMD64) ShouldVerboseLogsBeEnabled() bool {
	return true
}

func (converterAMD64) HasVMPort() bool {
	return true
}
