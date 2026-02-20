package hypervisor

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor/kvm"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

type DomainBuilderFactory interface {
	MakeDomainBuilder(vmi *v1.VirtualMachineInstance, c *types.ConverterContext) *types.DomainBuilder
}

func NewDomainBuilderFactory(hypervisor string) DomainBuilderFactory {
	switch hypervisor {
	// Other hypervisors can be added here
	default:
		return &kvm.KvmDomainBuilderFactory{} // Default to KVM
	}
}
