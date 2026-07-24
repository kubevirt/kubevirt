package regressions_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/envtest/framework"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

// Bug #16071: Stale status.preferenceRef after removing spec.preference
//
// Removing spec.preference left status.preferenceRef populated. The
// instancetype controller's Clear() now clears stale status refs when
// the corresponding spec matchers are nil.
var _ = Describe("Bug #16071", func() {
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

	It("should clear status.preferenceRef when spec.preference is removed", func() {
		By("creating a VirtualMachinePreference")
		pref := &instancetypev1beta1.VirtualMachinePreference{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-preference",
				Namespace: "default",
			},
			Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{},
		}
		_, err := f.VirtClient().VirtualMachinePreference("default").Create(ctx, pref, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("creating a halted VM with spec.preference referencing the preference")
		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithResourceMemory("128Mi"),
			),
			libvmi.WithPreference("test-preference"),
		)
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("waiting for status.preferenceRef to be populated by the instancetype controller")
		Eventually(matcher.ThisVM(vm), 30*time.Second, 100*time.Millisecond).Should(
			matcher.HavePreferenceControllerRevisionRef())

		By("removing spec.preference from the VM")
		Eventually(func() error {
			vm, err = f.VirtClient().VirtualMachine("default").Get(ctx, vm.Name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			vm.Spec.Preference = nil
			_, err = f.VirtClient().VirtualMachine("default").Update(ctx, vm, metav1.UpdateOptions{})
			return err
		}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

		By("verifying status.preferenceRef is cleared by the instancetype controller")
		Eventually(matcher.ThisVM(vm), 10*time.Second, 100*time.Millisecond).ShouldNot(
			matcher.HavePreferenceControllerRevisionRef())
	})
})
