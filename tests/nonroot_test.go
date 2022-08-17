package tests_test

import (
	"fmt"

	"kubevirt.io/kubevirt/tests/libvmi"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
	})

	Context("should cause fail in creating of vmi with", func() {
		BeforeEach(func() {
			if !checks.HasFeature(virtconfig.NonRoot) {
				Skip("Test specific to NonRoot featureGate that is not enabled")
			}
		})

		It("[test_id:7127]VirtioFS", func() {
			if !checks.HasFeature(virtconfig.VirtIOFSGate) {
				Skip(fmt.Sprintf("Missing %s, enable %s featureGate.", virtconfig.VirtIOFSGate, virtconfig.VirtIOFSGate))
			}

			vmi := libvmi.New(libvmi.WithFilesystemPVC("test"))
			_, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(And(ContainSubstring("VirtioFS"), ContainSubstring("nonroot")))

		})
	})

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
