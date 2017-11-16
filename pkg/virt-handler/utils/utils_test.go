package utils

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Virt Handler Utils", func() {
	Context("PID Lookup", func() {
		var workDir string
		var err error
		BeforeEach(func() {
			workDir, err = ioutil.TempDir("", "kubetest-")
			Expect(err).To(BeNil())
		})

		AfterEach(func() {
			os.RemoveAll(workDir)
		})

		It("Should successfully read pid file", func() {
			fakePid := 32459

			fn := path.Join(workDir, "pidfile")
			f, err := os.Create(fn)
			Expect(err).ToNot(HaveOccurred())
			f.WriteString(fmt.Sprintf("%d", fakePid))

			pid, err := GetLibvirtPidFromFile(fn)
			Expect(err).ToNot(HaveOccurred())
			Expect(pid).To(Equal(fakePid))
		})

		It("Should get pid from socket", func() {
			socketPath := path.Join(workDir, "socket")

			// Just listening without accepting won't block...
			l, err := net.Listen("unix", socketPath)
			defer l.Close()

			pid, err := GetPid(socketPath)
			Expect(err).ToNot(HaveOccurred())
			// Since this process opened the socket, it will have the same pid
			Expect(pid).To(Equal(os.Getpid()))
		})

		It("Should fail to get pid of invalid socket", func() {
			socketPath := path.Join(workDir, "socket")
			pid, err := GetPid(socketPath)
			Expect(err).To(HaveOccurred())
			Expect(pid).To(Equal(-1))
			Expect(strings.Contains(err.Error(), "no such file or directory")).To(BeTrue())
		})
	})

	Context("Namespace Lookup", func() {
		It("Should map pids to namespaces", func() {
			pid := 12345
			ns := GetNSFromPid(pid)
			Expect(ns.Mnt).To(Equal(fmt.Sprintf("/proc/%d/ns/mnt", pid)))
		})
	})
})

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Virt Handler Utils")
}
