package integration_test

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
)

var _ = Describe("VM Lifecycle", func() {
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
			pods, err := f.VirtClient().CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=virt-launcher,%s=%s", virtv1.AppLabel, virtv1.DeprecatedVirtualMachineNameLabel, vm.Name),
			})
			if err != nil {
				return nil, err
			}
			return pods.Items, err
		}, 10*time.Second, 100*time.Millisecond).Should(HaveLen(1))

		By("waiting for the VMI to reach Scheduled phase after pod simulator makes pod Ready")
		Eventually(ctx, matcher.ThisVMIWith(ns, vm.Name), 10*time.Second, 100*time.Millisecond).Should(matcher.BeInPhase(virtv1.Scheduled))
	})
})
