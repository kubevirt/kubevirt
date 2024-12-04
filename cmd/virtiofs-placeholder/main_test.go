package virtiofs_placeholder_test

import (
	"flag"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var placeholderBinary string

func init() {
	flag.StringVar(&placeholderBinary, "placeholder-binary", "_out/cmd/virtiofs-placeholder", "path to virtiofs placeholder binary")
}

var _ = Describe("the virtiofs placeholder binary", func() {
	var (
		sock string
		cmd  *exec.Cmd
	)
	BeforeEach(func() {
		if !strings.Contains(placeholderBinary, "../../") {
			placeholderBinary = filepath.Join("../../", placeholderBinary)
		}
		dir := GinkgoT().TempDir()
		sock = filepath.Join(dir, "placeholder.sock")
		cmd = exec.Command(placeholderBinary, "--socket", sock)
		Expect(cmd.Start()).To(Succeed())

	})
	It("should be able to handle 200 connections in 5 seconds without rejecting one of them", func() {
		time.Sleep(1 * time.Second)
		for i := 0; i < 200; i++ {
			conn, err := net.Dial("unix", sock)
			Expect(err).ToNot(HaveOccurred())
			conn.Close()
			time.Sleep(25 * time.Millisecond)
		}
		Expect(cmd.Process.Kill()).To(Succeed())
	})
	It("should exit if it recieves a SIGCHLD", func() {
		done := make(chan bool, 1)
		Expect(cmd.Process).ToNot(BeNil())
		go func() {
			Expect(cmd.Wait()).To(Succeed())
			done <- true
		}()
		Eventually(func() int {
			return cmd.Process.Pid
		}).WithTimeout(20 * time.Second).Should(BeNumerically(">", 0))
		By(fmt.Sprintf("Sending SIGCHLD to pid: %d", cmd.Process.Pid))
		out, err := exec.Command("kill", "-SIGCHLD", fmt.Sprintf("%d", cmd.Process.Pid)).CombinedOutput()
		Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("output:%s", string(out)))
		select {
		case <-done:
		case <-time.After(30 * time.Second):
			Fail("Timout expired waiting for the process to terminates")
		}
	})
})
