package tests_test

import (
	"context"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/libvmi"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libvmops"

	"kubevirt.io/kubevirt/tests/libnet"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/tests/libinfra"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
)

var _ = Describe("[sig-compute] Hyper-V enlightenments", decorators.SigCompute, func() {
	var virtClient kubecli.KubevirtClient
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

			It("should have TSC frequency set up in label and domain", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), reEnlightenmentVMI, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				reEnlightenmentVMI = libwait.WaitForSuccessfulVMIStart(reEnlightenmentVMI)

				virtLauncherPod, err := libpod.GetPodByVirtualMachineInstance(reEnlightenmentVMI, reEnlightenmentVMI.Namespace)
				Expect(err).NotTo(HaveOccurred())
				foundNodeSelector := false
				for key := range virtLauncherPod.Spec.NodeSelector {
					if strings.HasPrefix(key, topology.TSCFrequencySchedulingLabel+"-") {
						foundNodeSelector = true
						break
					}
				}
				Expect(foundNodeSelector).To(BeTrue(), "wasn't able to find a node selector key with prefix ", topology.TSCFrequencySchedulingLabel)

				domainSpec, err := libdomain.GetRunningVMIDomainSpec(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())

				foundTscTimer := false
				for _, timer := range domainSpec.Clock.Timer {
					if timer.Name == "tsc" {
						foundTscTimer = true
						break
					}
				}
				Expect(foundTscTimer).To(BeTrue(), "wasn't able to find tsc timer in domain spec")
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

		It("the vmi with HyperV feature matching a nfd label on a node should be scheduled", func() {
			enableHyperVInVMI := func(label string) v1.FeatureHyperv {
				features := v1.FeatureHyperv{}
				trueV := true
				switch label {
				case "vpindex":
					features.VPIndex = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "runtime":
					features.Runtime = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "reset":
					features.Reset = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "synic":
					features.SyNIC = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "frequencies":
					features.Frequencies = &v1.FeatureState{
						Enabled: &trueV,
					}
				case "reenlightenment":
					features.Reenlightenment = &v1.FeatureState{
						Enabled: &trueV,
					}
				}

				return features
			}
			var supportedKVMInfoFeature []string
			nodes := libnode.GetAllSchedulableNodes(virtClient)
			Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
			node := &nodes.Items[0]
			supportedCPUs := libnode.GetSupportedCPUModels(*nodes)
			Expect(supportedCPUs).ToNot(BeEmpty(), "There should be some supported cpu models")

			for key := range node.Labels {
				if strings.Contains(key, v1.HypervLabel) &&
					!strings.Contains(key, "tlbflush") &&
					!strings.Contains(key, "ipi") &&
					!strings.Contains(key, "synictimer") {
					supportedKVMInfoFeature = append(supportedKVMInfoFeature, strings.TrimPrefix(key, v1.HypervLabel))
				}
			}

			for _, label := range supportedKVMInfoFeature {
				vmi := libvmifact.NewCirros()
				features := enableHyperVInVMI(label)
				vmi.Spec.Domain.Features = &v1.Features{
					Hyperv: &features,
				}

				vmi, err := virtClient.VirtualMachineInstance(testsuite.NamespaceTestDefault).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should create VMI when using %v", label)
				libwait.WaitForSuccessfulVMIStart(vmi)

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI when using %v", label)
			}
		})

		DescribeTable(" the vmi with EVMCS HyperV feature should have correct HyperV and cpu features auto filled", Serial, func(featureState *v1.FeatureState) {
			config.EnableFeatureGate(featuregate.HypervStrictCheckGate)
			vmi := libvmifact.NewCirros()
			vmi.Spec.Domain.Features = &v1.Features{
				Hyperv: &v1.FeatureHyperv{
					EVMCS: featureState,
				},
			}
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)

			var err error
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should get VMI")
			Expect(vmi.Spec.Domain.Features.Hyperv.EVMCS).ToNot(BeNil(), "evmcs should not be nil")
			Expect(vmi.Spec.Domain.CPU).ToNot(BeNil(), "cpu topology can't be nil")
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

			if featureState.Enabled == nil || *featureState.Enabled == true {
				Expect(vmi.Spec.Domain.Features.Hyperv.VAPIC).ToNot(BeNil(), "vapic should not be nil")
				Expect(vmi.Spec.Domain.CPU.Features).To(HaveLen(1), "cpu topology has to contain 1 feature")
				Expect(vmi.Spec.Domain.CPU.Features[0].Name).To(Equal(nodelabellerutil.VmxFeature), "vmx cpu feature should be requested")
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(v1.CPUModelVendorLabel+"Intel", "true"))
			} else {
				Expect(pod.Spec.NodeSelector).ShouldNot(HaveKeyWithValue(v1.CPUModelVendorLabel+"Intel", "true"))
				Expect(vmi.Spec.Domain.Features.Hyperv.VAPIC).To(BeNil(), "vapic should be nil")
				Expect(vmi.Spec.Domain.CPU.Features).To(BeEmpty())
			}
		},
			Entry("hyperv and cpu features should be auto filled when EVMCS is enabled", decorators.VMX, &v1.FeatureState{Enabled: virtpointer.P(true)}),
			Entry("EVMCS should be enabled when vmi.Spec.Domain.Features.Hyperv.EVMCS is set but the EVMCS.Enabled field is nil ", decorators.VMX, &v1.FeatureState{Enabled: nil}),
			Entry("Verify that features aren't applied when enabled is false", &v1.FeatureState{Enabled: virtpointer.P(false)}),
		)
	})

	Context("VMI with HyperV passthrough", func() {
		It("should be usable and non-migratable", func() {
			vmi := libvmifact.NewCirros(withHypervPassthrough())
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)

			domSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec.Features.Hyperv.Mode).To(Equal(api.HypervModePassthrough))

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
