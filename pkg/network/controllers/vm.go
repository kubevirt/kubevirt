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
	"reflect"

	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubevirt"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

type clusterConfigurer interface {
	LiveUpdateNADRefEnabled() bool
}

type VMController struct {
	clientset         kubevirt.Interface
	clusterConfigurer clusterConfigurer
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

func NewVMController(clientset kubevirt.Interface, clusterConfigurer clusterConfigurer) *VMController {
	return &VMController{
		clientset:         clientset,
		clusterConfigurer: clusterConfigurer,
	}
}

func (v *VMController) Sync(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) (*v1.VirtualMachine, error) {
	if vmi != nil && vmi.DeletionTimestamp != nil {
		return vm, nil
	}

	var (
		vmiIfacesByName        map[string]v1.Interface
		vmiIfaceStatusesByName map[string]v1.VirtualMachineInstanceNetworkInterface
	)

	if vmi != nil {
		vmiIfaceStatusesByName = vmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces,
			func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool { return true },
		)

		updatedVMI := syncVMIInterfaces(vm, vmi, vmiIfaceStatusesByName, v.clusterConfigurer.LiveUpdateNADRefEnabled())

		if err := v.vmiInterfacesPatch(&updatedVMI.Spec, vmi); err != nil {
			return vm, &syncError{
				fmt.Errorf("error encountered when trying to patch vmi: %v", err),
				hotPlugNetworkInterfaceErrorReason,
			}
		}

		vmiIfacesByName = vmispec.IndexInterfaceSpecByName(updatedVMI.Spec.Domain.Devices.Interfaces)
	}

	vmCopy := vm.DeepCopy()
	ifaces, networks := clearDetachedIfacesFromVM(
		vmCopy.Spec.Template.Spec.Domain.Devices.Interfaces,
		vmCopy.Spec.Template.Spec.Networks,
		vmiIfacesByName,
		vmiIfaceStatusesByName,
	)
	vmCopy.Spec.Template.Spec.Domain.Devices.Interfaces = ifaces
	vmCopy.Spec.Template.Spec.Networks = networks

	return vmCopy, nil
}

func syncVMIInterfaces(
	vm *v1.VirtualMachine,
	vmi *v1.VirtualMachineInstance,
	indexedStatusIfaces map[string]v1.VirtualMachineInstanceNetworkInterface,
	isLiveUpdateNADRefEnabled bool,
) *v1.VirtualMachineInstance {
	vmiCopy := vmi.DeepCopy()
	hasOrdinalIfaces := namescheme.HasOrdinalSecondaryIfaces(vmi.Spec.Networks, vmi.Status.Interfaces)
	updatedVmiSpec := applyDynamicIfaceRequestOnVMI(vm, vmiCopy, hasOrdinalIfaces, isLiveUpdateNADRefEnabled)
	vmiCopy.Spec = *updatedVmiSpec

	ifaces, networks := clearDetachedIfacesFromVMI(vmiCopy.Spec.Domain.Devices.Interfaces, vmiCopy.Spec.Networks, indexedStatusIfaces)
	vmiCopy.Spec.Domain.Devices.Interfaces = ifaces
	vmiCopy.Spec.Networks = networks

	return vmiCopy
}

