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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/framework/matcher"
)

//nolint:funlen // Register the related Ginkgo scenarios in the shared container.
func (s *instancetypeSuite) describeReferences() {
	Context("Bug #16071", func() {
		var vm *virtv1.VirtualMachine

		BeforeEach(func(ctx SpecContext) {
			By("creating a VirtualMachinePreference")
			pref := &instancetypev1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-preference",
					Namespace: s.namespace,
				},
				Spec: instancetypev1beta1.VirtualMachinePreferenceSpec{},
			}
			_, err := s.f.VirtClient().VirtualMachinePreference(s.namespace).Create(ctx, pref, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("creating a halted VM with spec.preference referencing the preference")
			vm = libvmi.NewVirtualMachine(
				libvmi.New(
					libvmi.WithMemoryRequest("128Mi"),
				),
				libvmi.WithPreference("test-preference"),
			)
			vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Create(ctx, vm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("waiting for status.preferenceRef to be populated by the instancetype controller")
			Eventually(ctx, matcher.ThisVM(vm), 30*time.Second, 100*time.Millisecond).Should(
				matcher.HavePreferenceControllerRevisionRef(),
			)
		})

		It("should clear status.preferenceRef when spec.preference is removed", func(ctx SpecContext) {
			By("removing spec.preference from the VM")
			payload, err := patch.New(patch.WithRemove("/spec/preference")).GeneratePayload()
			Expect(err).NotTo(HaveOccurred())
			_, err = s.f.VirtClient().VirtualMachine(s.namespace).Patch(ctx, vm.Name, types.JSONPatchType, payload, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("verifying status.preferenceRef is cleared by the instancetype controller")
			Eventually(ctx, matcher.ThisVM(vm), 10*time.Second, 100*time.Millisecond).Should(
				HaveField("Status.PreferenceRef", BeNil()),
			)
		})
	})

	DescribeTableSubtree(
		"Missing references", func(cluster, preference bool, reason string) {
			var vm *virtv1.VirtualMachine
			var resourceName string

			BeforeEach(func(ctx SpecContext) {
				resourceName = s.namespace + "-missing"
				option := libvmi.WithInstancetype(resourceName)
				switch {
				case cluster && preference:
					option = libvmi.WithClusterPreference(resourceName)
				case cluster:
					option = libvmi.WithClusterInstancetype(resourceName)
				case preference:
					option = libvmi.WithPreference(resourceName)
				}
				vm = libvmi.NewVirtualMachine(libvmi.New(libvmi.WithMemoryRequest("128Mi")), option,
					libvmi.WithRunStrategy(virtv1.RunStrategyAlways))
				var err error
				vm, err = s.f.VirtClient().VirtualMachine(s.namespace).Create(ctx, vm, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("should recover when the missing resource is created", func(ctx SpecContext) {
				Eventually(ctx, matcher.ThisVM(vm), 30*time.Second, 100*time.Millisecond).Should(
					matcher.HaveConditionTrueWithReason(virtv1.VirtualMachineFailure, reason),
				)
				Eventually(ctx, matcher.ThisVM(vm), 30*time.Second, 100*time.Millisecond).Should(
					matcher.HavePrintableStatus(virtv1.VirtualMachineStatusStopped),
				)
				Expect(matcher.ThisVMIWith(s.namespace, vm.Name)()).To(matcher.BeGone())

				By("creating the missing resource")
				switch {
				case cluster && preference:
					createTestClusterPreference(ctx, s.f, &instancetypev1beta1.VirtualMachineClusterPreference{
						ObjectMeta: metav1.ObjectMeta{Name: resourceName}, Spec: newPreference().Spec,
					})
				case cluster:
					createTestClusterInstancetype(ctx, s.f, &instancetypev1beta1.VirtualMachineClusterInstancetype{
						ObjectMeta: metav1.ObjectMeta{Name: resourceName}, Spec: newInstancetype().Spec,
					})
				case preference:
					pref := newPreference()
					pref.Name, pref.GenerateName = resourceName, ""
					_, err := s.f.VirtClient().VirtualMachinePreference(s.namespace).Create(ctx, pref, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
				default:
					it := newInstancetype()
					it.Name, it.GenerateName = resourceName, ""
					_, err := s.f.VirtClient().VirtualMachineInstancetype(s.namespace).Create(ctx, it, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
				}
				Eventually(ctx, matcher.ThisVM(vm), 30*time.Second, 100*time.Millisecond).ShouldNot(
					matcher.HaveConditionTrueWithReason(virtv1.VirtualMachineFailure, reason),
				)
				waitForScheduledVMI(ctx, s.f, vm)
			})
		},
		Entry("cluster instancetype [test_id:CNV-9086]", true, false, "FailedFindInstancetype"),
		Entry("namespaced instancetype [test_id:CNV-9089]", false, false, "FailedFindInstancetype"),
		Entry("cluster preference [test_id:CNV-9091]", true, true, "FailedFindPreference"),
		Entry("namespaced preference [test_id:CNV-9090]", false, true, "FailedFindPreference"),
	)
}
