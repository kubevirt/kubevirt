package envtest_test

import (
	"context"
	"fmt"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
})
