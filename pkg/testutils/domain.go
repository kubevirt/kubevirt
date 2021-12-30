package testutils

import (
	"strings"

	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const virtioTrans = "virtio-transitional"

func ExpectVirtioTransitionalOnly(dom *api.DomainSpec) {
	hit := false
	for _, disk := range dom.Devices.Disks {
		if disk.Target.Bus == "virtio" {
			ExpectWithOffset(1, disk.Model).To(Equal(virtioTrans))
			hit = true
		}
	}
	ExpectWithOffset(1, hit).To(BeTrue())

	hit = false
	for _, ifc := range dom.Devices.Interfaces {
		if strings.HasPrefix(ifc.Model.Type, "virtio") {
			ExpectWithOffset(1, ifc.Model.Type).To(Equal(virtioTrans))
			hit = true
		}
	}
	ExpectWithOffset(1, hit).To(BeTrue())

	hit = false
	for _, input := range dom.Devices.Inputs {
		if strings.HasPrefix(input.Model, "virtio") {
			// All our input types only exist only as virtio 1.0 and only accept virtio
			ExpectWithOffset(1, input.Model).To(Equal("virtio"))
			hit = true
		}
	}
	ExpectWithOffset(1, hit).To(BeTrue())

	hitCount := 0
	for _, controller := range dom.Devices.Controllers {
		if controller.Type == "virtio-serial" {
			ExpectWithOffset(1, controller.Model).To(Equal(virtioTrans))
			hitCount++
		}
		if controller.Type == "scsi" {
			ExpectWithOffset(1, controller.Model).To(Equal(virtioTrans))
			hitCount++
		}
	}
	ExpectWithOffset(1, hitCount).To(BeNumerically("==", 2))

	ExpectWithOffset(1, dom.Devices.Rng.Model).To(Equal(virtioTrans))
	ExpectWithOffset(1, dom.Devices.Ballooning.Model).To(Equal(virtioTrans))
}
