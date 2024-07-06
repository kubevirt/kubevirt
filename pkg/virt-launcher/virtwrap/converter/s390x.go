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
var _ = archConverter(&archConverterS390X{})

type archConverterS390X struct{}

func (archConverterS390X) addGraphicsDevice(vmi *v1.VirtualMachineInstance, domain *api.Domain, _ *ConverterContext) {
	domain.Spec.Devices.Video = []api.Video{
		{
			Model: api.VideoModel{
				Type:  v1.VirtIO,
				Heads: pointer.P(graphicsDeviceDefaultHeads),
			},
		},
	}
}

func (archConverterS390X) scsiController(c *ConverterContext, driver *api.ControllerDriver) api.Controller {
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
