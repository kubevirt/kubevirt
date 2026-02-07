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

package vmliveupdate

import (
	"reflect"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

// IsRestartRequired - Checks if the changes in network related fields require a reset of the VM
// in order for them to be applied
func IsRestartRequired(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) bool {
	desiredIfaces := vm.Spec.Template.Spec.Domain.Devices.Interfaces
	currentIfaces := vmi.Spec.Domain.Devices.Interfaces

	desiredNets := vm.Spec.Template.Spec.Networks
	currentNets := vmi.Spec.Networks

	return shouldIfacesChangeRequireRestart(desiredIfaces, currentIfaces) ||
		shouldNetsChangeRequireRestart(desiredNets, currentNets)
}

func shouldIfacesChangeRequireRestart(desiredIfaces, currentIfaces []v1.Interface) bool {
	desiredIfacesByName := vmispec.IndexInterfaceSpecByName(desiredIfaces)
	currentIfacesByName := vmispec.IndexInterfaceSpecByName(currentIfaces)

	return haveCurrentIfacesBeenRemoved(desiredIfacesByName, currentIfacesByName) ||
		haveCurrentIfacesChanged(desiredIfacesByName, currentIfacesByName)
}

func shouldNetsChangeRequireRestart(desiredNets, currentNets []v1.Network) bool {
	isPodNetworkInDesiredNets := vmispec.LookupPodNetwork(desiredNets) != nil
	isPodNetworkInCurrentNets := vmispec.LookupPodNetwork(currentNets) != nil

	if isPodNetworkInDesiredNets && !isPodNetworkInCurrentNets {
		return true
	}

	desiredNetsByName := vmispec.IndexNetworkSpecByName(desiredNets)
	currentNetsByName := vmispec.IndexNetworkSpecByName(currentNets)

	return haveCurrentNetsBeenRemoved(desiredNetsByName, currentNetsByName) ||
		haveCurrentNetsChanged(desiredNetsByName, currentNetsByName)
}

// haveCurrentIfacesBeenRemoved checks if interfaces existing in the VMI spec were removed
// from the VM spec without using the hotunplug flow.
func haveCurrentIfacesBeenRemoved(desiredIfacesByName, currentIfacesByName map[string]v1.Interface) bool {
	for currentIfaceName := range currentIfacesByName {
		if _, desiredIfaceExists := desiredIfacesByName[currentIfaceName]; !desiredIfaceExists {
			return true
		}
	}

	return false
}

func haveCurrentIfacesChanged(desiredIfacesByName, currentIfacesByName map[string]v1.Interface) bool {
	for currentIfaceName, currentIface := range currentIfacesByName {
		desiredIface := desiredIfacesByName[currentIfaceName]

		if !areNormalizedIfacesEqual(desiredIface, currentIface) {
			return true
		}
	}

	return false
}

func areNormalizedIfacesEqual(iface1, iface2 v1.Interface) bool {
	normalizedIface1 := iface1.DeepCopy()
	normalizedIface1.State = ""

	normalizedIface2 := iface2.DeepCopy()
	normalizedIface2.State = ""

	return reflect.DeepEqual(normalizedIface1, normalizedIface2)
}

// haveCurrentNetsBeenRemoved checks if networks existing in the VMI spec were removed
// from the VM spec without using the hotunplug flow.
func haveCurrentNetsBeenRemoved(desiredNetsByName, currentNetsByName map[string]v1.Network) bool {
	for currentNetName := range currentNetsByName {
		if _, desiredNetExists := desiredNetsByName[currentNetName]; !desiredNetExists {
			return true
		}
	}

	return false
}

func haveCurrentNetsChanged(desiredNetsByName, currentNetsByName map[string]v1.Network) bool {
	for currentNetName, currentNet := range currentNetsByName {
		desiredNet := desiredNetsByName[currentNetName]

		if !reflect.DeepEqual(desiredNet, currentNet) {
			return true
		}
	}

	return false
}
