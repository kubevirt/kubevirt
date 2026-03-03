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

package common

import (
	"github.com/mitchellh/go-ps"

	. "github.com/onsi/ginkgo/v2"
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
		DescribeTable("should return the correct child processes of the given pid",
			func(processes []ps.Process, ppid int, expectedProcesses []ps.Process) {
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
				testProcesses, processTestPID, []ps.Process{procStub2, procStub4},
			),
		)
	})

	Context("lookup process by executable prefix", func() {
		procStub5 := ProcessStub{ppid: 100, pid: 220, binary: processTestExecPath}

		DescribeTable("should find no process",
			func(processes []ps.Process, executablePrefix string) {
				Expect(lookupProcessByExecutablePrefix(processes, executablePrefix)).To(BeNil())
			},
			Entry("given no input processes and empty string as executable prefix",
				emptyProcessList, "",
			),
			Entry("given no input processes and executable prefix",
				emptyProcessList, "processA",
			),
			Entry("given processes list and empty string",
				testProcesses, "",
			),
		)

		DescribeTable("should return the first occurrence of a process with the given executable prefix",
			func(processes []ps.Process, executablePrefix string, expectedProcess ps.Process) {
				Expect(lookupProcessByExecutablePrefix(processes, executablePrefix)).
					To(Equal(expectedProcess))
			},
			Entry("given processes list that includes exactly one process with the executable prefix",
				testProcesses, processTestExecPath, procStub1,
			),
			Entry("given processes list that includes more than one process with the executable prefix",
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

var _ = Describe("findIsolatedQemuProcess", func() {
	const virtLauncherPid = 1
	fakeProcess1 := ProcessStub{pid: virtLauncherPid, ppid: 0, binary: "fake-process-1"}
	fakeProcess2 := ProcessStub{pid: 26, ppid: virtLauncherPid, binary: "fake-process-2"}
	fakeProcess3 := ProcessStub{pid: 226, ppid: 26, binary: "fake-process-3"}
	virtLauncherProcesses := []ps.Process{
		fakeProcess1,
		fakeProcess2,
		fakeProcess3}

	qemuKvmProc := ProcessStub{pid: 101, ppid: virtLauncherPid, binary: "qemu-kvm"}
	qemuSystemProc := ProcessStub{pid: 101, ppid: virtLauncherPid, binary: "qemu-system-x86_64"}

	qemuProcessExecutablePrefixes := []string{"qemu-system", "qemu-kvm"}

	DescribeTable("should return QEMU process",
		func(processes []ps.Process, pid int, expectedProcess ps.Process) {
			proc, err := FindIsolatedQemuProcess(qemuProcessExecutablePrefixes, processes, pid)
			Expect(err).NotTo(HaveOccurred())
			Expect(proc).To(Equal(expectedProcess))
		},
		Entry("when qemu-kvm binary running",
			append(virtLauncherProcesses, qemuKvmProc),
			virtLauncherPid,
			qemuKvmProc,
		),
		Entry("when qemu-system binary running",
			append(virtLauncherProcesses, qemuSystemProc),
			virtLauncherPid,
			qemuSystemProc,
		),
	)
	It("should fail when no QEMU process exists", func() {
		proc, err := FindIsolatedQemuProcess(qemuProcessExecutablePrefixes, virtLauncherProcesses, virtLauncherPid)
		Expect(err).To(HaveOccurred())
		Expect(proc).To(BeNil())
	})
})
