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
