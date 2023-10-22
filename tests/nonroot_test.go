package tests_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/tests/decorators"

	"kubevirt.io/client-go/kubecli"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[sig-compute]NonRoot feature", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient
	var err error

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("[verify-nonroot] NonRoot feature", func() {
		It("Fails if can't be tested", func() {
			Expect(checks.HasFeature(virtconfig.Root)).To(BeFalse())

			vmi := tests.NewRandomVMI()
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
			Expect(err).NotTo(HaveOccurred())

			By("Check that runtimeuser was set on creation")
			Expect(vmi.Status.RuntimeUser).To(Equal(uint64(107)))

			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Check that user used is equal to 107")
			Expect(tests.GetIdOfLauncher(vmi)).To(Equal("107"))

		})
	})
})
