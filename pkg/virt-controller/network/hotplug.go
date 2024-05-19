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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package network

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-controller/ipamclaims/libipam"

	k8scnicncfiov1 "kubevirt.io/client-go/generated/network-attachment-definition-client/clientset/versioned/typed/k8s.cni.cncf.io/v1"
)

func CalculateInterfacesAndNetworksForMultusAnnotationUpdate(vmi *v1.VirtualMachineInstance) ([]v1.Interface, []v1.Network, bool) {
	vmiNonAbsentSpecIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	ifacesToHotUnplugExist := len(vmi.Spec.Domain.Devices.Interfaces) > len(vmiNonAbsentSpecIfaces)

	ifacesStatusByName := vmispec.IndexInterfaceStatusByName(vmi.Status.Interfaces, nil)
	ifacesToAnnotate := vmispec.FilterInterfacesSpec(vmiNonAbsentSpecIfaces, func(iface v1.Interface) bool {
		_, ifaceInStatus := ifacesStatusByName[iface.Name]
		sriovIfaceNotPlugged := iface.SRIOV != nil && !ifaceInStatus
		return !sriovIfaceNotPlugged
	})

	networksToAnnotate := vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, ifacesToAnnotate)

	ifacesToHotplug := vmispec.FilterInterfacesSpec(ifacesToAnnotate, func(iface v1.Interface) bool {
		_, inStatus := ifacesStatusByName[iface.Name]
		return !inStatus
	})
	ifacesToHotplugExist := len(ifacesToHotplug) > 0

	isIfaceChangeRequired := ifacesToHotplugExist || ifacesToHotUnplugExist
	if !isIfaceChangeRequired {
		return nil, nil, false
	}
	return ifacesToAnnotate, networksToAnnotate, isIfaceChangeRequired
}

func ApplyDynamicIfaceRequestOnVMI(networkClient k8scnicncfiov1.K8sCniCncfIoV1Interface, vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance, hasOrdinalIfaces bool) (*v1.VirtualMachineInstanceSpec, error) {
	vmiSpecCopy := vmi.Spec.DeepCopy()
	vmiIndexedInterfaces := vmispec.IndexInterfaceSpecByName(vmiSpecCopy.Domain.Devices.Interfaces)
	vmIndexedNetworks := vmispec.IndexNetworkSpecByName(vm.Spec.Template.Spec.Networks)
	for _, vmIface := range vm.Spec.Template.Spec.Domain.Devices.Interfaces {
		_, existsInVMISpec := vmiIndexedInterfaces[vmIface.Name]
		shouldBeHotPlug := !existsInVMISpec && vmIface.State != v1.InterfaceStateAbsent && (vmIface.InterfaceBindingMethod.Bridge != nil || vmIface.InterfaceBindingMethod.SRIOV != nil)
		shouldBeHotUnplug := !hasOrdinalIfaces && existsInVMISpec && vmIface.State == v1.InterfaceStateAbsent
		if shouldBeHotPlug || shouldBeHotUnplug {
			if persistentIP, err := isPersistentIPNetwork(networkClient, vm.Namespace, vmIndexedNetworks[vmIface.Name]); err != nil {
				return nil, err
			} else if persistentIP {
				log.Log.Errorf("Skipping hot{un}plug of persistentIP interface %s, not supported yet", vmIface.Name)
				continue
			}
		}

		if shouldBeHotPlug {
			vmiSpecCopy.Networks = append(vmiSpecCopy.Networks, vmIndexedNetworks[vmIface.Name])
			vmiSpecCopy.Domain.Devices.Interfaces = append(vmiSpecCopy.Domain.Devices.Interfaces, vmIface)
		}
		if shouldBeHotUnplug {
			vmiIface := vmispec.LookupInterfaceByName(vmiSpecCopy.Domain.Devices.Interfaces, vmIface.Name)
			vmiIface.State = v1.InterfaceStateAbsent
		}
	}
	return vmiSpecCopy, nil
}

func ClearDetachedInterfaces(specIfaces []v1.Interface, specNets []v1.Network, statusIfaces map[string]v1.VirtualMachineInstanceNetworkInterface) ([]v1.Interface, []v1.Network) {
	var ifaces []v1.Interface
	for _, iface := range specIfaces {
		if _, existInStatus := statusIfaces[iface.Name]; (existInStatus && iface.State == v1.InterfaceStateAbsent) ||
			iface.State != v1.InterfaceStateAbsent {
			ifaces = append(ifaces, iface)
		}
	}

	return ifaces, vmispec.FilterNetworksByInterfaces(specNets, ifaces)
}

func isPersistentIPNetwork(networkClient k8scnicncfiov1.K8sCniCncfIoV1Interface, namespace string, network v1.Network) (bool, error) {
	if !vmispec.IsSecondaryMultusNetwork(network) {
		return false, nil
	}

	nads, err := GetNetworkAttachmentDefinitionByName(networkClient, namespace, []v1.Network{network})
	if err != nil {
		return false, fmt.Errorf("failed retrieving network attachment definitions: %w", err)
	}

	conf, err := libipam.GetPersistentIPsConf(nads[network.Name])
	if err != nil {
		return false, fmt.Errorf("failed retrieving netConf: %w", err)
	}

	if conf.AllowPersistentIPs {
		return true, nil
	}

	return false, nil
}
