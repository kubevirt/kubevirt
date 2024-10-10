package main

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Virt chroot sanity", func() {
	It("virt-chroot should not be bloated with dependencies", func() {
		path, err := os.Executable()
		Expect(err).ToNot(HaveOccurred())
		f, err := os.Stat(path)
		Expect(err).ToNot(HaveOccurred())
		limitMi := 10
		Expect(f.Size()).To(BeNumerically("<", limitMi*1024*1024), "virt-chroot binary is larger than expected; consider cleaning up dependencies")
	})
})
