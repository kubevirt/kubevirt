package container_disk_v2alpha_test

import (
	"flag"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var containerDiskBinary string

func init() {
	testing.Init()
	flag.StringVar(&containerDiskBinary, "container-disk-binary", "_out/cmd/container-disk-v2alpha/container-disk", "path to container disk binary")
	flag.Parse()
	containerDiskBinary = filepath.Join("../../", containerDiskBinary)
}

var _ = Describe("the containerDisk binary", func() {

	It("should be able to handle 200 connections in 5 seconds without rejecting one of them", func() {
		dir, err := ioutil.TempDir("", "container-disk")
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
