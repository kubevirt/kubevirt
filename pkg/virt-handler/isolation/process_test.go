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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package isolation

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("process", func() {
	const (
		processTestExecPath = "processA"
		processTestPID      = 110
		nonExistPPid        = 300
	)
	emptyProcessList := []Process{}
	procStub1 := ProcessStub{ppid: 1, pid: 120, binary: processTestExecPath}
	procStub2 := ProcessStub{ppid: processTestPID, pid: 2222, binary: "processB"}
	procStub3 := ProcessStub{ppid: 1, pid: 110, binary: "processC"}
	procStub4 := ProcessStub{ppid: processTestPID, pid: 3333, binary: "processD"}
	testProcesses := []Process{procStub1, procStub3, procStub2, procStub4}

	Context("find child processes", func() {
		DescribeTable("should return the correct child processes of the given pid",
			func(processes []Process, ppid int, expectedProcesses []Process) {
				Expect(childProcesses(processes, ppid)).
					To(ConsistOf(expectedProcesses))
			},
			Entry("given no input processes, there are no child processes",
				emptyProcessList, nonExistPPid, emptyProcessList,
			),
			Entry("given process list and non-exist pid, should return no child processes",
				testProcesses, nonExistPPid, emptyProcessList,
			),
			Entry("given process list and pid where there are child processes of the given pid",
				testProcesses, processTestPID, []Process{procStub2, procStub4},
			),
		)
	})

	Context("lookup process by executable", func() {
		procStub5 := ProcessStub{ppid: 100, pid: 220, binary: processTestExecPath}

		DescribeTable("should find no process",
			func(processes []Process, executable string) {
				Expect(lookupProcessByExecutable(processes, executable)).To(BeNil())
			},
			Entry("given no input processes and empty string as executable",
				emptyProcessList, "",
			),
			Entry("given no input processes and executable",
				emptyProcessList, "processA",
			),
			Entry("given processes list and empty string",
				testProcesses, "",
			),
		)

		DescribeTable("should return the first occurrence of a process that runs the given executable",
			func(processes []Process, executable string, expectedProcess Process) {
				Expect(lookupProcessByExecutable(processes, executable)).
					To(Equal(expectedProcess))
			},
			Entry("given processes list that includes exactly one process that runs the executable",
				testProcesses, processTestExecPath, procStub1,
			),
			Entry("given processes list that includes more than one process that runs the executable",
				append(testProcesses, procStub5), processTestExecPath, procStub1,
			),
		)
	})

	Context("find a process matching the pid", func() {
		It("should match process id with current process", func() {
			p, err := FindProcess(os.Getpid())
			Expect(err).ToNot(HaveOccurred())
			Expect(p).ToNot(BeNil())
			Expect(p.Pid()).To(Equal(os.Getpid()))
		})

		It("should return nil if it can't find matching process", func() {
			p, err := FindProcess(-1)
			Expect(err).ToNot(HaveOccurred())
			Expect(p).To(BeNil())
		})
	})

	Context("find process that triggered the tests", func() {
		It("should be able to find a go or bazel process in the slice", func() {
			p, err := Processes()
			Expect(err).ToNot(HaveOccurred())
			Expect(p).ToNot(BeEmpty())
			Eventually(func() bool {
				for _, p1 := range p {
					if p1.Executable() == "bazel" || p1.Executable() == "go" || p1.Executable() == "go.exe" {
						return true
					}
				}
				return false
			}).Should(BeTrue())
		})

	})

})

type ProcessStub struct {
	ppid   int
	pid    int
	binary string
}

func (p ProcessStub) Pid() int {
	return p.pid
}

func (p ProcessStub) PPid() int {
	return p.ppid
}

func (p ProcessStub) Executable() string {
	return p.binary
}
