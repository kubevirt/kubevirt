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
var _ = archConverter(&archConverterPPC64{})

type archConverterPPC64 struct{}

func (archConverterPPC64) addGraphicsDevice(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *ConverterContext) {
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

func (archConverterPPC64) scsiController(c *ConverterContext, driver *api.ControllerDriver) api.Controller {
	return defaultSCSIController(c, driver)
}

func (archConverterPPC64) isUSBNeeded(_ *v1.VirtualMachineInstance) bool {
	//In ppc64le usb devices like mouse / keyboard are set by default,
	//so we can't disable the controller otherwise we run into the following error:
	//"unsupported configuration: USB is disabled for this domain, but USB devices are present in the domain XML"
	return true
}
