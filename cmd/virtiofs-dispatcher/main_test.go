package virtiofs_dispatcher_test

import (
	"flag"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var dispatcherBinary string

func init() {
	flag.StringVar(&dispatcherBinary, "dispatcher-binary", "_out/cmd/virtiofs-dispatcher", "path to virtiofs dispatcher binary")
}

var _ = Describe("the virtiosfd dispatcherr binary", func() {
	var (
		sock       string
		dir        string
		cmdUnshare *exec.Cmd
		pid        int
	)
	BeforeEach(func() {
		if !strings.Contains(dispatcherBinary, "../../") {
			dispatcherBinary = filepath.Join("../../", dispatcherBinary)
		}

		// Start unshare with a separate pid namespace
		cmdUnshare = exec.Command("unshare", "-p", "-C", "/usr/bin/tail", "-f", "/dev/null")
		Expect(cmdUnshare.Start()).To(Succeed())
		Expect(cmdUnshare.Process).ToNot(BeNil())
		Eventually(func() int {
			return cmdUnshare.Process.Pid
		}).WithTimeout(20 * time.Second).Should(BeNumerically(">", 0))

		dir = GinkgoT().TempDir()
		dirSock := GinkgoT().TempDir()
		sock = filepath.Join(dirSock, "vfsd.sock")
		pid = cmdUnshare.Process.Pid
	})
	AfterEach(func() {
		cmdUnshare.Process.Kill()
	})

	It("should be able to start the dispatcher", func() {
		cmd := exec.Command(dispatcherBinary, "--pid", strconv.Itoa(pid),
			"--shared-dir", dir, "--socket-path", sock)
		out, err := cmd.CombinedOutput()
		Expect(err).ShouldNot(HaveOccurred(), fmt.Sprintf("with output: %s", out))
	})

	It("should failed to start the dispatcher if container is terminated", func() {
		cmd := exec.Command(dispatcherBinary, "--pid", strconv.Itoa(pid),
			"--shared-dir", dir, "--socket-path", sock)
		Expect(cmdUnshare.Process.Kill()).To(Succeed())
		out, err := cmd.CombinedOutput()
		Expect(err).To(HaveOccurred())
		Expect(string(out)).To(ContainSubstring("failed to move process into the namespace"))
	})
})
