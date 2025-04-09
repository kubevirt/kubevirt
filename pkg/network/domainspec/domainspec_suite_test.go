package domainspec

import (
	"testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/testutils"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func TestDomainSpec(t *testing.T) {
	testutils.KubeVirtTestSuiteSetup(t)
}

func newVMIMasqueradeInterface(namespace, vmiName string) *v1.VirtualMachineInstance {
	return libvmi.New(
		libvmi.WithNamespace(namespace),
		libvmi.WithName(vmiName),
		libvmi.WithNetwork(v1.DefaultPodNetwork()),
		libvmi.WithInterface(*v1.DefaultMasqueradeNetworkInterface()),
	)
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
