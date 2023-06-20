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
	"encoding/json"
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

func calculateDynamicInterfaces(vmi *v1.VirtualMachineInstance, hasOrdinalIfaces bool) ([]v1.Interface, []v1.Network, bool) {
	vmiSpecIfaces := vmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	ifacesToHotUnplugExist := len(vmi.Spec.Domain.Devices.Interfaces) > len(vmiSpecIfaces)

	if ifacesToHotUnplugExist && hasOrdinalIfaces {
		vmiSpecIfaces = vmi.Spec.Domain.Devices.Interfaces
		ifacesToHotUnplugExist = false
		log.Log.Object(vmi).Error("hot-unplug is not supported on old VMIs with ordered pod interface names")
	}
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

func trimDoneInterfaceRequests(vm *v1.VirtualMachine) {
	if len(vm.Status.InterfaceRequests) == 0 {
		return
	}

	indexedInterfaces := vmispec.IndexInterfaceSpecByName(vm.Spec.Template.Spec.Domain.Devices.Interfaces)
	updateIfaceRequests := make([]v1.VirtualMachineInterfaceRequest, 0)
	for _, request := range vm.Status.InterfaceRequests {

		var ifaceName string

		removeRequest := false

		switch {
		case request.AddInterfaceOptions != nil:
			ifaceName = request.AddInterfaceOptions.Name
			if _, exists := indexedInterfaces[ifaceName]; exists {
				removeRequest = true
			}
		case request.RemoveInterfaceOptions != nil:
			ifaceName = request.RemoveInterfaceOptions.Name
			if iface, exists := indexedInterfaces[ifaceName]; exists &&
				iface.State == v1.InterfaceStateAbsent {
				removeRequest = true
			}
		}

		if !removeRequest {
			updateIfaceRequests = append(updateIfaceRequests, request)
		}
	}
	vm.Status.InterfaceRequests = updateIfaceRequests
}

func createPatchToCancelInterfacesRemoval(interfaces []v1.Interface) ([]byte, error) {
	absentIfaces := vmispec.FilterInterfacesSpec(interfaces, func(iface v1.Interface) bool {
		return iface.State == v1.InterfaceStateAbsent
	})
	if len(absentIfaces) == 0 {
		return nil, nil
	}

	var updatedInterfaces []v1.Interface
	for _, iface := range interfaces {
		iface.State = ""
		updatedInterfaces = append(updatedInterfaces, iface)
	}

	oldIfacesJSON, err := json.Marshal(interfaces)
	if err != nil {
		return nil, err
	}
	newIfacesJSON, err := json.Marshal(updatedInterfaces)
	if err != nil {
		return nil, err
	}

	const verb = "add"
	testInterfaces := fmt.Sprintf(`{ "op": "test", "path": "/spec/domain/devices/interfaces", "value": %s}`, string(oldIfacesJSON))
	updateInterfaces := fmt.Sprintf(`{ "op": %q, "path": "/spec/domain/devices/interfaces", "value": %s}`, verb, string(newIfacesJSON))

	patch := fmt.Sprintf("[%s, %s]", testInterfaces, updateInterfaces)
	return []byte(patch), nil
}
