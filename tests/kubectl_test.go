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
	var k8sClient string

	BeforeEach(func() {
		k8sClient = clientcmd.GetK8sCmdClient()
		clientcmd.FailIfNoCmd(k8sClient)
	})

	DescribeTable("[test_id:3812]explain vm/vmi", func(resource string) {
		output, stderr, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), k8sClient, "explain", resource)
		// kubectl will not find resource for the first time this command is issued
		if err != nil {
			output, _, err = clientcmd.RunCommand(testsuite.GetTestNamespace(nil), k8sClient, "explain", resource)
		}
		Expect(err).NotTo(HaveOccurred(), stderr)
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(SatisfyAny(
			ContainSubstring("metadata	<Object>"),
			ContainSubstring("metadata	<ObjectMeta>"),
		))
		Expect(output).To(ContainSubstring("spec	<Object>"))
		Expect(output).To(ContainSubstring("status	<Object>"))
	},
		Entry("[test_id:3810]explain vm", "vm"),
		Entry("[test_id:3811]explain vmi", "vmi"),
		Entry("[test_id:5178]explain vmim", "vmim"),
		Entry("[test_id:5179]explain kv", "kv"),
		Entry("[test_id:5180]explain vmsnapshot", "vmsnapshot"),
		Entry("[test_id:5181]explain vmsnapshotcontent", "vmsnapshotcontent"),
	)

	It("[test_id:5182]vmipreset have validation", func() {
		output, _, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), k8sClient, "explain", "vmipreset")
		if err != nil {
			output, _, err = clientcmd.RunCommand(testsuite.GetTestNamespace(nil), k8sClient, "explain", "vmipreset")
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(SatisfyAny(
			ContainSubstring("metadata	<Object>"),
			ContainSubstring("metadata	<ObjectMeta>"),
		))
		Expect(output).To(ContainSubstring("spec	<Object>"))
	})

	It("[test_id:5183]vmirs have validation", func() {
		output, _, err := clientcmd.RunCommand(testsuite.GetTestNamespace(nil), k8sClient, "explain", "vmirs")
		if err != nil {
			output, _, err = clientcmd.RunCommand(testsuite.GetTestNamespace(nil), k8sClient, "explain", "vmirs")
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(output).To(ContainSubstring("apiVersion	<string>"))
		Expect(output).To(ContainSubstring("kind	<string>"))
		Expect(output).To(SatisfyAny(
			ContainSubstring("metadata	<Object>"),
			ContainSubstring("metadata	<ObjectMeta>"),
		))
		Expect(output).To(ContainSubstring("spec	<Object>"))
	})

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
