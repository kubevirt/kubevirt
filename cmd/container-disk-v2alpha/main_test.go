package container_disk_v2alpha_test

import (
	"flag"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var containerDiskBinary string

func init() {
	flag.StringVar(&containerDiskBinary, "container-disk-binary", "_out/cmd/container-disk-v2alpha/container-disk", "path to container disk binary")
}

var _ = Describe("the containerDisk binary", func() {
	BeforeEach(func() {
		if !strings.Contains(containerDiskBinary, "../../") {
			containerDiskBinary = filepath.Join("../../", containerDiskBinary)
		}
	})

	It("should be able to handle 200 connections in 5 seconds without rejecting one of them", func() {
		dir, err := os.MkdirTemp("", "container-disk")
		Expect(err).ToNot(HaveOccurred())
		defer os.RemoveAll(dir)
		cmd := exec.Command(containerDiskBinary, "-c", filepath.Join(dir, "testsocket"))
		Expect(cmd.Start()).To(Succeed())

		time.Sleep(1 * time.Second)
		for i := 0; i < 200; i++ {
			conn, err := net.Dial("unix", filepath.Join(dir, "testsocket.sock"))
			Expect(err).ToNot(HaveOccurred())
			conn.Close()
			time.Sleep(25 * time.Millisecond)
		}
		Expect(cmd.Process.Kill()).To(Succeed())
	})
})
