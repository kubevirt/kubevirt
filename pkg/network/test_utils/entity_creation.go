package test_utils

import (
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func NewVMI(namespace, name string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMIWithNS(namespace, name)
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}

func NewVMIBridgeInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := NewVMI(namespace, name)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func NewVMIMasqueradeInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := NewVMI(namespace, name)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func NewVMISlirpInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := NewVMI(namespace, name)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultSlirpNetworkInterface()}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func NewVMIMacvtapInterface(namespace string, vmiName string, ifaceName string) *v1.VirtualMachineInstance {
	vmi := NewVMI(namespace, vmiName)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMacvtapNetworkInterface(ifaceName)}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func NewDomainWithBridgeInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{
			Type: "virtio",
		},
		Type: "bridge",
		Source: api.InterfaceSource{
			Bridge: api.DefaultBridgeName,
		},
		Alias: api.NewUserDefinedAlias("default"),
	},
	}
	return domain
}

func NewDomainWithSlirpInterface() *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Model: &api.Model{
			Type: "e1000",
		},
		Type:  "user",
		Alias: api.NewUserDefinedAlias("default"),
	},
	}

	// Create network interface
	if domain.Spec.QEMUCmd == nil {
		domain.Spec.QEMUCmd = &api.Commandline{}
	}

	if domain.Spec.QEMUCmd.QEMUArg == nil {
		domain.Spec.QEMUCmd.QEMUArg = make([]api.Arg, 0)
	}

	return domain
}

func NewDomainWithMacvtapInterface(macvtapName string) *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Alias: api.NewUserDefinedAlias(macvtapName),
		Model: &api.Model{
			Type: "virtio",
		},
		Type: "ethernet",
	}}
	return domain
}
