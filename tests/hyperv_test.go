package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	"kubevirt.io/kubevirt/tests/libinfra"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/testsuite"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[Serial][sig-compute] Hyper-V enlightenments", Serial, decorators.SigCompute, func() {

	var (
		virtClient kubecli.KubevirtClient
	)
	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("VMI with HyperV re-enlightenment enabled", func() {
		var reEnlightenmentVMI *v1.VirtualMachineInstance

		withReEnlightenment := func(vmi *v1.VirtualMachineInstance) {
			if vmi.Spec.Domain.Features == nil {
				vmi.Spec.Domain.Features = &v1.Features{}
			}
			if vmi.Spec.Domain.Features.Hyperv == nil {
				vmi.Spec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
			}

			vmi.Spec.Domain.Features.Hyperv.Reenlightenment = &v1.FeatureState{Enabled: pointer.Bool(true)}
		}

		vmiWithReEnlightenment := func() *v1.VirtualMachineInstance {
			options := libvmi.WithMasqueradeNetworking()
			options = append(options, withReEnlightenment)
			return libvmi.NewAlpine(options...)
		}

		BeforeEach(func() {
			reEnlightenmentVMI = vmiWithReEnlightenment()
		})

		When("TSC frequency is exposed on the cluster", func() {
			BeforeEach(func() {
				if !isTSCFrequencyExposed(virtClient) {
					Skip("TSC frequency is not exposed on the cluster")
				}
			})

			It("should be able to migrate", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				reEnlightenmentVMI = libwait.WaitForSuccessfulVMIStart(reEnlightenmentVMI)

				By("Migrating the VM")
				migration := tests.NewRandomMigration(reEnlightenmentVMI.Name, reEnlightenmentVMI.Namespace)
				migrationUID := libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				By("Checking VMI, confirm migration state")
				libmigration.ConfirmVMIPostMigration(virtClient, reEnlightenmentVMI, migrationUID)
			})

			It("should have TSC frequency set up in label and domain", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				reEnlightenmentVMI = libwait.WaitForSuccessfulVMIStart(reEnlightenmentVMI)

				virtLauncherPod := tests.GetPodByVirtualMachineInstance(reEnlightenmentVMI)

				foundNodeSelector := false
				for key, _ := range virtLauncherPod.Spec.NodeSelector {
					if strings.HasPrefix(key, topology.TSCFrequencySchedulingLabel+"-") {
						foundNodeSelector = true
						break
					}
				}
				Expect(foundNodeSelector).To(BeTrue(), "wasn't able to find a node selector key with prefix ", topology.TSCFrequencySchedulingLabel)

				domainSpec, err := tests.GetRunningVMIDomainSpec(reEnlightenmentVMI)
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

		When("TSC frequency is not exposed on the cluster", decorators.Reenlightenment, decorators.TscFrequencies, func() {

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

			It("should be able to start successfully", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(reEnlightenmentVMI)
				Expect(console.LoginToAlpine(reEnlightenmentVMI)).To(Succeed())
			})

			It("should be marked as non-migratable", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(reEnlightenmentVMI)

				conditionManager := controller.NewVirtualMachineInstanceConditionManager()
				isNonMigratable := func() error {
					reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(context.Background(), reEnlightenmentVMI.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					cond := conditionManager.GetCondition(reEnlightenmentVMI, v1.VirtualMachineInstanceIsMigratable)
					const errFmt = "condition " + string(v1.VirtualMachineInstanceIsMigratable) + " is expected to be %s %s"

					if statusFalse := k8sv1.ConditionFalse; cond.Status != statusFalse {
						return fmt.Errorf(errFmt, "of status", string(statusFalse))
					}
					if notMigratableNoTscReason := v1.VirtualMachineInstanceReasonNoTSCFrequencyMigratable; cond.Reason != notMigratableNoTscReason {
						return fmt.Errorf(errFmt, "of reason", notMigratableNoTscReason)
					}
					if !strings.Contains(cond.Message, "HyperV Reenlightenment") {
						return fmt.Errorf(errFmt, "with message that contains", "HyperV Reenlightenment")
					}
					return nil
				}

				Eventually(isNonMigratable, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
				Consistently(isNonMigratable, 15*time.Second, 3*time.Second).ShouldNot(HaveOccurred())
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
			checks.SkipIfARM64(testsuite.Arch, "arm64 does not support cpu model")
			nodes := libnode.GetAllSchedulableNodes(virtClient)
			Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
			node := &nodes.Items[0]
			supportedCPUs := tests.GetSupportedCPUModels(*nodes)
			Expect(supportedCPUs).ToNot(BeEmpty(), "There should be some supported cpu models")

			for key := range node.Labels {
				if strings.Contains(key, services.NFD_KVM_INFO_PREFIX) &&
					!strings.Contains(key, "tlbflush") &&
					!strings.Contains(key, "ipi") &&
					!strings.Contains(key, "synictimer") {
					supportedKVMInfoFeature = append(supportedKVMInfoFeature, strings.TrimPrefix(key, services.NFD_KVM_INFO_PREFIX))
				}
			}

			for _, label := range supportedKVMInfoFeature {
				vmi := libvmi.NewCirros()
				features := enableHyperVInVMI(label)
				vmi.Spec.Domain.Features = &v1.Features{
					Hyperv: &features,
				}

				vmi, err := virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred(), "Should create VMI when using %v", label)
				libwait.WaitForSuccessfulVMIStart(vmi)

				_, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "Should get VMI when using %v", label)
			}
		})

		DescribeTable("the vmi with EVMCS HyperV feature should have correct HyperV and cpu features auto filled", func(featureState *v1.FeatureState) {
			tests.EnableFeatureGate(virtconfig.HypervStrictCheckGate)
			vmi := libvmi.NewCirros()
			vmi.Spec.Domain.Features = &v1.Features{
				Hyperv: &v1.FeatureHyperv{
					EVMCS: featureState,
				},
			}
			vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

			var err error
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should get VMI")
			Expect(vmi.Spec.Domain.Features.Hyperv.EVMCS).ToNot(BeNil(), "evmcs should not be nil")
			Expect(vmi.Spec.Domain.CPU).ToNot(BeNil(), "cpu topology can't be nil")
			pod, err := libvmi.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
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
			Entry("hyperv and cpu features should be auto filled when EVMCS is enabled", decorators.VMX, &v1.FeatureState{Enabled: pointer.BoolPtr(true)}),
			Entry("EVMCS should be enabled when vmi.Spec.Domain.Features.Hyperv.EVMCS is set but the EVMCS.Enabled field is nil ", decorators.VMX, &v1.FeatureState{Enabled: nil}),
			Entry("Verify that features aren't applied when enabled is false", &v1.FeatureState{Enabled: pointer.BoolPtr(false)}),
		)
	})
})
