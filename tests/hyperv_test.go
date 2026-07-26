package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-compute] Hyper-V enlightenments", decorators.SigCompute, func() {

	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("VMI with HyperV re-enlightenment enabled", func() {
		var reEnlightenmentVMI *v1.VirtualMachineInstance

		vmiWithReEnlightenment := func() *v1.VirtualMachineInstance {
			return libvmifact.NewAlpine(libnet.WithMasqueradeNetworking(), withReEnlightenment())
		}

		BeforeEach(func() {
			reEnlightenmentVMI = vmiWithReEnlightenment()
		})

		When("TSC frequency is exposed on the cluster", decorators.Invtsc, func() {
			BeforeEach(func() {
				if !isTSCFrequencyExposed(virtClient) {
					Fail("TSC frequency is not exposed on the cluster")
				}
			})

			It("should be able to migrate", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), reEnlightenmentVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				reEnlightenmentVMI = libwait.WaitForSuccessfulVMIStart(reEnlightenmentVMI)

				By("Migrating the VM")
				migration := libmigration.New(reEnlightenmentVMI.Name, reEnlightenmentVMI.Namespace)
				migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				By("Checking VMI, confirm migration state")
				libmigration.ConfirmVMIPostMigration(virtClient, reEnlightenmentVMI, migrationUID)
			})
		})

		When(" TSC frequency is not exposed on the cluster", Serial, decorators.Reenlightenment, decorators.TscFrequencies, func() {

			BeforeEach(func() {
				if isTSCFrequencyExposed(virtClient) {
					for _, node := range libnode.GetAllSchedulableNodes(virtClient).Items {
						libinfra.ExpectStoppingNodeLabellerToSucceed(node.Name, virtClient)
						removeTSCFrequencyFromNode(node)
					}
				}
			})

			AfterEach(func() {
				for _, node := range libnode.GetAllSchedulableNodes(virtClient).Items {
					_ = libinfra.ExpectResumingNodeLabellerToSucceed(node.Name, virtClient)
				}
			})

			It("Should start successfully and be marked as non-migratable", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), reEnlightenmentVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(reEnlightenmentVMI)
				Expect(console.LoginToAlpine(reEnlightenmentVMI)).To(Succeed())
				Eventually(matcher.ThisVMI(reEnlightenmentVMI)).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(matcher.HaveConditionFalseWithMessage(v1.VirtualMachineInstanceIsMigratable, "HyperV Reenlightenment VMIs cannot migrate when TSC Frequency is not exposed on the cluster"))
			})
		})

	})

	Context("VMI with HyperV passthrough", func() {
		It("should be usable and non-migratable", func() {
			vmi := libvmifact.NewAlpine(withHypervPassthrough())
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())

			Eventually(matcher.ThisVMI(vmi), 60*time.Second, 1*time.Second).Should(matcher.HaveConditionFalse(v1.VirtualMachineInstanceIsMigratable))
		})
	})
})

func withReEnlightenment() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{}
		}
		if vmi.Spec.Domain.Features.Hyperv == nil {
			vmi.Spec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
		}

		vmi.Spec.Domain.Features.Hyperv.Reenlightenment = &v1.FeatureState{Enabled: new(true)}
	}
}

func withHypervPassthrough() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{}
		}
		if vmi.Spec.Domain.Features.HypervPassthrough == nil {
			vmi.Spec.Domain.Features.HypervPassthrough = &v1.HyperVPassthrough{}
		}
		vmi.Spec.Domain.Features.HypervPassthrough.Enabled = new(true)
	}
}
