package domainspec

import (
	"testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-api/libvmi"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func TestDomainSpec(t *testing.T) {
	libvmi.KubeVirtTestSuiteSetup(t)
}

func newVMI(namespace, name string) *v1.VirtualMachineInstance {
	vmi := libvmi.New(libvmi.WithName(name), libvmi.WithNamespace(namespace))
	vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
	return vmi
}

func newVMIMasqueradeInterface(namespace, vmiName string) *v1.VirtualMachineInstance {
	vmi := newVMI(namespace, vmiName)
	vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultMasqueradeNetworkInterface()}
	libvmi.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func NewDomainInterface(name string) *api.Domain {
	domain := &api.Domain{}
	domain.Spec.Devices.Interfaces = []api.Interface{{
		Alias: api.NewUserDefinedAlias(name),
		Model: &api.Model{
			Type: v1.VirtIO,
		},
		Type: "ethernet",
	}}
	return domain
}
