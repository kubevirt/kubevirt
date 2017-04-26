package virt_manifest

import (
	"k8s.io/client-go/pkg/util/rand"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

func AddMinimalVMSpec(vm *v1.VM) {
	// Make sure the domain name matches the VM name
	if vm.Spec.Domain == nil {
		vm.Spec.Domain = new(v1.DomainSpec)
	}
	vm.Spec.Domain.Name = vm.ObjectMeta.Name + "-" + rand.String(5)

	AddMinimalDomainSpec(vm.Spec.Domain)
}

func AddMinimalDomainSpec(dom *v1.DomainSpec) {
	for idx, graphics := range dom.Devices.Graphics {
		if graphics.Type == "spice" {
			if graphics.Listen.Type == "" {
				dom.Devices.Graphics[idx].Listen.Type = "address"
			}
			if ((graphics.Listen.Type == "address") ||
				(graphics.Listen.Type == "")) &&
				(graphics.Listen.Address == "") {
				dom.Devices.Graphics[idx].Listen.Address = "0.0.0.0"
			}
		}
	}
}
