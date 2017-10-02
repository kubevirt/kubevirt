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
	"os"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("VirtLauncher", func() {
	var mon *monitor
	var cmd *exec.Cmd

	dir := os.Getenv("PWD")
	dir = strings.TrimSuffix(dir, "pkg/virt-launcher")

	processName := "fake-qemu-process"
	processPath := dir + "/cmd/fake-qemu-process/" + processName

	StartProcess := func() {
		cmd = exec.Command(processPath)
		err := cmd.Start()
		Expect(err).ToNot(HaveOccurred())
	}

	StopProcess := func() {
		cmd.Process.Kill()
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
		mon = &monitor{
			exename:   processName,
			debugMode: true,
		}
	})

	Describe("VirtLauncher", func() {
		Context("process monitor", func() {
			It("verify pid detection works", func() {
				StartProcess()
				VerifyProcessStarted()
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
		})
	})
})
