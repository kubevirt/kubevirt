package rest

import (
	"kubevirt.io/kubevirt/pkg/api/v1"
	vmconfigv1 "kubevirt.io/kubevirt/pkg/virt-vmconfig/api/v1"
)

// ConfigToSpec is a function that returns the constructed VMSpec object from within VMConfig object.
func ConfigToSpec(vmconfig *vmconfigv1.VMConfig) *v1.DomainSpec {
	domainSpec := vmconfig.Spec.Template.Spec

	// TODO: Add the actual VMConfig transformation logic here.
	// This is just an example of how the conversion can be done.
	if vmconfig.Spec.Template.Features.OS == "other" {
		domainSpec.OS = v1.OS{Type: v1.OSType{OS: "hvm"}}
	}

	return domainSpec
}
