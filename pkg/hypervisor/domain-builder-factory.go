package hypervisor

import (
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
)

type DomainBuilderFactory interface {
	MakeDomainBuilder(c *types.ConverterContext) *types.DomainBuilder
}
