package hypervisor

import (
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

func MakeDomainBuilder(hypervisor string, vmi *v1.VirtualMachineInstance, c *types.ConverterContext) *types.DomainBuilder {
	return nil
}
