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

var _ = Describe("[sig-compute]oc/kubectl integration", decorators.SigCompute, func() {
	Describe("oc/kubectl logs", func() {
		var (
			vm *v1.VirtualMachineInstance
		)

		It("oc/kubectl logs <vmi-pod> return default container log", func() {
			vm = libvmifact.NewCirros()
			vm = libvmops.RunVMIAndExpectLaunch(vm, 30)

			k8sClient := clientcmd.GetK8sCmdClient()
			pod, err := libpod.GetPodByVirtualMachineInstance(vm, vm.Namespace)
			Expect(err).NotTo(HaveOccurred())
			output, _, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), k8sClient, "logs", pod.Name)
			Expect(err).NotTo(HaveOccurred())

			Expect(output).To(ContainSubstring("component"))
			Expect(output).To(ContainSubstring("level"))
			Expect(output).To(ContainSubstring("msg"))
			Expect(output).To(ContainSubstring("pos"))
			Expect(output).To(ContainSubstring("timestamp"))
		})
	})
})
