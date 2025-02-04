package containerdisk

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/os/disk"
)

var _ = Describe("Validation", func() {

	var diskInfo disk.DiskInfo
	var sizeStub int64

	BeforeEach(func() {
		diskInfo = disk.DiskInfo{}
		sizeStub = 12345
	})

	Context("verify qcow2", func() {

		It("should return error if format is not qcow2", func() {
			diskInfo.Format = "not qcow2"
			err := VerifyQCOW2(&diskInfo)
			Expect(err).Should(HaveOccurred())
		})

		It("should return error if backing file exists", func() {
			diskInfo.Format = "qcow2"
			diskInfo.BackingFile = "my-super-awesome-file"
			err := VerifyQCOW2(&diskInfo)
			Expect(err).Should(HaveOccurred())
		})

		It("should run successfully", func() {
			diskInfo.Format = "qcow2"
			diskInfo.ActualSize = sizeStub
			diskInfo.VirtualSize = sizeStub
			err := VerifyQCOW2(&diskInfo)
			Expect(err).ShouldNot(HaveOccurred())
		})

	})

	Context("verify image", func() {

		It("should be successful if image is raw", func() {
			diskInfo.Format = "raw"
			err := VerifyImage(&diskInfo)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should succeed on qcow2 valid disk info", func() {
			diskInfo.Format = "qcow2"
			diskInfo.ActualSize = sizeStub
			diskInfo.VirtualSize = sizeStub
			err := VerifyImage(&diskInfo)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should fail on unknown format", func() {
			diskInfo.Format = "unknown format"
			err := VerifyImage(&diskInfo)
			Expect(err).Should(HaveOccurred())
		})

	})

})
