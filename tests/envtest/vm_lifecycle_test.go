package envtest_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/envtest/framework"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

var _ = Describe("VM Lifecycle", func() {
	var f *framework.Framework
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New()
		f.Start()
		DeferCleanup(f.Stop)
	})

	It("should create a VMI and pod when a VM is created with RunStrategyAlways", func() {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithMemoryRequest("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)

		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the VM controller to create a VMI")
		Eventually(matcher.ThisVMIWith("default", vm.Name), 10*time.Second, 100*time.Millisecond).Should(matcher.Exist())

		By("waiting for the VMI controller to create a virt-launcher pod")
		Eventually(func() int {
			pods, err := f.K8sClient().CoreV1().Pods("default").List(ctx, metav1.ListOptions{
				LabelSelector: "kubevirt.io=virt-launcher",
			})
			if err != nil {
				return 0
			}
			return len(pods.Items)
		}, 10*time.Second, 100*time.Millisecond).Should(Equal(1))

		By("waiting for the VMI to reach Scheduled phase after pod simulator makes pod Ready")
		Eventually(matcher.ThisVMIWith("default", vm.Name), 10*time.Second, 100*time.Millisecond).Should(matcher.BeInPhase(virtv1.Scheduled))
	})

	It("should keep VMI in Scheduling phase when pod simulator holds launcher pod in Pending phase", func() {
		f.PodSimulator().SetHook(func(pod *k8sv1.Pod) framework.PodSimulationResult {
			return framework.PodSimulationResult{
				Action: framework.ActionSkip,
			}
		})

		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)

		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the VM controller to create a VMI")
		Eventually(matcher.ThisVMIWith("default", vm.Name), 10*time.Second, 100*time.Millisecond).Should(matcher.Exist())

		By("verifying the VMI stays in Scheduling phase because launcher pod is not bound")
		Consistently(matcher.ThisVMIWith("default", vm.Name), 3*time.Second, 200*time.Millisecond).Should(matcher.BeInPhase(virtv1.Scheduling))
	})

	It("should transition VMI to Failed phase when launcher pod fails", func() {
		f.PodSimulator().SetHook(func(pod *k8sv1.Pod) framework.PodSimulationResult {
			return framework.PodSimulationResult{
				Action:  framework.ActionFail,
				Reason:  "OOMKilled",
				Message: "container exceeded memory limit",
			}
		})

		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyOnce),
		)

		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for VMI to reach Failed phase")
		Eventually(matcher.ThisVMIWith("default", vm.Name), 30*time.Second, 100*time.Millisecond).Should(matcher.BeInPhase(virtv1.Failed))

		By("verifying the launcher pod has PodFailed phase")
		pods, err := f.K8sClient().CoreV1().Pods("default").List(ctx, metav1.ListOptions{
			LabelSelector: "kubevirt.io=virt-launcher",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).To(HaveLen(1))
		Expect(pods.Items[0].Status.Phase).To(Equal(k8sv1.PodFailed))
	})

	It("should reach Scheduled phase when launcher pod is running but not ready", func() {
		f.PodSimulator().SetHook(func(pod *k8sv1.Pod) framework.PodSimulationResult {
			return framework.PodSimulationResult{
				Action: framework.ActionNotReady,
			}
		})

		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)

		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the VMI to reach Scheduled (pod is bound and running)")
		Eventually(matcher.ThisVMIWith("default", vm.Name), 30*time.Second, 100*time.Millisecond).Should(matcher.BeInPhase(virtv1.Scheduled))

		By("verifying the launcher pod is running but not ready")
		pods, err := f.K8sClient().CoreV1().Pods("default").List(ctx, metav1.ListOptions{
			LabelSelector: "kubevirt.io=virt-launcher",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(pods.Items).To(HaveLen(1))
		Expect(pods.Items[0].Status.Phase).To(Equal(k8sv1.PodRunning))
		readyCond := false
		for _, c := range pods.Items[0].Status.Conditions {
			if c.Type == k8sv1.PodReady {
				readyCond = c.Status == k8sv1.ConditionFalse
			}
		}
		Expect(readyCond).To(BeTrue(), "pod PodReady condition should be False")
	})
})
