// This file is part of the KubeVirt project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Copyright The KubeVirt Authors.

package virt_launcher_monitor_test

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
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

func launcherEnv(launcher string) []string {
	env := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if !strings.HasPrefix(item, "VIRT_LAUNCHER=") {
			env = append(env, item)
		}
	}
	return append(env, "VIRT_LAUNCHER="+launcher)
}

func expectMonitorExitCode(cmd *exec.Cmd, want int) {
	err := cmd.Run()
	if want == 0 {
		Expect(err).To(Succeed())
		return
	}
	Expect(err).To(HaveOccurred())
	var exitErr *exec.ExitError
	Expect(errors.As(err, &exitErr)).To(BeTrue())
	Expect(exitErr.ExitCode()).To(Equal(want))
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
		diskDir := filepath.Join(tmpDir, "disks")
		Expect(os.Mkdir(diskDir, 0o755)).To(Succeed())
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nprintf '%s\\n' \"$@\" > \""+argsFile+"\"\nexit 0\n")

		cmd := exec.Command(monitorPath(), "--uid", "vmi-uid", "--container-disk-dir", diskDir,
			"--keep-after-failure", "--keep-after-failure=false", "--qemu-timeout", "1s",
			"--unknown", "-v", "4")
		cmd.Env = launcherEnv(launcher)
		Expect(cmd.Run()).To(Succeed())

		body, err := os.ReadFile(argsFile)
		Expect(err).ToNot(HaveOccurred())
		got := strings.Split(strings.TrimSpace(string(body)), "\n")
		Expect(got).To(Equal([]string{"--uid", "vmi-uid", "--container-disk-dir", diskDir,
			"--keep-after-failure=false", "--qemu-timeout", "1s", "--unknown", "-v", "4"}))
	})

	It("returns exit code 1 when virt-launcher cannot be exec'd", func() {
		cmd := exec.Command(monitorPath(), "--uid", "vmi-uid")
		cmd.Env = launcherEnv(filepath.Join(tmpDir, "does-not-exist"))
		expectMonitorExitCode(cmd, 1)
	})

	It("propagates the virt-launcher exit code", func() {
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 42\n")
		cmd := exec.Command(monitorPath(), "--uid", "vmi-uid")
		cmd.Env = launcherEnv(launcher)
		err := cmd.Run()
		Expect(err).To(HaveOccurred())
		var exitErr *exec.ExitError
		Expect(errors.As(err, &exitErr)).To(BeTrue())
		Expect(exitErr.ExitCode()).To(Equal(42))
	})

	It("preserves a real launcher exit code of 127", func() {
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 127\n")
		cmd := exec.Command(monitorPath())
		cmd.Env = launcherEnv(launcher)
		expectMonitorExitCode(cmd, 127)
	})

	It("does not park after a successful launcher exit", func() {
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 0\n")
		cmd := exec.Command(monitorPath(), "--keep-after-failure")
		cmd.Env = launcherEnv(launcher)
		done := make(chan error, 1)
		go func() {
			done <- cmd.Run()
		}()
		Eventually(done, 5*time.Second).Should(Receive(BeNil()))
	})

	It("does not park when --keep-after-failure is explicitly false", func() {
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 42\n")
		cmd := exec.Command(monitorPath(), "--keep-after-failure=false")
		cmd.Env = launcherEnv(launcher)
		expectMonitorExitCode(cmd, 42)
	})

	It("parks after a failing launcher when --keep-after-failure is set", func() {
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 42\n")
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		cmd := exec.CommandContext(ctx, monitorPath(), "--keep-after-failure")
		cmd.Env = launcherEnv(launcher)
		_ = cmd.Run()
		Expect(ctx.Err()).To(Equal(context.DeadlineExceeded))
	})

	It("removes only top-level .sock entries from the container disk directory", func() {
		diskDir := filepath.Join(tmpDir, "disks")
		nestedDir := filepath.Join(diskDir, "nested")
		Expect(os.MkdirAll(nestedDir, 0o755)).To(Succeed())
		socketFile := filepath.Join(diskDir, "disk.sock")
		ordinaryFile := filepath.Join(diskDir, "disk.img")
		nestedSocket := filepath.Join(nestedDir, "nested.sock")
		Expect(os.WriteFile(socketFile, []byte("socket"), 0o600)).To(Succeed())
		Expect(os.WriteFile(ordinaryFile, []byte("image"), 0o600)).To(Succeed())
		Expect(os.WriteFile(nestedSocket, []byte("nested"), 0o600)).To(Succeed())
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 0\n")
		cmd := exec.Command(monitorPath(), "--container-disk-dir", diskDir)
		cmd.Env = launcherEnv(launcher)
		Expect(cmd.Run()).To(Succeed())
		_, err := os.Stat(socketFile)
		Expect(os.IsNotExist(err)).To(BeTrue())
		_, err = os.Stat(ordinaryFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(nestedSocket)
		Expect(err).ToNot(HaveOccurred())
	})

	It("uses the last container-disk-dir before -- and ignores later values", func() {
		firstDir := filepath.Join(tmpDir, "first")
		lastDir := filepath.Join(tmpDir, "last")
		afterDashDir := filepath.Join(tmpDir, "after-dash")
		for _, directory := range []string{firstDir, lastDir, afterDashDir} {
			Expect(os.Mkdir(directory, 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(directory, "disk.sock"), []byte("socket"), 0o600)).To(Succeed())
		}
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\nexit 0\n")
		cmd := exec.Command(monitorPath(), "--container-disk-dir="+firstDir,
			"--container-disk-dir", lastDir, "--", "--container-disk-dir="+afterDashDir)
		cmd.Env = launcherEnv(launcher)
		Expect(cmd.Run()).To(Succeed())
		for _, directory := range []string{firstDir, afterDashDir} {
			_, err := os.Stat(filepath.Join(directory, "disk.sock"))
			Expect(err).ToNot(HaveOccurred())
		}
		_, err := os.Stat(filepath.Join(lastDir, "disk.sock"))
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("forwards SIGTERM to virt-launcher", func() {
		started := filepath.Join(tmpDir, "started")
		received := filepath.Join(tmpDir, "received-term")
		launcher := writeFakeLauncher(tmpDir, "#!/bin/sh\ntouch \""+started+"\"\ntrap 'touch \""+received+"\"; exit 0' TERM\nwhile :; do :; done\n")
		cmd := exec.Command(monitorPath())
		cmd.Env = launcherEnv(launcher)
		Expect(cmd.Start()).To(Succeed())
		Eventually(func() bool {
			_, err := os.Stat(started)
			return err == nil
		}, 5*time.Second).Should(BeTrue())
		Expect(cmd.Process.Signal(syscall.SIGTERM)).To(Succeed())
		Expect(cmd.Wait()).To(Succeed())
		_, err := os.Stat(received)
		Expect(err).ToNot(HaveOccurred())
	})

	It("reaps orphaned qemu-system and qemu-kvm children during cleanup", func() {
		for _, processName := range []string{"qemu-system-x86_64", "qemu-kvm"} {
			pidFile := filepath.Join(tmpDir, processName+".pid")
			launcher := writeFakeLauncher(tmpDir, "#!/bin/bash\n/bin/bash -c 'exec -a "+processName+" /bin/bash -c \"while :; do :; done\"' &\necho $! > \""+pidFile+"\"\nsleep 0.1\nexit 0\n")
			cmd := exec.Command(monitorPath())
			cmd.Env = launcherEnv(launcher)
			Expect(cmd.Run()).To(Succeed())
			pidBytes, err := os.ReadFile(pidFile)
			Expect(err).ToNot(HaveOccurred())
			var pid int
			_, err = fmt.Sscanf(string(pidBytes), "%d", &pid)
			Expect(err).ToNot(HaveOccurred())
			DeferCleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
			killErr := syscall.Kill(pid, 0)
			var cmdline, stat []byte
			if killErr == nil {
				cmdline, _ = os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
				stat, _ = os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
			}
			Expect(killErr).To(HaveOccurred(), "process %s pid %d still exists (cmdline=%q stat=%q)",
				processName, pid, cmdline, stat)
		}
	})

	It("does not mistake qemu-img for a QEMU guest process", func() {
		pidFile := filepath.Join(tmpDir, "qemu-img.pid")
		launcher := writeFakeLauncher(tmpDir, "#!/bin/bash\n/bin/bash -c 'exec -a qemu-img /bin/bash -c \"while :; do :; done\"' &\necho $! > \""+pidFile+"\"\nsleep 0.1\nexit 0\n")
		cmd := exec.Command(monitorPath())
		cmd.Env = launcherEnv(launcher)
		Expect(cmd.Run()).To(Succeed())
		pidBytes, err := os.ReadFile(pidFile)
		Expect(err).ToNot(HaveOccurred())
		var pid int
		_, err = fmt.Sscanf(string(pidBytes), "%d", &pid)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
		Expect(syscall.Kill(pid, 0)).To(Succeed())
		Expect(syscall.Kill(pid, syscall.SIGTERM)).To(Succeed())
	})
})
