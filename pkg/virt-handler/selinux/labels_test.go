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
		It("should detect that it is disabled if getenforce returns Disabled", func() {
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("disabled"), nil
			}
			present, mode, err := selinux.IsPresent()
			Expect(err).To(BeNil())
			Expect(present).To(BeFalse())
			Expect(mode).To(Equal("disabled"))
		})
		It("should detect that it is enabled if getenforce returns Permissive", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "getenforce"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("Permissive"), nil
			}
			present, _, err := selinux.IsPresent()
			Expect(err).To(BeNil())
			Expect(present).To(BeTrue())
		})
		It("should detect that it is enabled if getenforce does not return Disabled", func() {
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("enforcing"), nil
			}
			present, mode, err := selinux.IsPresent()
			Expect(err).To(BeNil())
			Expect(present).To(BeTrue())
			Expect(mode).To(Equal("enforcing"))
		})
	})

	Context("installing the selinux policy", func() {

		It("should fail if semanage exists but the command fails", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "semodule"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return nil, fmt.Errorf("I failed")
			}
			selinux.copyPolicyFunc = func(policyName string, dir string) (err error) {
				return fmt.Errorf("someting went wrong")
			}
			Expect(selinux.InstallPolicy("whatever")).ToNot(Succeed())
		})

		It("should succeed if semanage exists and the command runs successful", func() {
			touch(filepath.Join(tempDir, "/usr/bin", "semodule"))
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("I succeeded"), nil
			}
			selinux.copyPolicyFunc = func(policyName string, dir string) (err error) {
				return nil
			}
			Expect(selinux.InstallPolicy("whatever")).To(Succeed())
		})
		It("should fail if the semanage command does not exist and selinux is enabled", func() {
			selinux.mode = "enforcing"
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("I succeeded"), nil
			}
			selinux.copyPolicyFunc = func(policyName string, dir string) (err error) {
				return nil
			}
			Expect(selinux.InstallPolicy("whatever")).To(Not(Succeed()))
		})
		It("should succeed if the semanage command does not exist and selinux is permissive", func() {
			selinux.mode = "permissive"
			selinux.execFunc = func(binary string, args ...string) (bytes []byte, e error) {
				return []byte("I succeeded"), nil
			}
			selinux.copyPolicyFunc = func(policyName string, dir string) (err error) {
				return nil
			}
			Expect(selinux.InstallPolicy("whatever")).To(Succeed())
		})
	})
})

func touch(path string) {
	f, err := os.Create(path)
	Expect(err).ToNot(HaveOccurred())
	Expect(f.Close()).To(Succeed())
}
