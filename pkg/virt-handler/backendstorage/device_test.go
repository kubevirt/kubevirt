package backendstorage

import (
	"log"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/safepath"
)

var _ = Describe("Check filesystem", func() {
	BeforeEach(func() {
		var err error
		testTempDir, err = prepareFilesystemTestEnv("check-backend-filesystem-")
		log.Printf("Temporary directory created at %s", testTempDir)
		Expect(err).ToNot(HaveOccurred())
	})
	AfterEach(func() {
		err := cleanupFilesystemTestEnv(testTempDir)
		Expect(err).ToNot(HaveOccurred())
	})
	It("should properly detect ext3 filesystem", func() {
		log.Printf("Using temporary directory: %s", testTempDir)
		vmStateDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, "/var/lib/libvirt/vm-state")
		Expect(err).ToNot(HaveOccurred())
		m, err := isExt3Mounted(vmStateDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(m).To(BeTrue())

		nvramDir, err := safepath.JoinAndResolveWithRelativeRoot(testTempDir, "/var/lib/libvirt/qemu/nvram")
		Expect(err).ToNot(HaveOccurred())
		m, err = isExt3Mounted(nvramDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(m).To(BeFalse())
	})
})
