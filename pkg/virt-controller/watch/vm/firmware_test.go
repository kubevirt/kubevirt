/*
Copyright The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vm

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("VM Firmware Controller", func() {
	It("sync does nothing when a vm already has a uuid", func() {
		clientset := fake.NewSimpleClientset()
		fc := NewFirmwareController(clientset)
		vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithFirmwareUUID("some-existing-uid")))
		originalVM := vm.DeepCopy()
		updatedVM, err := fc.Sync(vm, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(updatedVM).To(Equal(originalVM))
		Expect(clientset.Actions()).To(BeEmpty())
	})

	It("sync fails when VM patch returns an error", func() {
		clientset := fake.NewSimpleClientset()
		fc := NewFirmwareController(clientset)

		injectedPatchError := errors.New("test patch error")
		clientset.Fake.PrependReactor(
			"patch",
			"virtualmachines",
			func(action testing.Action) (handled bool, obj k8sruntime.Object, err error) {
				return true, nil, injectedPatchError
			})

		vm := libvmi.NewVirtualMachine(libvmi.New())
		vm.Spec.Template.Spec.Domain.Firmware = nil

		_, err := clientset.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := fc.Sync(vm, nil)
		Expect(vm).To(Equal(originalVM))
		Expect(err).To(MatchError(ContainSubstring(injectedPatchError.Error())))
		Expect(updatedVM).To(Equal(originalVM))
	})

	DescribeTable("sync succeeds to patch firmware UUID", func(firmware *v1.Firmware) {
		clientset := fake.NewSimpleClientset()
		c := NewFirmwareController(clientset)

		vm := libvmi.NewVirtualMachine(libvmi.New())

		vm.Spec.Template.Spec.Domain.Firmware = firmware

		_, err := clientset.KubevirtV1().VirtualMachines(vm.Namespace).Create(context.Background(), vm, k8smetav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		originalVM := vm.DeepCopy()
		updatedVM, err := c.Sync(vm, nil)
		Expect(vm).To(Equal(originalVM))
		Expect(err).NotTo(HaveOccurred())

		legacyUUID := CalculateLegacyUUID(vm.Name)
		expectedVM := originalVM.DeepCopy()
		expectedVM.Spec.Template.Spec.Domain.Firmware = &v1.Firmware{UUID: legacyUUID}
		Expect(updatedVM).To(Equal(expectedVM))

		updatedServerVM, err := clientset.KubevirtV1().VirtualMachines(vm.Namespace).Get(context.Background(), vm.Name, k8smetav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updatedServerVM.Spec.Template.Spec.Domain.Firmware).To(Equal(expectedVM.Spec.Template.Spec.Domain.Firmware))
	},
		Entry("when the VM has no firmware", nil),
		Entry("when the VM has firmware with an empty UUID", &v1.Firmware{UUID: ""}),
	)
})
