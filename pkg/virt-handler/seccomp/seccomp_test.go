package seccomp

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Seccomp", func() {
	Context("Install", func() {
		var path string

		BeforeEach(func() {
			var err error
			path, err = os.MkdirTemp("", "seccomp")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(path)).To(Succeed())
		})

		It("Should install", func() {
			err := InstallPolicy(path)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(filepath.Join(path, "seccomp", "kubevirt", "kubevirt.json"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should not install if equal", func() {
			err := InstallPolicy(path)
			Expect(err).NotTo(HaveOccurred())

			fInfo, err := os.Stat(filepath.Join(path, "seccomp", "kubevirt", "kubevirt.json"))
			Expect(err).NotTo(HaveOccurred())
			modified := fInfo.ModTime()

			time.Sleep(10 * time.Millisecond)

			err = InstallPolicy(path)
			Expect(err).NotTo(HaveOccurred())
			fInfo, err = os.Stat(filepath.Join(path, "seccomp", "kubevirt", "kubevirt.json"))
			Expect(err).NotTo(HaveOccurred())

			Expect(fInfo.ModTime()).To(Equal(modified))
		})

		It("Should reinstall", func() {
			policyDir := filepath.Join(path, "seccomp", "kubevirt")
			policyPath := filepath.Join(policyDir, "kubevirt.json")
			err := os.MkdirAll(policyDir, 0o700)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(policyPath, []byte{}, 0o777)
			Expect(err).NotTo(HaveOccurred())

			err = InstallPolicy(path)
			Expect(err).NotTo(HaveOccurred())

			b, err := os.ReadFile(policyPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(b).NotTo(Equal([]byte{}))
		})
	})
})
