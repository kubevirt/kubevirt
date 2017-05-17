package virt_manifest

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Mapper", func() {
	Context("Map PersistentVolumeClaims", func() {
		var dom *v1.DomainSpec
		BeforeEach(func() {
			dom = new(v1.DomainSpec)
			dom.Devices.Disks = []v1.Disk{
				v1.Disk{Type: Type_Network,
					Source: v1.DiskSource{Name: "network"}},
				v1.Disk{Type: Type_PersistentVolumeClaim,
					Source: v1.DiskSource{Name: "pvc"}},
			}
		})

		It("Should extract PVCs from disks", func() {
			domCopy, pvcs := ExtractPvc(dom)
			Expect(len(domCopy.Devices.Disks)).To(Equal(2))
			Expect(domCopy.Devices.Disks[0].Type).To(Equal(Type_Network))

			Expect(len(pvcs)).To(Equal(1))
			Expect(pvcs[0].disk.Type).To(Equal(Type_PersistentVolumeClaim))

		})
	})

})

func TestMapper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Mapper")
}
