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
 * Copyright The KubeVirt Authors.
 *
 */

package arch

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// Ensure that there is a compile error should the struct not implement the archConverter interface anymore.
var _ = Converter(&converterARM64{})

type converterARM64 struct{}

func (converterARM64) GetArchitecture() string {
	return arm64
}

func (converterARM64) AddGraphicsDevice(vmi *v1.VirtualMachineInstance, domain *api.Domain, _ bool) {
	// For arm64, qemu-kvm only support virtio-gpu display device, so set it as default video device.
	// tablet and keyboard devices are necessary for control the VM via vnc connection
	domain.Spec.Devices.Video = []api.Video{
		{
			Model: api.VideoModel{
				Type:  v1.VirtIO,
				Heads: pointer.P(graphicsDeviceDefaultHeads),
			},
		},
	}

	if !hasTabletDevice(vmi) {
		domain.Spec.Devices.Inputs = append(domain.Spec.Devices.Inputs,
			api.Input{
				Bus:  "usb",
				Type: "tablet",
			},
		)
	}

	domain.Spec.Devices.Inputs = append(domain.Spec.Devices.Inputs,
		api.Input{
			Bus:  "usb",
			Type: "keyboard",
		},
	)
}

func (converterARM64) ScsiController(model string, driver *api.ControllerDriver) api.Controller {
	return defaultSCSIController(model, driver)
}

func (converterARM64) IsUSBNeeded(_ *v1.VirtualMachineInstance) bool {
	return true
}

func (converterARM64) SupportCPUHotplug() bool {
	return false
}

func (converterARM64) IsSMBiosNeeded() bool {
	// ARM64 use UEFI boot by default, set SMBios is unnecessary.
	return false
}

func (converterARM64) TransitionalModelType(useVirtioTransitional bool) string {
	return defaultTransitionalModelType(useVirtioTransitional)
}

func (converterARM64) IsROMTuningSupported() bool {
	return true
}

func (converterARM64) RequiresMPXCPUValidation() bool {
	// skip the mpx CPU feature validation for anything that is not x86 as it is not supported.
	return false
}

func (converterARM64) ShouldVerboseLogsBeEnabled() bool {
	return false
}

func (converterARM64) HasVMPort() bool {
	return false
}

func hasTabletDevice(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.Devices.Inputs != nil {
		for _, device := range vmi.Spec.Domain.Devices.Inputs {
			if device.Type == "tablet" {
				return true
			}
		}
	}
	return false
}

func (converterARM64) ConvertWatchdog(source *v1.Watchdog, watchdog *api.Watchdog) error {
	return fmt.Errorf("watchdog is not supported on architecture ARM64")
}
