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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/libvmi"
)

var _ = Describe("VM Firmware Controller", func() {
	Context("Synchronizer Contract Compliance", func() {
		It("should not make any API calls when VM already has UUID", func() {
			clientset := fake.NewSimpleClientset()
			fc := NewFirmwareController(clientset)
			vm := libvmi.NewVirtualMachine(libvmi.New(libvmi.WithFirmwareUUID("some-existing-uid")))
			originalVM := vm.DeepCopy()

			updatedVM, err := fc.Sync(vm, nil)

			Expect(err).ToNot(HaveOccurred())
			Expect(updatedVM).To(Equal(originalVM))
			// Verify synchronizer contract: no API calls made
			Expect(clientset.Actions()).To(BeEmpty())
		})

		DescribeTable("should only modify local VM object without API calls", func(firmware *v1.Firmware) {
			clientset := fake.NewSimpleClientset()
			fc := NewFirmwareController(clientset)
			vm := libvmi.NewVirtualMachine(libvmi.New())
			vm.Spec.Template.Spec.Domain.Firmware = firmware
			originalVM := vm.DeepCopy()

			updatedVM, err := fc.Sync(vm, nil)

			Expect(err).NotTo(HaveOccurred())
			// Verify input VM was not modified (synchronizer should return a copy)
			Expect(vm).To(Equal(originalVM))

			// Verify returned VM has UUID set
			legacyUUID := CalculateLegacyUUID(vm.Name)
			Expect(updatedVM.Spec.Template.Spec.Domain.Firmware).ToNot(BeNil())
			Expect(updatedVM.Spec.Template.Spec.Domain.Firmware.UUID).To(Equal(legacyUUID))

			// Verify synchronizer contract: no API calls made
			// VM controller is responsible for persisting changes via Update()
			Expect(clientset.Actions()).To(BeEmpty())
		},
			Entry("when the VM has no firmware", nil),
			Entry("when the VM has firmware with an empty UUID", &v1.Firmware{UUID: ""}),
		)
	})
})
