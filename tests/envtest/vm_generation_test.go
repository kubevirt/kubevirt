package envtest_test

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/envtest/framework"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

var _ = Describe("VM Generation Tracking", func() {
	var f *framework.Framework
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New()
		f.Start()
	})

	AfterEach(func() {
		f.Stop()
	})

	It("should track generations and ControllerRevisions across VM and VMI", func() {
		By("creating a halted VM")
		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithResourceMemory("128Mi"),
				libvmi.WithHostname("original"),
			),
		)
		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vm.Generation).To(Equal(int64(1)))

		By("starting the VM")
		Eventually(func() error {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.RunStrategy = pointer.P(virtv1.RunStrategyAlways)
			_, err = f.VirtClient().VirtualMachine("default").Update(ctx, vm, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("waiting for VMI to reach Scheduled phase")
		Eventually(matcher.ThisVMIWith("default", vm.Name), 30*time.Second, 100*time.Millisecond).Should(
			matcher.BeInPhase(virtv1.Scheduled))

		By("verifying generations are in sync after start")
		Eventually(func(g Gomega) {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(vm.Status.ObservedGeneration).To(Equal(vm.Status.DesiredGeneration),
				"ObservedGeneration should equal DesiredGeneration after start")
			g.Expect(vm.Status.DesiredGeneration).To(Equal(vm.Generation))
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		startGeneration := vm.Generation

		By("verifying VMI has the generation annotation")
		vmi, err := f.VirtClient().VirtualMachineInstance("default").Get(ctx, vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vmi.Annotations).To(HaveKeyWithValue(
			virtv1.VirtualMachineGenerationAnnotation,
			strconv.FormatInt(startGeneration, 10)))

		By("verifying a ControllerRevision exists for the VM spec")
		revisionName := vmi.Status.VirtualMachineRevisionName
		Expect(revisionName).NotTo(BeEmpty())
		cr, err := f.K8sClient().AppsV1().ControllerRevisions("default").Get(ctx, revisionName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cr.Revision).To(Equal(startGeneration))
		Expect(metav1.IsControlledBy(cr, vm)).To(BeTrue())

		By("updating a non-template field (RunStrategy)")
		Eventually(func() error {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.RunStrategy = pointer.P(virtv1.RunStrategyRerunOnFailure)
			_, err = f.VirtClient().VirtualMachine("default").Update(ctx, vm, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vm.Generation).To(BeNumerically(">", startGeneration),
			"generation should increment after spec change")
		nonTemplateGeneration := vm.Generation

		By("verifying ObservedGeneration catches up (template unchanged)")
		Eventually(func(g Gomega) {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(vm.Status.ObservedGeneration).To(Equal(nonTemplateGeneration),
				"ObservedGeneration should bump for non-template changes")
			g.Expect(vm.Status.DesiredGeneration).To(Equal(nonTemplateGeneration))
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying the ControllerRevision was NOT recreated")
		vmi, err = f.VirtClient().VirtualMachineInstance("default").Get(ctx, vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(revisionName),
			"ControllerRevision should not change for non-template updates")

		By("updating a template field (hostname)")
		Eventually(func() error {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.Template.Spec.Hostname = "changed"
			_, err = f.VirtClient().VirtualMachine("default").Update(ctx, vm, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vm.Generation).To(BeNumerically(">", nonTemplateGeneration),
			"generation should increment after template change")
		templateGeneration := vm.Generation

		By("verifying ObservedGeneration does NOT catch up (template changed)")
		Eventually(func(g Gomega) {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(vm.Status.DesiredGeneration).To(Equal(templateGeneration))
			g.Expect(vm.Status.ObservedGeneration).To(Equal(nonTemplateGeneration),
				"ObservedGeneration should NOT bump when template changed without restart")
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying the ControllerRevision still points to the original")
		vmi, err = f.VirtClient().VirtualMachineInstance("default").Get(ctx, vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vmi.Status.VirtualMachineRevisionName).To(Equal(revisionName),
			fmt.Sprintf("ControllerRevision should still be %s until VMI is restarted", revisionName))
	})

	// Regression test for https://github.com/kubevirt/kubevirt/issues/18700
	//
	// FIXME: ObservedGeneration should converge after a live-updatable field
	// change (e.g. affinity) because the controller applies it to the VMI.
	// It does not because conditionallyBumpGenerationAnnotationOnVmi compares
	// vm.Spec.Template against the ControllerRevision (boot-time snapshot),
	// which still has the old values. Flip the final assertion once #18700 is
	// fixed.
	It("should track ObservedGeneration after live-updatable template changes", func() {
		f = framework.New(framework.WithKubeVirtConfig(&virtv1.KubeVirtConfiguration{
			VMRolloutStrategy: pointer.P(virtv1.VMRolloutStrategyLiveUpdate),
		}))
		f.Start()

		By("creating and starting a VM")
		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithResourceMemory("128Mi"),
			),
		)
		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.RunStrategy = pointer.P(virtv1.RunStrategyAlways)
			_, err = f.VirtClient().VirtualMachine("default").Update(ctx, vm, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("waiting for VMI to reach Scheduled phase")
		Eventually(matcher.ThisVMIWith("default", vm.Name), 30*time.Second, 100*time.Millisecond).Should(
			matcher.BeInPhase(virtv1.Scheduled))

		By("waiting for generations to sync after start")
		Eventually(func(g Gomega) {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(vm.Status.ObservedGeneration).To(Equal(vm.Status.DesiredGeneration))
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		preChangeGeneration := vm.Generation

		By("updating a live-updatable template field (node affinity)")
		Eventually(func() error {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.Template.Spec.Affinity = &k8sv1.Affinity{
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
			_, err = f.VirtClient().VirtualMachine("default").Update(ctx, vm, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(vm.Generation).To(BeNumerically(">", preChangeGeneration))
		affinityGeneration := vm.Generation

		By("verifying the controller applied the affinity change to the VMI")
		Eventually(func(g Gomega) {
			vmi, err := f.VirtClient().VirtualMachineInstance("default").Get(ctx, vm.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(vmi.Spec.Affinity).NotTo(BeNil(),
				"controller should live-update the VMI affinity")
			g.Expect(vmi.Spec.Affinity.NodeAffinity).NotTo(BeNil())
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying DesiredGeneration reflects the new generation")
		Eventually(func(g Gomega) {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(vm.Status.DesiredGeneration).To(Equal(affinityGeneration))
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		// FIXME(#18700): ObservedGeneration should equal affinityGeneration here
		// because the change was successfully applied to the VMI. It remains stuck
		// at preChangeGeneration because the template-equality check in
		// conditionallyBumpGenerationAnnotationOnVmi fails — the ControllerRevision
		// still has the old (nil) affinity.
		By("verifying ObservedGeneration does NOT converge (bug #18700)")
		Consistently(func(g Gomega) {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(vm.Status.ObservedGeneration).To(Equal(preChangeGeneration),
				"ObservedGeneration is stuck at the pre-change value due to bug #18700")
		}, 3*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying RestartRequired is NOT set (live-updatable field)")
		vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		conditionManager := matcher.ThisVM(vm)
		Expect(conditionManager()).NotTo(matcher.HaveConditionTrue(virtv1.VirtualMachineRestartRequired),
			"RestartRequired should not be set for a live-updatable field change")
	})
})
