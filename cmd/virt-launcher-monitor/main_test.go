package virt_launcher_monitor_test

import (
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var monitorBinary string

func init() {
	flag.StringVar(&monitorBinary, "virt-launcher-monitor-binary", "_out/cmd/virt-launcher-monitor/virt-launcher-monitor", "path to virt-launcher-monitor")
}

func monitorPath() string {
	if filepath.IsAbs(monitorBinary) || strings.Contains(monitorBinary, "../../") {
		return monitorBinary
	}
	return filepath.Join("../../", monitorBinary)
}

func writeFakeLauncher(dir, script string) string {
	path := filepath.Join(dir, "fake-virt-launcher")
	Expect(os.WriteFile(path, []byte(script), 0o755)).To(Succeed())
	return path
}

var _ = Describe("virt-launcher-monitor", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "virt-launcher-monitor")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("forwards launcher arguments except --keep-after-failure", func() {
		argsFile := filepath.Join(tmpDir, "args")
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nprintf '%s\\n' \"$@\" > \""+argsFile+"\"\nexit 0\n")

		cmd := exec.Command(monitorPath(), "--uid", "vmi-uid", "--keep-after-failure", "--qemu-timeout", "1s", "--unknown")
		cmd.Env = append(os.Environ(), "VIRT_LAUNCHER="+launcher)
		Expect(cmd.Run()).To(Succeed())

		body, err := os.ReadFile(argsFile)
		Expect(err).ToNot(HaveOccurred())
		got := strings.Split(strings.TrimSpace(string(body)), "\n")
		Expect(got).To(Equal([]string{"--uid", "vmi-uid", "--qemu-timeout", "1s", "--unknown"}))
	})

	It("propagates the virt-launcher exit code", func() {
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 42\n")
		cmd := exec.Command(monitorPath(), "--uid", "vmi-uid")
		cmd.Env = append(os.Environ(), "VIRT_LAUNCHER="+launcher)
		err := cmd.Run()
		Expect(err).To(HaveOccurred())
		var exitErr *exec.ExitError
		Expect(errors.As(err, &exitErr)).To(BeTrue())
		Expect(exitErr.ExitCode()).To(Equal(42))
	})

	It("does not park after a successful launcher exit", func() {
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 0\n")
		cmd := exec.Command(monitorPath(), "--keep-after-failure")
		cmd.Env = append(os.Environ(), "VIRT_LAUNCHER="+launcher)
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()
		Eventually(done, 5*time.Second).Should(Receive(BeNil()))
	})
})
