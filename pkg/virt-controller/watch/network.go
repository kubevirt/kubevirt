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

package watch

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func calculateDynamicInterfaces(vmi *v1.VirtualMachineInstance) ([]v1.Interface, []v1.Network, bool) {
	vmiSpecIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	ifacesToHotUnplugExist := len(vmi.Spec.Domain.Devices.Interfaces) > len(vmiSpecIfaces)

	vmiSpecNets := vmi.Spec.Networks
	if ifacesToHotUnplugExist {
		vmiSpecNets = vmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, vmiSpecIfaces)
	}
	ifacesToHotplugExist := len(vmispec.NetworksToHotplug(vmiSpecNets, vmi.Status.Interfaces)) > 0

	isIfaceChangeRequired := ifacesToHotplugExist || ifacesToHotUnplugExist
	if !isIfaceChangeRequired {
		return nil, nil, false
	}
	return vmiSpecIfaces, vmiSpecNets, isIfaceChangeRequired
}

func trimDoneInterfaceRequests(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) {
	if len(vm.Status.InterfaceRequests) == 0 {
		return
	}

	vmiExist := vmi != nil
	var vmiIndexedInterfaces map[string]v1.Interface
	if vmiExist {
		vmiIndexedInterfaces = vmispec.IndexInterfaceSpecByName(vmi.Spec.Domain.Devices.Interfaces)
	}

	vmIndexedInterfaces := vmispec.IndexInterfaceSpecByName(vm.Spec.Template.Spec.Domain.Devices.Interfaces)
	updateIfaceRequests := make([]v1.VirtualMachineInterfaceRequest, 0)
	for _, request := range vm.Status.InterfaceRequests {
		removeRequest := false
		switch {
		case request.AddInterfaceOptions != nil:
			ifaceName := request.AddInterfaceOptions.Name
			_, existsInVMTemplate := vmIndexedInterfaces[ifaceName]

			if vmiExist {
				_, existsInVMISpec := vmiIndexedInterfaces[ifaceName]
				removeRequest = existsInVMTemplate && existsInVMISpec
			} else {
				removeRequest = existsInVMTemplate
			}
		case request.RemoveInterfaceOptions != nil:
			ifaceName := request.RemoveInterfaceOptions.Name
			vmIface, existsInVMTemplate := vmIndexedInterfaces[ifaceName]
			absentIfaceInVMTemplate := existsInVMTemplate && vmIface.State == v1.InterfaceStateAbsent

			if vmiExist {
				vmiIface, existsInVMISpec := vmiIndexedInterfaces[ifaceName]
				absentIfaceInVMISpec := existsInVMISpec && vmiIface.State == v1.InterfaceStateAbsent
				removeRequest = absentIfaceInVMTemplate && absentIfaceInVMISpec
			} else {
				removeRequest = absentIfaceInVMTemplate
			}
		}

		if !removeRequest {
			updateIfaceRequests = append(updateIfaceRequests, request)
		}
	}
	vm.Status.InterfaceRequests = updateIfaceRequests
}
