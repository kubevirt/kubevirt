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
	"context"
	"fmt"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubevirt"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/common"
)

type FirmwareController struct {
	clientset kubevirt.Interface
}

const (
	firmwareUUIDErrorReason = "FirmwareUUIDError"
)

func NewFirmwareController(clientset kubevirt.Interface) *FirmwareController {
	return &FirmwareController{
		clientset: clientset,
	}
}

func (fc *FirmwareController) Sync(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) (*v1.VirtualMachine, error) {
	firmware := vm.Spec.Template.Spec.Domain.Firmware
	if firmware == nil {
		firmware = &v1.Firmware{}
	}

	if firmware.UUID != "" {
		return vm, nil
	}

	firmware = firmware.DeepCopy()
	firmware.UUID = CalculateLegacyUUID(vm.Name)

	updatedVM, err := fc.vmFirmwarePatch(firmware, vm)
	if err != nil {
		return vm, common.NewSyncError(fmt.Errorf("error encountered when trying to patch VM firmware: %w", err), firmwareUUIDErrorReason)
	}

	return updatedVM, nil
}

func (fc *FirmwareController) vmFirmwarePatch(updatedFirmware *v1.Firmware, vm *v1.VirtualMachine) (*v1.VirtualMachine, error) {
	patchBytes, err := patch.New(
		patch.WithTest("/spec/template/spec/domain/firmware", vm.Spec.Template.Spec.Domain.Firmware),
		patch.WithAdd("/spec/template/spec/domain/firmware", updatedFirmware),
	).GeneratePayload()
	if err != nil {
		return vm, err
	}

	return fc.clientset.KubevirtV1().
		VirtualMachines(vm.Namespace).
		Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
}

const magicUUID = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareUUIDns = uuid.MustParse(magicUUID)

func CalculateLegacyUUID(name string) types.UID {
	return types.UID(uuid.NewSHA1(firmwareUUIDns, []byte(name)).String())
}
