package rest

import (
	"kubevirt.io/kubevirt/pkg/api/v1"
	vmconfigv1 "kubevirt.io/kubevirt/pkg/virt-vmconfig/api/v1"
)

// ConfigToSpec is a function that returns the constructed VMSpec object from within VMConfig object.
func ConfigToSpec(vmconfig *vmconfigv1.VMConfig) *v1.DomainSpec {
	// TODO: Add the actual VMConfig transformation logic here.
	return vmconfig.Spec.Template.Spec
}