func (v *VMController) vmiInterfacesPatch(newVmiSpec *v1.VirtualMachineInstanceSpec, vmi *v1.VirtualMachineInstance) error {
	if equality.Semantic.DeepEqual(vmi.Spec.Domain.Devices.Interfaces, newVmiSpec.Domain.Devices.Interfaces) &&
		equality.Semantic.DeepEqual(vmi.Spec.Networks, newVmiSpec.Networks) {
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

func applyDynamicIfaceRequestOnVMI(
	vm *v1.VirtualMachine,
	vmi *v1.VirtualMachineInstance,
	hasOrdinalIfaces bool,
	isLiveUpdateNADRefEnabled bool,
) *v1.VirtualMachineInstanceSpec {
	vmiSpecCopy := vmi.Spec.DeepCopy()
	vmiIndexedInterfaces := vmispec.IndexInterfaceSpecByName(vmiSpecCopy.Domain.Devices.Interfaces)
	vmiIndexedNetworks := vmispec.IndexNetworkSpecByName(vmiSpecCopy.Networks)
	vmIndexedNetworks := vmispec.IndexNetworkSpecByName(vm.Spec.Template.Spec.Networks)
	for _, vmIface := range vm.Spec.Template.Spec.Domain.Devices.Interfaces {
		vmiIfaceCopy, existsInVMISpec := vmiIndexedInterfaces[vmIface.Name]

		shouldHotplugIface := !existsInVMISpec &&
			vmIface.State != v1.InterfaceStateAbsent &&
			(vmIface.InterfaceBindingMethod.Bridge != nil || vmIface.InterfaceBindingMethod.SRIOV != nil)

		shouldUpdateExistingIfaceState := existsInVMISpec &&
			vmIface.State != vmiIfaceCopy.State &&
			vmiIfaceCopy.State != v1.InterfaceStateAbsent

		if isLiveUpdateNADRefEnabled {
			syncNetworkNADReference(vmiIndexedNetworks, vmIndexedNetworks, vmIface, vmiSpecCopy)
		}

		switch {
		case shouldHotplugIface:
			vmiSpecCopy.Networks = append(vmiSpecCopy.Networks, vmIndexedNetworks[vmIface.Name])
			vmiSpecCopy.Domain.Devices.Interfaces = append(vmiSpecCopy.Domain.Devices.Interfaces, vmIface)

		case shouldUpdateExistingIfaceState:
			if !(hasOrdinalIfaces && vmIface.State == v1.InterfaceStateAbsent) {
				vmiIface := vmispec.LookupInterfaceByName(vmiSpecCopy.Domain.Devices.Interfaces, vmIface.Name)
				vmiIface.State = vmIface.State
			}
		}
	}
	return vmiSpecCopy
}

func syncNetworkNADReference(
	vmiIndexedNetworks map[string]v1.Network,
	vmIndexedNetworks map[string]v1.Network,
	vmIface v1.Interface,
	vmiSpec *v1.VirtualMachineInstanceSpec,
) {
	vmNet, vmNetExists := vmIndexedNetworks[vmIface.Name]
	vmiNet, vmiNetExists := vmiIndexedNetworks[vmIface.Name]

	if vmiNetExists && vmNetExists && !reflect.DeepEqual(vmNet, vmiNet) {
		if areNormalizedNetsEqual(vmNet, vmiNet) {
			for i, n := range vmiSpec.Networks {
				if n.Name == vmNet.Name {
					vmiSpec.Networks[i] = *vmNet.DeepCopy()
					break
				}
			}
		}
	}
}

func areNormalizedNetsEqual(net1, net2 v1.Network) bool {
	normalizedNet1 := net1.DeepCopy()
	if normalizedNet1.Multus != nil {
		normalizedNet1.Multus.NetworkName = ""
	}

	normalizedNet2 := net2.DeepCopy()
	if normalizedNet2.Multus != nil {
		normalizedNet2.Multus.NetworkName = ""
	}

	return reflect.DeepEqual(normalizedNet1, normalizedNet2)
}

func clearDetachedIfacesFromVMI(
	specIfaces []v1.Interface,
	specNets []v1.Network,
	ifaceStatusesByName map[string]v1.VirtualMachineInstanceNetworkInterface,
) ([]v1.Interface, []v1.Network) {
	var retainedIfaces []v1.Interface

	for _, iface := range specIfaces {
		if _, existsInStatus := ifaceStatusesByName[iface.Name]; iface.State != v1.InterfaceStateAbsent || existsInStatus {
			retainedIfaces = append(retainedIfaces, iface)
		}
	}

	return retainedIfaces, vmispec.FilterNetworksByInterfaces(specNets, retainedIfaces)
}

func clearDetachedIfacesFromVM(
	specIfaces []v1.Interface,
	specNets []v1.Network,
	vmiIfacesByName map[string]v1.Interface,
	vmiIfaceStatusesByName map[string]v1.VirtualMachineInstanceNetworkInterface,
) ([]v1.Interface, []v1.Network) {
	var retainedIfaces []v1.Interface

	for _, iface := range specIfaces {
		_, existsInVMISpec := vmiIfacesByName[iface.Name]
		_, existsInVMIStatus := vmiIfaceStatusesByName[iface.Name]

		if iface.State != v1.InterfaceStateAbsent || existsInVMISpec || existsInVMIStatus {
			retainedIfaces = append(retainedIfaces, iface)
		}
	}

	return retainedIfaces, vmispec.FilterNetworksByInterfaces(specNets, retainedIfaces)
}
