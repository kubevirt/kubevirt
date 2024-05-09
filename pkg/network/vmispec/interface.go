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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package vmispec

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
)

func FilterSRIOVInterfaces(ifaces []v1.Interface) []v1.Interface {
	var sriovIfaces []v1.Interface
	for _, iface := range ifaces {
		if iface.SRIOV != nil {
			sriovIfaces = append(sriovIfaces, iface)
		}
	}
	return sriovIfaces
}

func SRIOVInterfaceExist(ifaces []v1.Interface) bool {
	for _, iface := range ifaces {
		if iface.SRIOV != nil {
			return true
		}
	}
	return false
}

func FilterInterfacesSpec(ifaces []v1.Interface, predicate func(i v1.Interface) bool) []v1.Interface {
	var filteredIfaces []v1.Interface
	for _, iface := range ifaces {
		if predicate(iface) {
			filteredIfaces = append(filteredIfaces, iface)
		}
	}
	return filteredIfaces
}

func VerifyVMIMigratable(vmi *v1.VirtualMachineInstance, bindingPlugins map[string]v1.InterfaceBindingPlugin) error {
	ifaces := vmi.Spec.Domain.Devices.Interfaces
	if len(ifaces) == 0 {
		return nil
	}

	_, allowPodBridgeNetworkLiveMigration := vmi.Annotations[v1.AllowPodBridgeNetworkLiveMigrationAnnotation]
	if allowPodBridgeNetworkLiveMigration && IsPodNetworkWithBridgeBindingInterface(vmi.Spec.Networks, ifaces) {
		return nil
	}
	if IsPodNetworkWithMasqueradeBindingInterface(vmi.Spec.Networks, ifaces) || IsPodNetworkWithMigratableBindingPlugin(vmi.Spec.Networks, ifaces, bindingPlugins) {
		return nil
	}

	return fmt.Errorf("cannot migrate VMI which does not use masquerade, bridge with %s VM annotation or a migratable plugin to connect to the pod network", v1.AllowPodBridgeNetworkLiveMigrationAnnotation)

}

func IsPodNetworkWithMigratableBindingPlugin(networks []v1.Network, ifaces []v1.Interface, bindingPlugins map[string]v1.InterfaceBindingPlugin) bool {
	if podNetwork := LookupPodNetwork(networks); podNetwork != nil {
		if podInterface := LookupInterfaceByName(ifaces, podNetwork.Name); podInterface != nil {
			if podInterface.Binding != nil {
				binding, exist := bindingPlugins[podInterface.Binding.Name]
				return exist && binding.Migration != nil
			}
		}
	}
	return false
}

func IsPodNetworkWithMasqueradeBindingInterface(networks []v1.Network, ifaces []v1.Interface) bool {
	if podNetwork := LookupPodNetwork(networks); podNetwork != nil {
		if podInterface := LookupInterfaceByName(ifaces, podNetwork.Name); podInterface != nil {
			return podInterface.Masquerade != nil
		}
	}
	return true
}

func IsPodNetworkWithBridgeBindingInterface(networks []v1.Network, ifaces []v1.Interface) bool {
	if podNetwork := LookupPodNetwork(networks); podNetwork != nil {
		if podInterface := LookupInterfaceByName(ifaces, podNetwork.Name); podInterface != nil {
			return podInterface.Bridge != nil
		}
	}
	return true
}

func PopInterfaceByNetwork(statusIfaces []v1.VirtualMachineInstanceNetworkInterface, network *v1.Network) (*v1.VirtualMachineInstanceNetworkInterface, []v1.VirtualMachineInstanceNetworkInterface) {
	if network == nil {
		return nil, statusIfaces
	}
	for index, currStatusIface := range statusIfaces {
		if currStatusIface.Name == network.Name {
			primaryIface := statusIfaces[index]
			statusIfaces = append(statusIfaces[:index], statusIfaces[index+1:]...)
			return &primaryIface, statusIfaces
		}
	}
	return nil, statusIfaces
}

func LookupInterfaceStatusByMac(interfaces []v1.VirtualMachineInstanceNetworkInterface, macAddress string) *v1.VirtualMachineInstanceNetworkInterface {
	for index := range interfaces {
		if interfaces[index].MAC == macAddress {
			return &interfaces[index]
		}
	}
	return nil
}

func LookupInterfaceStatusByName(interfaces []v1.VirtualMachineInstanceNetworkInterface, name string) *v1.VirtualMachineInstanceNetworkInterface {
	for index := range interfaces {
		if interfaces[index].Name == name {
			return &interfaces[index]
		}
	}
	return nil
}

func IndexInterfaceSpecByName(interfaces []v1.Interface) map[string]v1.Interface {
	ifacesByName := map[string]v1.Interface{}
	for _, ifaceSpec := range interfaces {
		ifacesByName[ifaceSpec.Name] = ifaceSpec
	}
	return ifacesByName
}

func LookupInterfaceByName(ifaces []v1.Interface, name string) *v1.Interface {
	for idx := range ifaces {
		if ifaces[idx].Name == name {
			return &ifaces[idx]
		}
	}
	return nil
}

func IndexInterfaceStatusByName(interfaces []v1.VirtualMachineInstanceNetworkInterface, p func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool) map[string]v1.VirtualMachineInstanceNetworkInterface {
	indexedInterfaceStatus := map[string]v1.VirtualMachineInstanceNetworkInterface{}
	for _, iface := range interfaces {
		if p == nil || p(iface) {
			indexedInterfaceStatus[iface.Name] = iface
		}
	}
	return indexedInterfaceStatus
}

func FilterInterfacesByNetworks(interfaces []v1.Interface, networks []v1.Network) []v1.Interface {
	var ifaces []v1.Interface
	ifacesByName := IndexInterfaceSpecByName(interfaces)
	for _, net := range networks {
		if iface, exists := ifacesByName[net.Name]; exists {
			ifaces = append(ifaces, iface)
		}
	}
	return ifaces
}

func BindingPluginNetworkWithDeviceInfoExist(ifaces []v1.Interface, bindingPlugins map[string]v1.InterfaceBindingPlugin) bool {
	for _, iface := range ifaces {
		if HasBindingPluginDeviceInfo(iface, bindingPlugins) {
			return true
		}
	}
	return false
}

func HasBindingPluginDeviceInfo(iface v1.Interface, bindingPlugins map[string]v1.InterfaceBindingPlugin) bool {
	if iface.Binding != nil {
		binding, exist := bindingPlugins[iface.Binding.Name]
		return exist && binding.DownwardAPI == v1.DeviceInfo
	}
	return false
}
