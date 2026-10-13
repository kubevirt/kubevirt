package compute_test

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

var _ = Describe("VM Generation Tracking", Ordered, ContinueOnFailure, func() {
	var f *framework.Framework

	BeforeAll(func() {
		f = framework.New()
		f.Start(context.Background())
	})

	It("should track generations and ControllerRevisions across VM and VMI", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		By("creating a halted VM")
		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithHostname("original"),
			),
		)
		var err error
		vm, err = f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		vmName := vm.Name
		Expect(vm.Generation).To(Equal(int64(1)))

		By("starting the VM")
		vm = patchVM(ctx, f, vm, patch.WithAdd("/spec/runStrategy", virtv1.RunStrategyAlways))

		By("waiting for VMI to reach Scheduled phase")
		Eventually(ctx, matcher.ThisVMIWith(ns, vmName), 30*time.Second, 100*time.Millisecond).Should(
			matcher.BeInPhase(virtv1.Scheduled),
		)

		By("verifying generations are in sync after start")
		Eventually(ctx, func(g Gomega) {
			currentVM, getErr := f.VirtClient().VirtualMachine(ns).Get(ctx, vmName, metav1.GetOptions{})
			g.Expect(getErr).NotTo(HaveOccurred())
			g.Expect(currentVM.Status.ObservedGeneration).To(Equal(currentVM.Status.DesiredGeneration),
				"ObservedGeneration should equal DesiredGeneration after start")
			g.Expect(currentVM.Status.DesiredGeneration).To(Equal(currentVM.Generation))
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		startGeneration := vm.Generation

		By("verifying VMI has the generation annotation")
		vmi, err := f.VirtClient().VirtualMachineInstance(ns).Get(ctx, vmName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		vmiUID := vmi.UID
		Expect(vmi.Annotations).To(HaveKeyWithValue(
			virtv1.VirtualMachineGenerationAnnotation,
			strconv.FormatInt(startGeneration, 10),
		))

		By("verifying a ControllerRevision exists for the VM spec")
		revisionName := vmi.Status.VirtualMachineRevisionName
		Expect(revisionName).NotTo(BeEmpty())
		cr, err := f.K8sClient().AppsV1().ControllerRevisions(ns).Get(ctx, revisionName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cr.Revision).To(Equal(startGeneration))
		Expect(metav1.IsControlledBy(cr, vm)).To(BeTrue())

		By("updating a non-template field (RunStrategy)")
		vm = patchVM(ctx, f, vm, patch.WithAdd("/spec/runStrategy", virtv1.RunStrategyRerunOnFailure))

		Expect(vm.Generation).To(BeNumerically(">", startGeneration),
			"generation should increment after spec change")
		nonTemplateGeneration := vm.Generation

		By("verifying generations converge without requiring a restart (template unchanged)")
		Eventually(ctx, func(g Gomega) {
			currentVM, getErr := f.VirtClient().VirtualMachine(ns).Get(ctx, vmName, metav1.GetOptions{})
			g.Expect(getErr).NotTo(HaveOccurred())
			g.Expect(currentVM.Status.ObservedGeneration).To(Equal(nonTemplateGeneration),
				"ObservedGeneration should bump for non-template changes")
			g.Expect(currentVM.Status.DesiredGeneration).To(Equal(nonTemplateGeneration))
			g.Expect(currentVM).NotTo(matcher.HaveConditionTrue(virtv1.VirtualMachineRestartRequired))
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying the ControllerRevision was not recreated")
		vmi, err = f.VirtClient().VirtualMachineInstance(ns).Get(ctx, vmName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vmi.UID).To(Equal(vmiUID), "non-template changes should preserve the running VMI")
		Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(revisionName),
			"ControllerRevision should not change for non-template updates")
		currentRevision, err := f.K8sClient().AppsV1().ControllerRevisions(ns).Get(ctx, revisionName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(currentRevision.UID).To(Equal(cr.UID), "ControllerRevision should not be recreated")
		Expect(currentRevision.Data).To(Equal(cr.Data), "ControllerRevision should retain the boot-time spec")

		By("updating a template field (hostname)")
		vm = patchVM(ctx, f, vm, patch.WithAdd("/spec/template/spec/hostname", "changed"))

		Expect(vm.Generation).To(BeNumerically(">", nonTemplateGeneration),
			"generation should increment after template change")
		templateGeneration := vm.Generation

		By("waiting for DesiredGeneration and RestartRequired to reflect the template change")
		Eventually(ctx, func(g Gomega) {
			currentVM, getErr := f.VirtClient().VirtualMachine(ns).Get(ctx, vmName, metav1.GetOptions{})
			g.Expect(getErr).NotTo(HaveOccurred())
			g.Expect(currentVM.Status.DesiredGeneration).To(Equal(templateGeneration))
			g.Expect(currentVM).To(matcher.HaveConditionTrue(virtv1.VirtualMachineRestartRequired))
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying ObservedGeneration stays unchanged without a restart")
		Consistently(ctx, func(g Gomega) {
			currentVM, getErr := f.VirtClient().VirtualMachine(ns).Get(ctx, vmName, metav1.GetOptions{})
			g.Expect(getErr).NotTo(HaveOccurred())
			g.Expect(currentVM.Status.ObservedGeneration).To(Equal(nonTemplateGeneration),
				"ObservedGeneration should NOT bump when template changed without restart")
		}, 3*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying the ControllerRevision still points to the original")
		vmi, err = f.VirtClient().VirtualMachineInstance(ns).Get(ctx, vmName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vmi.UID).To(Equal(vmiUID), "hostname changes should preserve the running VMI")
		Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(revisionName),
			fmt.Sprintf("ControllerRevision should still be %s until VMI is restarted", revisionName))
		currentRevision, err = f.K8sClient().AppsV1().ControllerRevisions(ns).Get(ctx, revisionName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(currentRevision.UID).To(Equal(cr.UID), "ControllerRevision should not be recreated")
		Expect(currentRevision.Data).To(Equal(cr.Data), "ControllerRevision should retain the boot-time spec")
	})

	It("should reproduce the ObservedGeneration lag after live-updatable template changes", func(ctx SpecContext) {
		testAffinityGenerationTracking(ctx, f, virtv1.VMRolloutStrategyLiveUpdate)
	})
})

var _ = Describe("VM Generation Tracking with staged updates", func() {
	var f *framework.Framework

	BeforeEach(func() {
		f = framework.New(framework.WithKubeVirtConfig(&virtv1.KubeVirtConfiguration{
			VMRolloutStrategy: pointer.P(virtv1.VMRolloutStrategyStage),
		}))
		f.Start(context.Background())
	})

	It("should defer node affinity changes until a restart", func(ctx SpecContext) {
		testAffinityGenerationTracking(ctx, f, virtv1.VMRolloutStrategyStage)
	})
})

func testAffinityGenerationTracking(ctx SpecContext, f *framework.Framework, strategy virtv1.VMRolloutStrategy) {
	GinkgoHelper()
	ns := f.CreateNamespace(ctx)

	By("creating and starting a VM")
	vm := libvmi.NewVirtualMachine(
		libvmi.New(
			libvmi.WithMemoryRequest("128Mi"),
		),
	)
	var err error
	vm, err = f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	vmName := vm.Name

	vm = patchVM(ctx, f, vm, patch.WithAdd("/spec/runStrategy", virtv1.RunStrategyAlways))

	By("waiting for VMI to reach Scheduled phase")
	Eventually(ctx, matcher.ThisVMIWith(ns, vmName), 30*time.Second, 100*time.Millisecond).Should(
		matcher.BeInPhase(virtv1.Scheduled),
	)

	By("waiting for generations to sync after start")
	Eventually(ctx, func(g Gomega) {
		currentVM, getErr := f.VirtClient().VirtualMachine(ns).Get(ctx, vmName, metav1.GetOptions{})
		g.Expect(getErr).NotTo(HaveOccurred())
		g.Expect(currentVM.Status.ObservedGeneration).To(Equal(currentVM.Generation))
		g.Expect(currentVM.Status.DesiredGeneration).To(Equal(currentVM.Generation))
	}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

	preChangeGeneration := vm.Generation
	vmi, err := f.VirtClient().VirtualMachineInstance(ns).Get(ctx, vmName, metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	vmiUID := vmi.UID

	By("updating a live-updatable template field (node affinity)")
	affinity := &k8sv1.Affinity{
		NodeAffinity: &k8sv1.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []k8sv1.PreferredSchedulingTerm{{
				Weight: 1,
				Preference: k8sv1.NodeSelectorTerm{
					MatchExpressions: []k8sv1.NodeSelectorRequirement{{
						Key:      "topology.kubernetes.io/zone",
						Operator: k8sv1.NodeSelectorOpIn,
						Values:   []string{"us-east-1a"},
					}},
				},
			}},
		},
	}
	vm = patchVM(ctx, f, vm, patch.WithAdd("/spec/template/spec/affinity", affinity))

	Expect(vm.Generation).To(BeNumerically(">", preChangeGeneration))
	affinityGeneration := vm.Generation

	expectedAffinity := vmi.Spec.Affinity
	expectedRestartRequired := matcher.HaveConditionTrue(virtv1.VirtualMachineRestartRequired)
	if strategy == virtv1.VMRolloutStrategyLiveUpdate {
		expectedAffinity = affinity
		expectedRestartRequired = Not(expectedRestartRequired)
	}

	By("waiting for the controller to process the affinity change")
	Eventually(ctx, func(g Gomega) {
		currentVM, getErr := f.VirtClient().VirtualMachine(ns).Get(ctx, vmName, metav1.GetOptions{})
		g.Expect(getErr).NotTo(HaveOccurred())
		g.Expect(currentVM.Status.DesiredGeneration).To(Equal(affinityGeneration))
		g.Expect(currentVM).To(expectedRestartRequired)
	}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

	By("verifying the VMI affinity reflects the rollout strategy")
	Eventually(ctx, func(g Gomega) {
		currentVMI, getErr := f.VirtClient().VirtualMachineInstance(ns).Get(ctx, vmName, metav1.GetOptions{})
		g.Expect(getErr).NotTo(HaveOccurred())
		g.Expect(currentVMI.UID).To(Equal(vmiUID))
		g.Expect(currentVMI.Spec.Affinity).To(Equal(expectedAffinity))
	}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

	// FIXME(#18700): With LiveUpdate, ObservedGeneration should equal
	// affinityGeneration after the change reaches the VMI. It remains at
	// preChangeGeneration because conditionallyBumpGenerationAnnotationOnVmi
	// compares the template with the ControllerRevision's old affinity.
	// Update the LiveUpdate expectation once the bug is fixed. With Stage,
	// the unchanged ObservedGeneration is expected until the VM restarts.
	By("verifying ObservedGeneration, RestartRequired and the VMI affinity remain stable")
	Consistently(ctx, func(g Gomega) {
		currentVM, getErr := f.VirtClient().VirtualMachine(ns).Get(ctx, vmName, metav1.GetOptions{})
		g.Expect(getErr).NotTo(HaveOccurred())
		g.Expect(currentVM.Status.ObservedGeneration).To(Equal(preChangeGeneration))
		g.Expect(currentVM).To(expectedRestartRequired)
		currentVMI, getErr := f.VirtClient().VirtualMachineInstance(ns).Get(ctx, vmName, metav1.GetOptions{})
		g.Expect(getErr).NotTo(HaveOccurred())
		g.Expect(currentVMI.UID).To(Equal(vmiUID))
		g.Expect(currentVMI.Spec.Affinity).To(Equal(expectedAffinity))
	}, 3*time.Second, 100*time.Millisecond).Should(Succeed())
}

func patchVM(ctx context.Context, f *framework.Framework, vm *virtv1.VirtualMachine, opts ...patch.PatchOption) *virtv1.VirtualMachine {
	GinkgoHelper()
	payload, err := patch.New(opts...).GeneratePayload()
	Expect(err).NotTo(HaveOccurred())
	updatedVM, err := f.VirtClient().VirtualMachine(vm.Namespace).Patch(ctx, vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
	Expect(err).NotTo(HaveOccurred())
	return updatedVM
}
