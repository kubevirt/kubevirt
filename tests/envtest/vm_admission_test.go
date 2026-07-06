package envtest_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/envtest/framework"
)

var _ = Describe("VM Admission", func() {
	var f *framework.Framework
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		f = framework.New(framework.WithWebhooks())
		f.Start()
	})

	AfterEach(func() {
		f.Stop()
	})

	It("should mutate a VM with defaults on creation", func() {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
		)
		Expect(vm.Spec.Template.Spec.Domain.Firmware).To(BeNil(),
			"firmware should not be set before admission")

		var err error
		vm, err = f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		By("verifying the mutating webhook set firmware defaults")
		Expect(vm.Spec.Template.Spec.Domain.Firmware).NotTo(BeNil())
		Expect(vm.Spec.Template.Spec.Domain.Firmware.UUID).NotTo(BeEmpty(),
			"firmware UUID should be defaulted by the mutating webhook")
	})

	It("should reject a VM with an invalid disk", func() {
		vm := libvmi.NewVirtualMachine(
			libvmi.New(libvmi.WithResourceMemory("128Mi")),
		)
		vm.Spec.Template.Spec.Domain.Devices.Disks = []virtv1.Disk{
			{
				Name: "missing-volume",
				DiskDevice: virtv1.DiskDevice{
					Disk: &virtv1.DiskTarget{Bus: virtv1.DiskBusVirtio},
				},
			},
		}

		_, err := f.VirtClient().VirtualMachine("default").Create(ctx, vm, metav1.CreateOptions{})
		Expect(err).To(HaveOccurred())
		Expect(errors.IsInvalid(err) || errors.IsForbidden(err)).To(BeTrue(),
			"expected the validating webhook to reject the VM")
	})
})
