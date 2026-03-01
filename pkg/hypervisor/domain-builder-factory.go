package hypervisor

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor/kvm"
	converter_types "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

func MakeDomainBuilder(hypervisor string, vmi *v1.VirtualMachineInstance, c *converter_types.ConverterContext) *converter_types.DomainBuilder {
	switch hypervisor {
	// Other hypervisors can be added here
	default:
		return kvm.MakeDomainBuilder(vmi, c)
	}
}
