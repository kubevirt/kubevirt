package tests_test

import (
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("[sig-compute]kubectl integration", decorators.SigCompute, func() {
	BeforeEach(func() {
		clientcmd.FailIfNoCmd("kubectl")
	})

	Describe("kubectl logs", func() {
		var (
			vm *v1.VirtualMachineInstance
		)

		It("kubectl logs <vmi-pod> return default container log", func() {
			vm = libvmifact.NewCirros()
			vm = libvmops.RunVMIAndExpectLaunch(vm, 30)

			pod, err := libpod.GetPodByVirtualMachineInstance(vm, vm.Namespace)
			Expect(err).NotTo(HaveOccurred())
			output, _, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), "kubectl", "logs", pod.Name)
			Expect(err).NotTo(HaveOccurred())

			Expect(output).To(ContainSubstring("component"))
			Expect(output).To(ContainSubstring("level"))
			Expect(output).To(ContainSubstring("msg"))
			Expect(output).To(ContainSubstring("pos"))
			Expect(output).To(ContainSubstring("timestamp"))
		})
	})
})
