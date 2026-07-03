package regressions_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/envtest/framework"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

// Bug #16719: RestartRequired condition never cleared after spec revert
//
// The VM controller set RestartRequired when non-live-updatable fields
// differed from the ControllerRevision spec but never cleared it when
// specs matched again, permanently blocking hotplug.
var _ = Describe("Bug #16719", func() {
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
		Eventually(matcher.ThisVMIWith("default", vm.Name), 10*time.Second, 100*time.Millisecond).Should(matcher.BeInPhase(virtv1.Scheduled))

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
		Eventually(matcher.ThisVMWith("default", vm.Name), 10*time.Second, 100*time.Millisecond).Should(
			matcher.HaveConditionTrue(virtv1.VirtualMachineRestartRequired))

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
		Eventually(matcher.ThisVMWith("default", vm.Name), 10*time.Second, 100*time.Millisecond).Should(
			matcher.HaveConditionMissingOrFalse(virtv1.VirtualMachineRestartRequired))
	})
})
