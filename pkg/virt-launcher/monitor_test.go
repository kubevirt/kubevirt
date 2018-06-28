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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virtlauncher

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/log"
)

var _ = Describe("VirtLauncher", func() {
	var mon *monitor
	var cmd *exec.Cmd

	uuid := "123-123-123-123"

	tmpDir, _ := ioutil.TempDir("", "monitortest")

	log.Log.SetIOWriter(GinkgoWriter)

	dir := os.Getenv("PWD")
	dir = strings.TrimSuffix(dir, "pkg/virt-launcher")

	processName := "fake-qemu-process"
	processPath := dir + "/_out/cmd/fake-qemu-process/" + processName

	processStarted := false

	StartProcess := func() {
		cmd = exec.Command(processPath, "--uuid", uuid)
		err := cmd.Start()
		Expect(err).ToNot(HaveOccurred())

		currentPid := cmd.Process.Pid
		Expect(currentPid).ToNot(Equal(0))

		processStarted = true
	}

	StopProcess := func() {
		cmd.Process.Kill()
		processStarted = false
	}

	CleanupProcess := func() {
		cmd.Wait()
	}

	VerifyProcessStarted := func() {
		Eventually(func() bool {

			mon.refresh()
			if mon.pid != 0 {
				return true
			}
			return false

		}).Should(BeTrue())

	}

	VerifyProcessStopped := func() {
		Eventually(func() bool {

			mon.refresh()
			if mon.pid == 0 && mon.isDone == true {
				return true
			}
			return false

		}).Should(BeTrue())

	}

	BeforeEach(func() {
		InitializeSharedDirectories(tmpDir)
		triggerFile := GracefulShutdownTriggerFromNamespaceName(tmpDir, "fakenamespace", "fakedomain")
		shutdownCallback := func(pid int) {
			syscall.Kill(pid, syscall.SIGTERM)
		}
		mon = &monitor{
			cmdlineMatchStr:             uuid,
			gracePeriod:                 30,
			gracefulShutdownTriggerFile: triggerFile,
			shutdownCallback:            shutdownCallback,
		}
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
		if processStarted == true {
			cmd.Process.Kill()
		}
		processStarted = false
	})
	Describe("VirtLauncher", func() {
		Context("process monitor", func() {
			It("verify pid detection works", func() {
				StartProcess()
				VerifyProcessStarted()
				go func() { CleanupProcess() }()
				StopProcess()
				VerifyProcessStopped()
			})

			It("verify start timeout works", func() {
				done := make(chan string)

				go func() {
					mon.RunForever(time.Second)
					done <- "exit"
				}()
				noExitCheck := time.After(3 * time.Second)

				exited := false
				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(Equal(true))
			})

			It("verify monitor loop exits when signal arrives and no pid is present", func() {
				signalChannel := make(chan os.Signal, 1)
				done := make(chan string)

				go func() {
					mon.monitorLoop(1*time.Second, signalChannel)
					done <- "exit"
				}()

				time.Sleep(time.Second)

				signalChannel <- syscall.SIGQUIT
				noExitCheck := time.After(5 * time.Second)
				exited := false

				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(Equal(true))
			})

			It("verify graceful shutdown trigger works", func() {
				signalChannel := make(chan os.Signal, 1)
				done := make(chan string)

				StartProcess()
				VerifyProcessStarted()
				go func() { CleanupProcess() }()

				go func() {
					mon.monitorLoop(1*time.Second, signalChannel)
					done <- "exit"
				}()

				time.Sleep(time.Second)

				exists, err := hasGracefulShutdownTrigger(tmpDir, "fakenamespace", "fakedomain")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(Equal(false))

				signalChannel <- syscall.SIGQUIT

				time.Sleep(time.Second)

				exists, err = hasGracefulShutdownTrigger(tmpDir, "fakenamespace", "fakedomain")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(Equal(true))
			})

			It("verify grace period works", func() {
				signalChannel := make(chan os.Signal, 1)
				done := make(chan string)

				StartProcess()
				VerifyProcessStarted()
				go func() { CleanupProcess() }()
				go func() {
					mon.gracePeriod = 1
					mon.monitorLoop(1*time.Second, signalChannel)
					done <- "exit"
				}()

				signalChannel <- syscall.SIGTERM
				noExitCheck := time.After(5 * time.Second)
				exited := false

				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(Equal(true))
			})
		})
	})
})
