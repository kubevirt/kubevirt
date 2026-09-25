/*
Copyright The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vm

import (
	"k8s.io/apimachinery/pkg/types"

	"github.com/google/uuid"
	v1 "kubevirt.io/api/core/v1"
)

type FirmwareController struct{}

func NewFirmwareController() *FirmwareController {
	return &FirmwareController{}
}

func (fc *FirmwareController) Sync(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) (*v1.VirtualMachine, error) {
	firmware := vm.Spec.Template.Spec.Domain.Firmware
	if firmware == nil {
		firmware = &v1.Firmware{}
	}

	if firmware.UUID != "" {
		return vm, nil
	}

	vmCopy := vm.DeepCopy()
	if vmCopy.Spec.Template.Spec.Domain.Firmware == nil {
		vmCopy.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{}
	}
	vmCopy.Spec.Template.Spec.Domain.Firmware.UUID = CalculateLegacyUUID(vm.Name)

	return vmCopy, nil
}

const magicUUID = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareUUIDns = uuid.MustParse(magicUUID)

func CalculateLegacyUUID(name string) types.UID {
	return types.UID(uuid.NewSHA1(firmwareUUIDns, []byte(name)).String())
}
