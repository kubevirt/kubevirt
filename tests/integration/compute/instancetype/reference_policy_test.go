/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package instancetype_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

func (s *instancetypeSuite) describeReferencePolicies() {
	Context("Reference policy", func() {
		var vm *virtv1.VirtualMachine
		BeforeEach(func(ctx SpecContext) {
			vm = createReferencePolicyVM(ctx, s.f, s.namespace)
		})
		It("should retain matchers and store revisions", func(ctx SpecContext) {
			waitForScheduledVMI(ctx, s.f, vm)
			var err error
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Get(ctx, vm.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(vm.Spec.Instancetype.Kind).To(Equal(instancetypeapi.SingularResourceName))
			Expect(vm.Spec.Preference.Kind).To(Equal(instancetypeapi.SingularPreferenceResourceName))
			Expect(vm).To(matcher.HaveControllerRevisionRefs())
			Expect(vm.Status.InstancetypeRef.Name).To(Equal(vm.Spec.Instancetype.Name))
			Expect(vm.Status.PreferenceRef.Name).To(Equal(vm.Spec.Preference.Name))
		})
	})
}

var _ = Describe("Instancetype expansion policies", func() {
	for _, policy := range []virtv1.InstancetypeReferencePolicy{virtv1.Expand, virtv1.ExpandAll} {
		Describe(fmt.Sprintf("Instancetype reference policy %s", policy), Ordered, ContinueOnFailure, func() {
			var f *framework.Framework
			var vm *virtv1.VirtualMachine

			BeforeAll(func() {
				f = framework.New(framework.WithKubeVirtConfig(&virtv1.KubeVirtConfiguration{
					Instancetype: &virtv1.InstancetypeConfiguration{ReferencePolicy: pointer.P(policy)},
				}))
				f.Start(context.Background())
			})
			BeforeEach(func(ctx SpecContext) {
				vm = createReferencePolicyVM(ctx, f, f.CreateNamespace(ctx))
			})
			It("should expand the instancetype and preference into the VM", func(ctx SpecContext) {
				vmi := waitForScheduledVMI(ctx, f, vm)
				var err error
				vm, err = f.VirtClient().VirtualMachine(vm.Namespace).Get(ctx, vm.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(vm.Spec.Instancetype).To(BeNil())
				Expect(vm.Spec.Preference).To(BeNil())
				Expect(vm.Status.InstancetypeRef).To(BeNil())
				Expect(vm.Status.PreferenceRef).To(BeNil())
				Expect(vm.Spec.Template.Spec.Domain.CPU.Cores).To(Equal(uint32(2)))
				Expect(vm.Spec.Template.Spec.Domain.Memory.Guest).To(Equal(pointer.P(resource.MustParse("128Mi"))))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks).To(HaveLen(1))
				Expect(vm.Spec.Template.Spec.Domain.Devices.Disks[0].Disk.Bus).To(Equal(virtv1.DiskBusVirtio))
				Expect(vmi.Spec.Domain.CPU.Cores).To(Equal(vm.Spec.Template.Spec.Domain.CPU.Cores))
			})
		})
	}
})

func createReferencePolicyVM(ctx context.Context, f *framework.Framework, namespace string) *virtv1.VirtualMachine {
	GinkgoHelper()
	it, err := f.VirtClient().VirtualMachineInstancetype(namespace).Create(ctx, newInstancetype(), metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	pref := newPreference()
	pref.Spec.CPU.PreferredCPUTopology = pointer.P(instancetypev1beta1.Cores)
	pref.Spec.Devices = &instancetypev1beta1.DevicePreferences{PreferredDiskBus: virtv1.DiskBusVirtio}
	pref, err = f.VirtClient().VirtualMachinePreference(namespace).Create(ctx, pref, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	vmi := libvmi.New(libvmi.WithEmptyDisk("empty", virtv1.DiskBusSATA, resource.MustParse("1Mi")))
	vmi.Spec.Domain.Devices.Disks = nil
	vm := libvmi.NewVirtualMachine(vmi, libvmi.WithInstancetype(it.Name), libvmi.WithPreference(pref.Name),
		libvmi.WithRunStrategy(virtv1.RunStrategyAlways))
	vm, err = f.VirtClient().VirtualMachine(namespace).Create(ctx, vm, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
	return vm
}
