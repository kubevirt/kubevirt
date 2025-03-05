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
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubevirt"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

type FirmwareController struct {
	clientset kubevirt.Interface
}

type syncError struct {
	err    error
	reason string
}

func (e *syncError) Error() string {
	return e.err.Error()
}

func (e *syncError) Reason() string {
	return e.reason
}

func (e *syncError) RequiresRequeue() bool {
	return true
}

const (
	FirmwareUUIDErrorReason = "FirmwareUUIDError"
)

func NewFirmwareController(clientset kubevirt.Interface) *FirmwareController {
	return &FirmwareController{
		clientset: clientset,
	}
}

func (fc *FirmwareController) Sync(vm *v1.VirtualMachine, _ *v1.VirtualMachineInstance) (*v1.VirtualMachine, error) {
	vmCopy := vm.DeepCopy()
	legacyUUID := CalculateLegacyUUID(vmCopy.Name)

	if vmCopy.Spec.Template.Spec.Domain.Firmware == nil {
		vmCopy.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{}
	}
	if vmCopy.Spec.Template.Spec.Domain.Firmware.UUID == "" {
		vmCopy.Spec.Template.Spec.Domain.Firmware.UUID = legacyUUID
	}

	if err := fc.vmFirmwarePatch(vmCopy.Spec.Template.Spec.Domain.Firmware, vm); err != nil {
		return vm, &syncError{
			err:    fmt.Errorf("error encountered when trying to patch VM firmware: %w", err),
			reason: FirmwareUUIDErrorReason,
		}
	}

	return vmCopy, nil
}

func (fc *FirmwareController) vmFirmwarePatch(updatedFirmware *v1.Firmware, vm *v1.VirtualMachine) error {
	if equality.Semantic.DeepEqual(vm.Spec.Template.Spec.Domain.Firmware, updatedFirmware) {
		return nil
	}

	patchBytes, err := patch.New(
		patch.WithTest("/spec/template/spec/domain/firmware", vm.Spec.Template.Spec.Domain.Firmware),
		patch.WithAdd("/spec/template/spec/domain/firmware", updatedFirmware),
	).GeneratePayload()
	if err != nil {
		return err
	}

	_, err = fc.clientset.KubevirtV1().
		VirtualMachines(vm.Namespace).
		Patch(context.Background(), vm.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}

const magicUUID = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareUUIDns = uuid.MustParse(magicUUID)

func CalculateLegacyUUID(name string) types.UID {
	return types.UID(uuid.NewSHA1(firmwareUUIDns, []byte(name)).String())
}
