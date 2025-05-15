/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
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
 * Copyright The KubeVirt Authors.
 *
 */

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubevirt"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type VMController struct {
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
	hotPlugNetworkInterfaceErrorReason = "HotPlugNetworkInterfaceError"
)

func NewVMController(clientset kubevirt.Interface) *VMController {
	return &VMController{
		clientset: clientset,
	}
}

func (v *VMController) Sync(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) (*v1.VirtualMachine, error) {
	vmCopy := vm.DeepCopy()

	if vmi == nil || vmi.DeletionTimestamp != nil {
		filterVMInterfacesAndNetworks(vmCopy, map[string]v1.VirtualMachineInstanceNetworkInterface{})
		return vmCopy, nil
	}

	indexedStatusIfaces := vmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces,
		func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool { return true },
	)
	filterVMInterfacesAndNetworks(vmCopy, indexedStatusIfaces)

	if err := v.patchVMIWithUpdatedInterfaces(vmCopy, vmi, indexedStatusIfaces); err != nil {
		return vm, err
	}

	return vmCopy, nil
}

func filterVMInterfacesAndNetworks(vm *v1.VirtualMachine, statusIfaces map[string]v1.VirtualMachineInstanceNetworkInterface) {
	ifaces, networks := ClearDetachedInterfaces(
		vm.Spec.Template.Spec.Domain.Devices.Interfaces,
		vm.Spec.Template.Spec.Networks,
		statusIfaces,
	)
	vm.Spec.Template.Spec.Domain.Devices.Interfaces = ifaces
	vm.Spec.Template.Spec.Networks = networks
}

func (v *VMController) patchVMIWithUpdatedInterfaces(
	vm *v1.VirtualMachine,
	vmi *v1.VirtualMachineInstance,
	indexedStatusIfaces map[string]v1.VirtualMachineInstanceNetworkInterface,
) error {
	vmiCopy := vmi.DeepCopy()

	ifaces, networks := ClearDetachedInterfaces(vmiCopy.Spec.Domain.Devices.Interfaces, vmiCopy.Spec.Networks, indexedStatusIfaces)
	vmiCopy.Spec.Domain.Devices.Interfaces = ifaces
	vmiCopy.Spec.Networks = networks

	hasOrdinalIfaces := namescheme.HasOrdinalSecondaryIfaces(vmi.Spec.Networks, vmi.Status.Interfaces)
	updatedVmiSpec := ApplyDynamicIfaceRequestOnVMI(vm, vmiCopy, hasOrdinalIfaces)
	vmiCopy.Spec = *updatedVmiSpec

	if err := v.vmiInterfacesPatch(&vmiCopy.Spec, vmi); err != nil {
		return &syncError{
			fmt.Errorf("error encountered when trying to patch vmi: %v", err),
			hotPlugNetworkInterfaceErrorReason,
		}
	}

	return nil
}

func (v *VMController) vmiInterfacesPatch(newVmiSpec *v1.VirtualMachineInstanceSpec, vmi *v1.VirtualMachineInstance) error {
	if equality.Semantic.DeepEqual(vmi.Spec.Domain.Devices.Interfaces, newVmiSpec.Domain.Devices.Interfaces) {
		return nil
	}
	patchBytes, err := patch.New(
		patch.WithTest("/spec/networks", vmi.Spec.Networks),
		patch.WithAdd("/spec/networks", newVmiSpec.Networks),
		patch.WithTest("/spec/domain/devices/interfaces", vmi.Spec.Domain.Devices.Interfaces),
		patch.WithAdd("/spec/domain/devices/interfaces", newVmiSpec.Domain.Devices.Interfaces),
	).GeneratePayload()
	if err != nil {
		return err
	}
	_, err = v.clientset.KubevirtV1().
		VirtualMachineInstances(vmi.Namespace).
		Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})

	return err
}
