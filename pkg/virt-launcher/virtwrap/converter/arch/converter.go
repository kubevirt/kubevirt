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
	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	amd64 = "amd64"
	arm64 = "arm64"
	s390x = "s390x"
)

type Converter interface {
	GetArchitecture() string
	ScsiController(model string, driver *api.ControllerDriver) api.Controller
	IsUSBNeeded(vmi *v1.VirtualMachineInstance) bool
	SupportCPUHotplug() bool
	IsSMBiosNeeded() bool
	HasVMPort() bool
	TransitionalModelType(useVirtioTransitional bool) string
	IsROMTuningSupported() bool
	RequiresMPXCPUValidation() bool
	ShouldVerboseLogsBeEnabled() bool
	ConvertWatchdog(source *v1.Watchdog, watchdog *api.Watchdog) error
	SupportPCIHole64Disabling() bool
}

func NewConverter(arch string) Converter {
	switch arch {
	case arm64:
		return converterARM64{}
	case s390x:
		return converterS390X{}
	case amd64:
		return converterAMD64{}
	default:
		log.Log.Warning("Trying to create an arch converter from an unknown arch: " + arch + ". Falling back to AMD64")
		return converterAMD64{}
	}
}

func defaultSCSIController(model string, driver *api.ControllerDriver) api.Controller {
	return api.Controller{
		Type:   "scsi",
		Index:  "0",
		Model:  model,
		Driver: driver,
	}
}

func defaultTransitionalModelType(useVirtioTransitional bool) string {
	if useVirtioTransitional {
		return "virtio-transitional"
	}
	return "virtio-non-transitional"
}
