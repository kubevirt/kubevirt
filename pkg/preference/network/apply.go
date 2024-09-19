//nolint:lll
package network

import (
	"reflect"

	"k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/pointer"
)

type (
	Conflicts []*field.Path
	Applier   struct{}
)

func isInterfaceBindingUnset(iface *v1.Interface) bool {
	return reflect.ValueOf(iface.InterfaceBindingMethod).IsZero() && iface.Binding == nil
}

func isInterfaceOnPodNetwork(interfaceName string, vmiSpec v1.VirtualMachineInstanceSpec) bool {
	for _, network := range vmiSpec.Networks {
		if network.Name == interfaceName {
			return network.Pod != nil
		}
	}
	return false
}

func (n *Applier) Apply(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec v1.VirtualMachineInstanceSpec) *v1.VirtualMachineInstanceSpec {
	if preferenceSpec == nil {
		return &vmiSpec
	}

	vmiSpecCopy := vmiSpec.DeepCopy()

	if preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue != nil && vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue == nil {
		vmiSpecCopy.Domain.Devices.NetworkInterfaceMultiQueue = pointer.P(*preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue)
	}

	for ifaceIndex := range vmiSpecCopy.Domain.Devices.Interfaces {
		vmiIface := &vmiSpecCopy.Domain.Devices.Interfaces[ifaceIndex]
		if preferenceSpec.Devices.PreferredInterfaceModel != "" && vmiIface.Model == "" {
			vmiIface.Model = preferenceSpec.Devices.PreferredInterfaceModel
		}
		if preferenceSpec.Devices.PreferredInterfaceMasquerade != nil && isInterfaceBindingUnset(vmiIface) && isInterfaceOnPodNetwork(vmiIface.Name, vmiSpec) {
			vmiIface.Masquerade = preferenceSpec.Devices.PreferredInterfaceMasquerade.DeepCopy()
		}
	}
	return vmiSpecCopy
}
