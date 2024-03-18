package selinux

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("selinux", func() {

	var tempDir string
	var selinux *SELinuxImpl

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(tempDir, "/usr/sbin"), 0777)).ToNot(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(tempDir, "/usr/bin"), 0777)).ToNot(HaveOccurred())
		selinux = &SELinuxImpl{
			Paths:         []string{"/usr/bin", "/usr/sbin"},
			procOnePrefix: tempDir,
		}
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("detecting if selinux is present", func() {
		It("should detect that it is disabled if getenforce returns Disabled", func() {
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("disabled"), nil
			}
			present, mode, err := selinux.IsPresent()
			Expect(err).ToNot(HaveOccurred())
			Expect(present).To(BeFalse())
			Expect(mode).To(Equal("disabled"))
		})
		It("should detect that it is enabled if getenforce returns Permissive", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "getenforce"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("Permissive"), nil
			}
			present, _, err := selinux.IsPresent()
			Expect(err).ToNot(HaveOccurred())
			Expect(present).To(BeTrue())
		})
		It("should detect that it is enabled if getenforce does not return Disabled", func() {
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("enforcing"), nil
			}
			present, mode, err := selinux.IsPresent()
			Expect(err).ToNot(HaveOccurred())
			Expect(present).To(BeTrue())
			Expect(mode).To(Equal("enforcing"))
		})
	})
})

func touch(path string) {
	f, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	Expect(f.Close()).To(Succeed())
}
