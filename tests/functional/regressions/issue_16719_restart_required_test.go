package regressions_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/functional/framework"
)

// Issue #16719: Sync RestartRequired condition removal during reconciliation loop
// Fix: PR #16606 (commit 6fbd89c579)
//
// The VM controller's syncRestartRequired sets the RestartRequired condition
// when non-live-updatable fields in the VM spec differ from the spec stored
// in the ControllerRevision at VMI creation time. However, it never cleared
// the condition when the specs matched again. This caused RestartRequired to
// persist indefinitely, blocking hotplug operations for CPU, memory, and
// volumes even when no restart was actually needed.
var _ = Describe("Issue #16719", func() {
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

	It("should clear RestartRequired when VM spec matches the original again", func() {
		By("creating a VM with a specific hostname")
		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithResourceMemory("128Mi"),
				libvmi.WithHostname("original"),
			),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)

		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for VMI to reach Scheduled phase")
		Eventually(func() virtv1.VirtualMachineInstancePhase {
			vmis, err := f.VirtClient().VirtualMachineInstance("default").List(ctx, metav1.ListOptions{})
			if err != nil || len(vmis.Items) == 0 {
				return ""
			}
			return vmis.Items[0].Status.Phase
		}, 10*time.Second, 100*time.Millisecond).Should(Equal(virtv1.Scheduled))

		By("changing hostname to a different value (non-live-updatable field)")
		Eventually(func() error {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.Template.Spec.Hostname = "changed"
			_, err = f.VirtClient().VirtualMachine("default").Update(ctx, vm, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("waiting for RestartRequired condition to be set")
		conditionManager := controller.NewVirtualMachineConditionManager()
		Eventually(func() bool {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			return conditionManager.HasCondition(vm, virtv1.VirtualMachineRestartRequired)
		}, 10*time.Second, 100*time.Millisecond).Should(BeTrue(),
			"RestartRequired should be set after changing a non-live-updatable field")

		By("restoring hostname to original value")
		Eventually(func() error {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.Template.Spec.Hostname = "original"
			_, err = f.VirtClient().VirtualMachine("default").Update(ctx, vm, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying RestartRequired condition is cleared")
		Eventually(func() bool {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return true
			}
			return conditionManager.HasCondition(vm, virtv1.VirtualMachineRestartRequired)
		}, 10*time.Second, 100*time.Millisecond).Should(BeFalse(),
			"RestartRequired should be cleared when spec matches the original")
	})
})
