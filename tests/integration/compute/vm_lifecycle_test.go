package compute_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

var _ = Describe("VM Lifecycle", func() {
	var f *framework.Framework

	BeforeEach(func() {
		f = framework.New()
		f.Start(context.Background())
	})

	It("should create a VMI and pod when a VM is created with RunStrategyAlways", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)

		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)

		var err error
		vm, err = f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the VM controller to create a VMI")
		Eventually(ctx, matcher.ThisVMIWith(ns, vm.Name), 10*time.Second, 100*time.Millisecond).Should(matcher.Exist())

		By("waiting for the VMI controller to create a virt-launcher pod")
		Eventually(ctx, func(ctx context.Context) ([]k8sv1.Pod, error) {
			pods, err := f.K8sClient().CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=virt-launcher,%s=%s", virtv1.AppLabel, virtv1.DeprecatedVirtualMachineNameLabel, vm.Name),
			})
			if err != nil {
				return nil, err
			}
			return pods.Items, nil
		}, 10*time.Second, 100*time.Millisecond).Should(HaveLen(1))

		By("waiting for the VMI to reach Scheduled phase after pod simulator makes pod Ready")
		Eventually(ctx, matcher.ThisVMIWith(ns, vm.Name), 10*time.Second, 100*time.Millisecond).Should(matcher.BeInPhase(virtv1.Scheduled))
	})

	It("keeps a VMI in Scheduling when the launcher pod is held Pending", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		f.PodSimulator().SetHook(func(_ *k8sv1.Pod) framework.PodSimulationResult {
			return framework.PodSimulationResult{Action: framework.ActionSkip}
		})
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)
		vm, err := f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(ctx, matcher.ThisVMIWith(ns, vm.Name), 30*time.Second, 100*time.Millisecond).
			Should(matcher.BeInPhase(virtv1.Scheduling))
		Consistently(ctx, matcher.ThisVMIWith(ns, vm.Name), 3*time.Second, 200*time.Millisecond).
			Should(matcher.BeInPhase(virtv1.Scheduling))
	})

	It("transitions a VMI to Failed when the launcher pod fails", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		f.PodSimulator().SetHook(func(_ *k8sv1.Pod) framework.PodSimulationResult {
			return framework.PodSimulationResult{
				Action: framework.ActionFail, Reason: "OOMKilled", Message: "container exceeded memory limit",
			}
		})
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyOnce),
		)
		vm, err := f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(ctx, matcher.ThisVMIWith(ns, vm.Name), 30*time.Second, 100*time.Millisecond).
			Should(matcher.BeInPhase(virtv1.Failed))
		pods, err := f.K8sClient().CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "kubevirt.io=virt-launcher"})
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).To(HaveLen(1))
		Expect(pods.Items[0].Status.Phase).To(Equal(k8sv1.PodFailed))
	})

	It("reaches Scheduled when the launcher pod is running without readiness", func(ctx SpecContext) {
		ns := f.CreateNamespace(ctx)
		f.PodSimulator().SetHook(func(_ *k8sv1.Pod) framework.PodSimulationResult {
			return framework.PodSimulationResult{Action: framework.ActionNotReady}
		})
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)
		vm, err := f.VirtClient().VirtualMachine(ns).Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(ctx, matcher.ThisVMIWith(ns, vm.Name), 30*time.Second, 100*time.Millisecond).
			Should(matcher.BeInPhase(virtv1.Scheduled))
		pods, err := f.K8sClient().CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "kubevirt.io=virt-launcher"})
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).To(HaveLen(1))
		Expect(pods.Items[0].Status.Phase).To(Equal(k8sv1.PodRunning))
		Expect(pods.Items[0].Status.Conditions).To(ContainElement(k8sv1.PodCondition{
			Type: k8sv1.PodReady, Status: k8sv1.ConditionFalse,
		}))
	})
})
