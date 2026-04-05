/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
