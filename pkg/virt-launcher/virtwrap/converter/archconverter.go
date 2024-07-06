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
	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	graphicsDeviceDefaultHeads uint = 1
	graphicsDeviceDefaultVRAM  uint = 16384
)

type archConverter interface {
	addGraphicsDevice(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *ConverterContext)
	scsiController(c *ConverterContext, driver *api.ControllerDriver) api.Controller
	isUSBNeeded(vmi *v1.VirtualMachineInstance) bool
}

func newArchConverter(arch string) archConverter {
	switch {
	case isARM64(arch):
		return archConverterARM64{}
	case isPPC64(arch):
		return archConverterPPC64{}
	case isS390X(arch):
		return archConverterS390X{}
	case isAMD64(arch):
		return archConverterAMD64{}
	default:
		log.Log.Warning("Trying to create an arch converter from an unknown arch: " + arch + ". Falling back to AMD64")
		return archConverterAMD64{}
	}
}

func defaultSCSIController(c *ConverterContext, driver *api.ControllerDriver) api.Controller {
	return api.Controller{
		Type:   "scsi",
		Index:  "0",
		Model:  InterpretTransitionalModelType(&c.UseVirtioTransitional, c.Architecture),
		Driver: driver,
	}
}
