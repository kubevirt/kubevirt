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

package virtdhcp

import (
	"os"
	"strings"

	"time"

	"fmt"
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/log"
)

func IsDNSMasqPidRunning(pid int) bool {
	path := fmt.Sprintf("/proc/%d/cmdline", pid)
	fileExist, err := OS.isFileExist(path)
	if err != nil {
		panic(err)
	}

	if fileExist {
		content, err := ioutil.ReadFile(path)
		if err != nil {
			return false
		}
		return strings.Contains(string(content), "dnsmasq")
	}
	return false
}

var _ = Describe("Virt-DHCP Monitor test", func() {
	var pid int
	var prevPid int
	var monStop = make(chan bool)

	log.Log.SetIOWriter(GinkgoWriter)

	OS = &OSHandler{}
	dir := os.Getenv("PWD")
	dir = strings.TrimSuffix(dir, "pkg/virt-dhcp")
	processName := "fake-dnsmasq-process"
	processPath := dir + "/_out/cmd/fake-dnsmasq-process/" + processName
	processArgs := []string{"-k", "-d", "--strict-order", "--bind-dynamic"}

	StartProcess := func() {
		go runMonitor(processPath, processArgs, monStop, &pid)
		time.Sleep(1 * time.Second)
	}

	StopProcess := func() {
		monStop <- true
	}

	KillProcess := func() {
		prevPid = pid
		OS.killProcessIfExist(pid)
		time.Sleep(1 * time.Second)
	}

	VerifyProcessRestarted := func() {
		Eventually(func() bool {
			if pid != prevPid {
				return IsDNSMasqPidRunning(pid)
			}
			return false
		}).Should(BeTrue())
	}

	VerifyProcessStarted := func() {
		Eventually(func() bool {
			if pid != 0 {
				return IsDNSMasqPidRunning(pid)
			}
			return false
		}).Should(BeTrue())
	}

	VerifyProcessStopped := func() {
		Eventually(func() bool {
			ret := !IsDNSMasqPidRunning(pid)
			return ret
		}).Should(BeTrue())
	}

	BeforeEach(func() {
		OS = &OSHandler{}
		pid = 0
		prevPid = 0
	})

	Describe("Monitor test", func() {
		Context("process monitor", func() {
			It("verify pid detection works", func() {
				StartProcess()
				VerifyProcessStarted()
				StopProcess()
				VerifyProcessStopped()
			})

			It("should restart dnsmasq if killed", func() {
				StartProcess()
				VerifyProcessStarted()
				KillProcess()
				VerifyProcessRestarted()
				StopProcess()
				VerifyProcessStopped()
			})
		})
	})
})
