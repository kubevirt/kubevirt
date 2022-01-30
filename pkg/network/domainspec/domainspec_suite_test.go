package domainspec

import (
	"testing"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"
	"kubevirt.io/client-go/testutils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func TestDomainSpec(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

func newVMI(namespace, name string) *v1.VirtualMachineInstance {
	vmi := api2.NewMinimalVMIWithNS(namespace, name)
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}

func newVMISlirpInterface(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := newVMI(namespace, name)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultSlirpNetworkInterface()}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func newVMIMacvtapInterface(namespace string, vmiName string, ifaceName string) *v1.VirtualMachineInstance {
	vmi := newVMI(namespace, vmiName)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMacvtapNetworkInterface(ifaceName)}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
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
