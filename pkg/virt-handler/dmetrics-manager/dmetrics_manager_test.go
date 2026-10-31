/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package dmetrics_manager

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("DownwardMetricsManager", func() {
	const vmiUID = "1234"

	newVMI := func() *v1.VirtualMachineInstance {
		vmi := &v1.VirtualMachineInstance{}
		vmi.UID = vmiUID
		return vmi
	}

	It("should deregister the VMI and record an event when the server fails to connect", func() {
		recorder := record.NewFakeRecorder(10)
		manager := NewDownwardMetricsManager("node01", recorder)
		vmi := newVMI()
		manager.stopServer[vmi.UID] = func() {}

		serverResult := make(chan error, 1)
		serverResult <- errors.New("connection failed")
		manager.watchServer(vmi, serverResult)

		Expect(manager.stopServer).ToNot(HaveKey(vmi.UID))
		Expect(recorder.Events).To(Receive(ContainSubstring("DownwardMetricsUnavailable")))
	})

	It("should deregister the VMI without recording an event on a clean stop", func() {
		recorder := record.NewFakeRecorder(10)
		manager := NewDownwardMetricsManager("node01", recorder)
		vmi := newVMI()
		manager.stopServer[vmi.UID] = func() {}

		serverResult := make(chan error, 1)
		serverResult <- nil
		manager.watchServer(vmi, serverResult)

		Expect(manager.stopServer).ToNot(HaveKey(vmi.UID))
		Consistently(recorder.Events).ShouldNot(Receive())
	})
})
