package disk

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Validation", func() {

	var diskInfo DiskInfo
	var sizeStub int64

	BeforeEach(func() {
		diskInfo = DiskInfo{}
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

	Context("verify raw", func() {

		// qemu-img reports any unrecognized file as raw, so size alignment is
		// the only signal separating disk images from arbitrary files (#19291).
		DescribeTable("should accept only positive multiples of the sector size",
			func(fileSize int64, expectSuccess bool) {
				diskInfo.Format = "raw"
				diskInfo.FileSize = fileSize
				err := VerifyRAW(&diskInfo)
				if expectSuccess {
					Expect(err).ShouldNot(HaveOccurred())
				} else {
					Expect(err).Should(HaveOccurred())
				}
			},
			Entry("single sector", int64(512), true),
			Entry("1MiB image", int64(1<<20), true),
			Entry("unaligned file", int64(1000), false),
			Entry("text file", int64(5465), false),
			Entry("empty file", int64(0), false),
		)

	})

	Context("verify image", func() {

		It("should be successful if image is raw", func() {
			diskInfo.Format = "raw"
			diskInfo.FileSize = 512
			err := VerifyImage(&diskInfo)
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should fail if raw image is not sector-aligned", func() {
			diskInfo.Format = "raw"
			diskInfo.FileSize = 1000
			err := VerifyImage(&diskInfo)
			Expect(err).Should(HaveOccurred())
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
