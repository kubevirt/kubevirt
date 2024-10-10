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
 * Copyright The KubeVirt Authors
 *
 */
package apply

import (
	"reflect"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

func isInterfaceBindingUnset(iface *virtv1.Interface) bool {
	return reflect.ValueOf(iface.InterfaceBindingMethod).IsZero() && iface.Binding == nil
}

func isInterfaceOnPodNetwork(interfaceName string, vmiSpec *virtv1.VirtualMachineInstanceSpec) bool {
	for _, network := range vmiSpec.Networks {
		if network.Name == interfaceName {
			return network.Pod != nil
		}
	}
	return false
}

func applyInterfacePreferences(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for ifaceIndex := range vmiSpec.Domain.Devices.Interfaces {
		vmiIface := &vmiSpec.Domain.Devices.Interfaces[ifaceIndex]
		if preferenceSpec.Devices.PreferredInterfaceModel != "" && vmiIface.Model == "" {
			vmiIface.Model = preferenceSpec.Devices.PreferredInterfaceModel
		}
		if preferenceSpec.Devices.PreferredInterfaceMasquerade != nil &&
			isInterfaceBindingUnset(vmiIface) &&
			isInterfaceOnPodNetwork(vmiIface.Name, vmiSpec) {
			vmiIface.Masquerade = preferenceSpec.Devices.PreferredInterfaceMasquerade.DeepCopy()
		}
	}
}
