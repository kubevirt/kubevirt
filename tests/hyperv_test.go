package tests_test

import (
	"context"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"libvirt.org/go/libvirtxml"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libvmi"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
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

	Context("VMI with an empty HyperV feature list", decorators.SigComputeMigrations, func() {
		It("should expand to the host's enlightenments and keep them across migration", decorators.RequiresTwoSchedulableNodes, func() {
			vmi := libvmifact.NewAlpine(libnet.WithMasqueradeNetworking(), withEmptyHyperv())
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Checking that libvirt expanded the hyperv features")
			hypervBefore := liveHypervFeatures(vmi)
			expectExpanded(hypervBefore)

			sourceNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Migrating the VMI")
			migration := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, libmigration.New(vmi.Name, vmi.Namespace))
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			By("Checking that the target pod required the hyperv labels of the source node")
			targetPod, err := libpod.GetTargetPodForMigration(migration)
			Expect(err).ToNot(HaveOccurred())
			sourceHypervLabels := hypervEntries(sourceNode.Labels)
			Expect(sourceHypervLabels).ToNot(BeEmpty())
			Expect(hypervEntries(targetPod.Spec.NodeSelector)).To(Equal(sourceHypervLabels))

			By("Checking that the hyperv features did not change")
			Expect(liveHypervFeatures(vmi)).To(Equal(hypervBefore))
		})

		It("should keep the expanded enlightenments across a dedicated-CPU migration", decorators.RequiresTwoWorkerNodesWithCPUManager, func() {
			vmi := libvmifact.NewAlpine(
				libnet.WithMasqueradeNetworking(),
				withEmptyHyperv(),
				libvmi.WithCPUCount(2, 1, 1),
				libvmi.WithDedicatedCPUPlacement(),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsSmall())
			hypervBefore := liveHypervFeatures(vmi)
			expectExpanded(hypervBefore)

			migration := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, libmigration.New(vmi.Name, vmi.Namespace))
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			Expect(liveHypervFeatures(vmi)).To(Equal(hypervBefore))
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

		vmi.Spec.Domain.Features.Hyperv.Reenlightenment = &v1.FeatureState{Enabled: virtpointer.P(true)}
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
		vmi.Spec.Domain.Features.HypervPassthrough.Enabled = virtpointer.P(true)
	}
}

func withEmptyHyperv() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		if vmi.Spec.Domain.Features == nil {
			vmi.Spec.Domain.Features = &v1.Features{}
		}
		vmi.Spec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
	}
}

// liveHypervFeatures uses the libvirt schema, which covers hyperv features
// that the VMI API cannot express.
func liveHypervFeatures(vmi *v1.VirtualMachineInstance) *libvirtxml.DomainFeatureHyperV {
	domXML, err := libdomain.GetRunningVirtualMachineInstanceDomainXML(kubevirt.Client(), vmi)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	domain := &libvirtxml.Domain{}
	ExpectWithOffset(1, domain.Unmarshal(domXML)).To(Succeed())
	ExpectWithOffset(1, domain.Features).ToNot(BeNil())
	ExpectWithOffset(1, domain.Features.HyperV).ToNot(BeNil())
	return domain.Features.HyperV
}

// expectExpanded checks the live features of a host-model domain: libvirt
// switched it to custom mode, enabled what the VMI left unset, and kept the
// reenlightenment override.
func expectExpanded(hyperv *libvirtxml.DomainFeatureHyperV) {
	ExpectWithOffset(1, hyperv.Mode).To(Equal("custom"))
	ExpectWithOffset(1, hyperv.Relaxed).To(Equal(&libvirtxml.DomainFeatureState{State: "on"}))
	ExpectWithOffset(1, hyperv.ReEnlightenment).To(Equal(&libvirtxml.DomainFeatureState{State: "off"}))
}

func hypervEntries(labels map[string]string) map[string]string {
	entries := map[string]string{}
	for key, value := range labels {
		if strings.HasPrefix(key, v1.HypervLabel) {
			entries[key] = value
		}
	}
	return entries
}
