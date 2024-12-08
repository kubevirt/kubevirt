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
var _ = ArchConverter(&archConverterS390X{})

type archConverterS390X struct{}

func (archConverterS390X) GetArchitecture() string {
	return "s390x"
}

func (archConverterS390X) addGraphicsDevice(_ *v1.VirtualMachineInstance, domain *api.Domain, _ *ConverterContext) {
	domain.Spec.Devices.Video = []api.Video{
		{
			Model: api.VideoModel{
				Type:  v1.VirtIO,
				Heads: pointer.P(graphicsDeviceDefaultHeads),
			},
		},
	}
}

func (archConverterS390X) scsiController(_ *ConverterContext, driver *api.ControllerDriver) api.Controller {
	return api.Controller{
		Type:   "scsi",
		Index:  "0",
		Model:  "virtio-scsi",
		Driver: driver,
	}
}

func (archConverterS390X) isUSBNeeded(_ *v1.VirtualMachineInstance) bool {
	return false
}

func (archConverterS390X) supportCPUHotplug() bool {
	return true
}

func (archConverterS390X) isSMBiosNeeded() bool {
	return false
}

func (archConverterS390X) transitionalModelType(_ bool) string {
	// "virtio-non-transitional" and "virtio-transitional" are both PCI devices, so they don't work on s390x.
	// "virtio" is a generic model that can be either PCI or CCW depending on the machine type
	return "virtio"
}

func (archConverterS390X) isROMTuningSupported() bool {
	return false
}

func (archConverterS390X) requiresMPXCPUValidation() bool {
	// skip the mpx CPU feature validation for anything that is not x86 as it is not supported.
	return false
}

func (archConverterS390X) shouldVerboseLogsBeEnabled() bool {
	return false
}
