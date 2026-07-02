package functional_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/functional/framework"
)

var _ = Describe("VM Lifecycle", func() {
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

	It("should create a VMI and pod when a VM is created with RunStrategyAlways", func() {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
			libvmi.WithRunStrategy(virtv1.RunStrategyAlways),
		)

		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the VM controller to create a VMI")
		Eventually(func() int {
			vmis, err := f.VirtClient().VirtualMachineInstance("default").List(ctx, metav1.ListOptions{})
			if err != nil {
				return 0
			}
			return len(vmis.Items)
		}, 10*time.Second, 100*time.Millisecond).Should(Equal(1))

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
		Eventually(func() virtv1.VirtualMachineInstancePhase {
			vmis, err := f.VirtClient().VirtualMachineInstance("default").List(ctx, metav1.ListOptions{})
			if err != nil || len(vmis.Items) == 0 {
				return ""
			}
			return vmis.Items[0].Status.Phase
		}, 10*time.Second, 100*time.Millisecond).Should(Equal(virtv1.Scheduled))
	})
})
