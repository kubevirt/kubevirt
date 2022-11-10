package tests_test

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	"kubevirt.io/kubevirt/tests/util"
)

var _ = Describe("[Serial][sig-compute] Hyper-V enlightenments", Serial, func() {

	var (
		virtClient kubecli.KubevirtClient
		err        error
	)
	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
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
			It("should be able to migrate", func() {
				if !isTSCFrequencyExposed(virtClient) {
					Skip("TSC frequency is not exposed on the cluster")
				}

				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(reEnlightenmentVMI, 360)

				By("Migrating the VM")
				migration := tests.NewRandomMigration(reEnlightenmentVMI.Name, reEnlightenmentVMI.Namespace)
				migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)

				By("Checking VMI, confirm migration state")
				tests.ConfirmVMIPostMigration(virtClient, reEnlightenmentVMI, migrationUID)
			})
		})

		When("TSC frequency is not exposed on the cluster", func() {

			BeforeEach(func() {
				if isTSCFrequencyExposed(virtClient) {
					nodeList, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
					Expect(err).ToNot(HaveOccurred())

					for _, node := range nodeList.Items {
						stopNodeLabeller(node.Name, virtClient)
						removeTSCFrequencyFromNode(node)
					}
				}
			})

			AfterEach(func() {
				nodeList, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, node := range nodeList.Items {
					_, isNodeLabellerStopped := node.Annotations[v1.LabellerSkipNodeAnnotation]
					Expect(isNodeLabellerStopped).To(BeTrue())

					updatedNode := resumeNodeLabeller(node.Name, virtClient)
					_, isNodeLabellerStopped = updatedNode.Annotations[v1.LabellerSkipNodeAnnotation]
					Expect(isNodeLabellerStopped).To(BeFalse(), "after node labeller is resumed, %s annotation is expected to disappear from node", v1.LabellerSkipNodeAnnotation)
				}
			})

			It("should be able to start successfully", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(reEnlightenmentVMI, 360)
				Expect(console.LoginToAlpine(reEnlightenmentVMI)).To(Succeed())
			})

			It("should be marked as non-migratable", func() {
				var err error
				By("Creating a windows VM")
				reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(reEnlightenmentVMI)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStartWithTimeout(reEnlightenmentVMI, 360)

				conditionManager := controller.NewVirtualMachineInstanceConditionManager()
				isNonMigratable := func() error {
					reEnlightenmentVMI, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(reEnlightenmentVMI.Name, &metav1.GetOptions{})
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
	})

})
