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
	"github.com/mitchellh/go-ps"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("process", func() {
	const (
		processTestExecPath = "processA"
		processTestPID      = 110
		nonExistPPid        = 300
	)
	emptyProcessList := []ps.Process{}
	procStub1 := ProcessStub{ppid: 1, pid: 120, binary: processTestExecPath}
	procStub2 := ProcessStub{ppid: processTestPID, pid: 2222, binary: "processB"}
	procStub3 := ProcessStub{ppid: 1, pid: 110, binary: "processC"}
	procStub4 := ProcessStub{ppid: processTestPID, pid: 3333, binary: "processD"}
	testProcesses := []ps.Process{procStub1, procStub3, procStub2, procStub4}

	Context("find child processes", func() {
		table.DescribeTable("should return the correct child processes of the given pid",
			func(processes []ps.Process, ppid int, expectedProcesses []ps.Process) {
				Expect(childProcesses(processes, ppid)).
					To(ConsistOf(expectedProcesses))
			},
			table.Entry("given no input processes, there are no child processes",
				emptyProcessList, nonExistPPid, emptyProcessList,
			),
			table.Entry("given process list and non-exist pid, should return no child processes",
				testProcesses, nonExistPPid, emptyProcessList,
			),
			table.Entry("given process list and pid where there are child processes of the given pid",
				testProcesses, processTestPID, []ps.Process{procStub2, procStub4},
			),
		)
	})

	Context("lookup process by executable", func() {
		procStub5 := ProcessStub{ppid: 100, pid: 220, binary: processTestExecPath}

		table.DescribeTable("should find no process",
			func(processes []ps.Process, executable string) {
				Expect(lookupProcessByExecutable(processes, executable)).To(BeNil())
			},
			table.Entry("given no input processes and empty string as executable",
				emptyProcessList, "",
			),
			table.Entry("given no input processes and executable",
				emptyProcessList, "processA",
			),
			table.Entry("given processes list and empty string",
				testProcesses, "",
			),
		)

		table.DescribeTable("should return the first occurrence of a process that runs the given executable",
			func(processes []ps.Process, executable string, expectedProcess ps.Process) {
				Expect(lookupProcessByExecutable(processes, executable)).
					To(Equal(expectedProcess))
			},
			table.Entry("given processes list that includes exactly one process that runs the executable",
				testProcesses, processTestExecPath, procStub1,
			),
			table.Entry("given processes list that includes more than one process that runs the executable",
				append(testProcesses, procStub5), processTestExecPath, procStub1,
			),
		)
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
