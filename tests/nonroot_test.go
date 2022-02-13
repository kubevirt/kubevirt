package tests_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute]NonRoot feature", func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		if !checks.HasFeature(virtconfig.NonRoot) {
			Skip("Test specific to NonRoot featureGate that is not enabled")
		}

		tests.BeforeTestCleanup()
	})

	virtioFsVM := func() *v1.VirtualMachineInstance {
		name := "test"
		return tests.NewRandomVMIWithPVCFS(name)
	}

	DescribeTable("should cause fail in creating of vmi with", func(createVMI func() *v1.VirtualMachineInstance, neededFeature, feature string) {
		if neededFeature != "" && !checks.HasFeature(neededFeature) {
			Skip(fmt.Sprintf("Missing %s, enable %s featureGate.", neededFeature, neededFeature))
		}

		vmi := createVMI()
		_, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(And(ContainSubstring(feature), ContainSubstring("nonroot")))

	},
		Entry("[test_id:7127]VirtioFS", virtioFsVM, virtconfig.VirtIOFSGate, "VirtioFS"),
	)

	Context("[verify-nonroot] NonRoot feature", func() {
		It("Fails if can't be tested", func() {
			Expect(checks.HasFeature(virtconfig.NonRoot)).To(BeTrue())

			vmi := tests.NewRandomVMI()
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).NotTo(HaveOccurred())

			By("Check that runtimeuser was set on creation")
			Expect(vmi.Status.RuntimeUser).To(Equal(uint64(107)))

			tests.WaitForSuccessfulVMIStart(vmi)

			By("Check that user used is equal to 107")
			Expect(tests.GetIdOfLauncher(vmi)).To(Equal("107"))

		})
	})
})
