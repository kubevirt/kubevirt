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
var _ = archConverter(&archConverterARM64{})

type archConverterARM64 struct{}

func (archConverterARM64) addGraphicsDevice(vmi *v1.VirtualMachineInstance, domain *api.Domain, _ *ConverterContext) {
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

func (archConverterARM64) scsiController(c *ConverterContext, driver *api.ControllerDriver) api.Controller {
	return defaultSCSIController(c, driver)
}

func (archConverterARM64) isUSBNeeded(_ *v1.VirtualMachineInstance) bool {
	return true
}
