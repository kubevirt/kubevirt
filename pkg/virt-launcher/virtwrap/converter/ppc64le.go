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

package converter

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

// Ensure that there is a compile error should the struct not implement the archConverter interface anymore.
var _ = ArchConverter(&archConverterPPC64{})

type archConverterPPC64 struct{}

func (archConverterPPC64) GetArchitecture() string {
	return "ppc64le"
}

func (archConverterPPC64) addGraphicsDevice(_ *v1.VirtualMachineInstance, domain *api.Domain, _ bool) {
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

func (archConverterPPC64) scsiController(model string, driver *api.ControllerDriver) api.Controller {
	return defaultSCSIController(model, driver)
}

func (archConverterPPC64) isUSBNeeded(_ *v1.VirtualMachineInstance) bool {
	//In ppc64le usb devices like mouse / keyboard are set by default,
	//so we can't disable the controller otherwise we run into the following error:
	//"unsupported configuration: USB is disabled for this domain, but USB devices are present in the domain XML"
	return true
}

func (archConverterPPC64) supportCPUHotplug() bool {
	return true
}

func (archConverterPPC64) isSMBiosNeeded() bool {
	// SMBios option does not work in Power, attempting to set it will result in the following error message:
	// "Option not supported for this target" issued by qemu-system-ppc64, so don't set it in case GOARCH is ppc64le
	return false
}

func (archConverterPPC64) transitionalModelType(useVirtioTransitional bool) string {
	return defaultTransitionalModelType(useVirtioTransitional)
}

func (archConverterPPC64) isROMTuningSupported() bool {
	return true
}

func (archConverterPPC64) requiresMPXCPUValidation() bool {
	// skip the mpx CPU feature validation for anything that is not x86 as it is not supported.
	return false
}

func (archConverterPPC64) shouldVerboseLogsBeEnabled() bool {
	return false
}

func (archConverterPPC64) hasVMPort() bool {
	return false
}
