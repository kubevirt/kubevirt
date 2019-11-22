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
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/client-go/log"
)

var fakeQEMUBinary string

func init() {
	flag.StringVar(&fakeQEMUBinary, "fake-qemu-binary-path", "_out/cmd/fake-qemu-process/fake-qemu-process", "path to cirros test image")
	flag.Parse()
	fakeQEMUBinary = filepath.Join("../../", fakeQEMUBinary)
}

var _ = Describe("VirtLauncher", func() {
	var mon *monitor
	var cmd *exec.Cmd
	var cmdLock sync.Mutex

	uuid := "123-123-123-123"

	tmpDir, _ := ioutil.TempDir("", "monitortest")
	tmpNetworkDir, _ := ioutil.TempDir("", "monitortest-network")

	log.Log.SetIOWriter(GinkgoWriter)

	dir := os.Getenv("PWD")
	dir = strings.TrimSuffix(dir, "pkg/virt-launcher")

	processStarted := false

	StartProcess := func() {
		cmdLock.Lock()
		defer cmdLock.Unlock()

		cmd = exec.Command(fakeQEMUBinary, "--uuid", uuid)
		err := cmd.Start()
		Expect(err).ToNot(HaveOccurred())

		currentPid := cmd.Process.Pid
		Expect(currentPid).ToNot(Equal(0))

		processStarted = true
	}

	StopProcess := func() {
		cmdLock.Lock()
		defer cmdLock.Unlock()

		cmd.Process.Kill()
		processStarted = false
	}

	CleanupProcess := func() {
		cmdLock.Lock()
		defer cmdLock.Unlock()

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
		InitializeSharedDirectories(tmpDir, tmpNetworkDir)
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
			cmdLock.Lock()
			defer cmdLock.Unlock()
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
				stopChan := make(chan struct{})
				done := make(chan string)

				go func() {
					mon.RunForever(time.Second, stopChan)
					done <- "exit"
				}()
				noExitCheck := time.After(3 * time.Second)

				exited := false
				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(BeTrue())
			})

			It("verify monitor loop exits when signal arrives and no pid is present", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				go func() {
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				time.Sleep(time.Second)

				close(stopChan)
				noExitCheck := time.After(5 * time.Second)
				exited := false

				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(BeTrue())
			})

			It("verify graceful shutdown trigger works", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				StartProcess()
				VerifyProcessStarted()
				go func() { CleanupProcess() }()

				go func() {
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				time.Sleep(time.Second)

				exists, err := hasGracefulShutdownTrigger(tmpDir, "fakenamespace", "fakedomain")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())

				close(stopChan)

				time.Sleep(time.Second)

				exists, err = hasGracefulShutdownTrigger(tmpDir, "fakenamespace", "fakedomain")
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeTrue())
			})

			It("verify grace period works", func() {
				stopChan := make(chan struct{})
				done := make(chan string)

				StartProcess()
				VerifyProcessStarted()
				go func() { CleanupProcess() }()
				go func() {
					mon.gracePeriod = 1
					mon.monitorLoop(1*time.Second, stopChan)
					done <- "exit"
				}()

				close(stopChan)
				noExitCheck := time.After(5 * time.Second)
				exited := false

				select {
				case <-noExitCheck:
				case <-done:
					exited = true
				}

				Expect(exited).To(BeTrue())
			})
		})
	})
})
