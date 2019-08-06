package selinux

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("selinux", func() {

	var tempDir string
	var selinux *SELinuxImpl

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "kubevirt")
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

		It("should detect that it is disabled if getenforce is not in path", func() {
			present, err := selinux.IsPresent()
			Expect(err).To(BeNil())
			Expect(present).To(BeFalse())
		})
		It("should detect that it is disabled if getenforce returns Disabled", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "getenforce"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("Disabled"), nil
			}
			present, err := selinux.IsPresent()
			Expect(err).To(BeNil())
			Expect(present).To(BeFalse())
		})
		It("should detect that it is enabled if getenforce does not return Disabled", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "getenforce"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("Enforced"), nil
			}
			present, err := selinux.IsPresent()
			Expect(err).To(BeNil())
			Expect(present).To(BeTrue())
		})
	})

	Context("labeling a directory", func() {

		It("should fail if semanage does not exist", func() {
			Expect(selinux.Label("whatever", "nothing")).ToNot(Succeed())
		})

		It("should fail if semanage exists but the command fails", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "semanage"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return nil, fmt.Errorf("I failed")
			}
			Expect(selinux.Label("whatever", "nothing")).ToNot(Succeed())
		})

		It("should succeed if semanage exists and the command runs successful", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "semanage"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("I succeeded"), nil
			}
			Expect(selinux.Label("whatever", "nothing")).To(Succeed())
		})
	})

	Context("getting the label of a directory", func() {

		It("should fail if semanage does not exist", func() {
			_, err := selinux.IsLabeled("whatever")
			Expect(err).To(HaveOccurred())
		})

		It("should fail if semanage exists but the command fails", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "semanage"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return nil, fmt.Errorf("I failed")
			}
			_, err := selinux.IsLabeled("whatever")
			Expect(err).To(HaveOccurred())
		})

		It("should detect if the directory is not labeled", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "semanage"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("not found"), nil
			}
			labeled, err := selinux.IsLabeled("whatever")
			Expect(err).ToNot(HaveOccurred())
			Expect(labeled).To(BeFalse())
		})

		It("should detect if the directory is labeled", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "semanage"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("whatever(/.*)?"), nil
			}
			labeled, err := selinux.IsLabeled("whatever")
			Expect(err).ToNot(HaveOccurred())
			Expect(labeled).To(BeTrue())
		})
	})

	Context("restoring labels on a directory", func() {

		It("should fail if restorecon does not exist", func() {
			Expect(selinux.Restore("whatever")).ToNot(Succeed())
		})

		It("should fail if restorecon exists but the command fails", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "restorecon"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return nil, fmt.Errorf("I failed")
			}
			Expect(selinux.Restore("nothing")).ToNot(Succeed())
		})

		It("should succeed if restorecon exists and the command runs successful", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "restorecon"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("I succeeded"), nil
			}
			Expect(selinux.Restore("nothing")).To(Succeed())
		})
	})
})

func touch(path string) {
	f, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	Expect(f.Close()).To(Succeed())
}
