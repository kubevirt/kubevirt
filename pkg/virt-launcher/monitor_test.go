/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package virtlauncher

import (
	"flag"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/google/uuid"
)

var fakeQEMUBinary string

func init() {
	flag.StringVar(&fakeQEMUBinary, "fake-qemu-binary-path", "_out/cmd/fake-qemu-process/fake-qemu-process", "path to cirros test image")
}

var _ = Describe("VirtLauncher", func() {
	var mon *monitor
	var cmd *exec.Cmd
	var gracefulShutdownChannel chan struct{}
	var pidDir string
	var processStarted bool

	startProcess := func() {
		cmd = exec.Command(fakeQEMUBinary, "--uuid", uuid.New().String(), "--pidfile", filepath.Join(pidDir, "fakens_fakevmi.pid"))
		err := cmd.Start()
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), "command failed to start")

		currentPid := cmd.Process.Pid
		ExpectWithOffset(1, currentPid).ToNot(Equal(0), "no PID")

		processStarted = true
	}

	stopProcess := func() {
		err := cmd.Process.Kill()
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), "failed to kill process")

		processStarted = false
	}

	verifyProcessStarted := func() {
		EventuallyWithOffset(1, func() bool {
			mon.refresh()
			return mon.pid != 0
		}).WithTimeout(10*time.Second).WithPolling(100*time.Millisecond).Should(BeTrue(), "process did not start")
	}

	verifyProcessStopped := func() {
		EventuallyWithOffset(1, func() bool {
			mon.refresh()
			return mon.pid == 0 && mon.isDone
		}).WithTimeout(10*time.Second).WithPolling(100*time.Millisecond).Should(BeTrue(), "process did not stop")
	}

	BeforeEach(func() {
		pidDir = GinkgoT().TempDir()

		processStarted = false
		if !strings.Contains(fakeQEMUBinary, "../../") {
			fakeQEMUBinary = filepath.Join("../../", fakeQEMUBinary)
		}
		gracefulShutdownChannel = make(chan struct{})
		shutdownCallback := func(pid int) {
			// Don't send SIGTERM to the current process group (i.e. to PID 0). That will interrupt
			// the test run.
			Expect(pid).ToNot(BeZero())
			err := syscall.Kill(pid, syscall.SIGTERM)
			Expect(err).ToNot(HaveOccurred())
		}
		gracefulShutdownCallback := func() {
			close(gracefulShutdownChannel)
		}
		mon = &monitor{
			domainName:               "fakens_fakevmi",
			pidDir:                   pidDir,
			gracePeriod:              30,
			finalShutdownCallback:    shutdownCallback,
			gracefulShutdownCallback: gracefulShutdownCallback,
		}
	})

	AfterEach(func() {
		if processStarted {
			stopProcess()
		}
		_ = cmd.Wait()
	})

	Describe("VirtLauncher", func() {
		Context("process monitor", func() {
			It("verify pid detection works", func() {
				startProcess()
				verifyProcessStarted()
				stopProcess()
				verifyProcessStopped()
			})

			It("verify zombie pid detection works", func() {
				startProcess()
				verifyProcessStarted()
				stopProcess()
				verifyProcessStopped()

				// cleanup after stopping ensures zombie process is detected
				_ = cmd.Wait()
			})

			It("verify start timeout works", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				go func() {
					defer GinkgoRecover()
					mon.RunForever(time.Second, stopChan)
					done <- "exit"
				}()
				Eventually(done).WithTimeout(3 * time.Second).WithPolling(100 * time.Millisecond).Should(Receive())
			})

			It("verify monitor loop exits when signal arrives and no pid is present", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				go func() {
					defer GinkgoRecover()
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				time.Sleep(time.Second)

				close(stopChan)

				Eventually(done).WithTimeout(5 * time.Second).WithPolling(100 * time.Millisecond).Should(
					Receive())
			})

			It("verify graceful shutdown trigger works", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				startProcess()
				verifyProcessStarted()

				go func() {
					defer GinkgoRecover()
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				Consistently(gracefulShutdownChannel).Should(Not(BeClosed()))
				close(stopChan)
				Eventually(gracefulShutdownChannel).Should(BeClosed())
			})

			It("verify grace period works", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				startProcess()
				verifyProcessStarted()
				go func() {
					defer GinkgoRecover()
					mon.gracePeriod = 1
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				close(stopChan)
				Eventually(done).WithTimeout(10 * time.Second).WithPolling(100 * time.Millisecond).Should(Receive())
			})
		})
	})
})
