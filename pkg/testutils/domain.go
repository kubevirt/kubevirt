package testutils

import (
	"strings"

	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const virtioTrans = "virtio-transitional"

func ExpectVirtioTransitionalOnly(dom *api.DomainSpec) {
	hit := false
	for _, disk := range dom.Devices.Disks {
		if disk.Target.Bus == v1.DiskBusVirtio {
			ExpectWithOffset(1, disk.Model).To(Equal(virtioTrans))
			hit = true
		}
	}
	ExpectWithOffset(1, hit).To(BeTrue())

	hit = false
	for _, ifc := range dom.Devices.Interfaces {
		if strings.HasPrefix(ifc.Model.Type, v1.VirtIO) {
			ExpectWithOffset(1, ifc.Model.Type).To(Equal(virtioTrans))
			hit = true
		}
	}
	ExpectWithOffset(1, hit).To(BeTrue())

	hit = false
	for _, input := range dom.Devices.Inputs {
		if strings.HasPrefix(input.Model, v1.VirtIO) {
			// All our input types only exist only as virtio 1.0 and only accept virtio
			ExpectWithOffset(1, input.Model).To(Equal(v1.VirtIO))
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
