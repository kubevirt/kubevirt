/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors
 *
 */

package subresources

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/compute"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	instancetypebuilder "kubevirt.io/kubevirt/tests/libinstancetype/builder"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(compute.SIG("ExpandSpec subresource", func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("instancetype", func() {
		var (
			instancetype               *instancetypev1beta1.VirtualMachineInstancetype
			clusterInstancetype        *instancetypev1beta1.VirtualMachineClusterInstancetype
			instancetypeMatcher        *v1.InstancetypeMatcher
			clusterInstancetypeMatcher *v1.InstancetypeMatcher
			expectedCpu                *v1.CPU

			instancetypeMatcherFn = func() *v1.InstancetypeMatcher {
				return instancetypeMatcher
			}
			clusterInstancetypeMatcherFn = func() *v1.InstancetypeMatcher {
				return clusterInstancetypeMatcher
			}
		)

		BeforeEach(func() {
			var err error
			instancetype = instancetypebuilder.NewInstancetype(
				instancetypebuilder.WithCPUs(2),
			)
			instancetype, err = virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Create(context.Background(), instancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			instancetypeMatcher = &v1.InstancetypeMatcher{
				Name: instancetype.Name,
				Kind: instancetypeapi.SingularResourceName,
			}

			clusterInstancetype = instancetypebuilder.NewClusterInstancetype(
				instancetypebuilder.WithCPUs(2),
			)
			clusterInstancetype, err = virtClient.VirtualMachineClusterInstancetype().
				Create(context.Background(), clusterInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			clusterInstancetypeMatcher = &v1.InstancetypeMatcher{
				Name: clusterInstancetype.Name,
				Kind: instancetypeapi.ClusterSingularResourceName,
			}

			expectedCpu = &v1.CPU{
				Sockets: 2,
				Cores:   1,
				Threads: 1,
			}
		})

		AfterEach(func() {
			Expect(virtClient.VirtualMachineInstancetype(testsuite.GetTestNamespace(instancetype)).
				Delete(context.Background(), instancetype.Name, metav1.DeleteOptions{})).To(Succeed())
			Expect(virtClient.VirtualMachineClusterInstancetype().
				Delete(context.Background(), clusterInstancetype.Name, metav1.DeleteOptions{})).To(Succeed())
		})

		Context("with existing VM", func() {
			It("[test_id:TODO] should return unchanged VirtualMachine, if instancetype is not used", func() {
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expandedVm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).
					GetWithExpandedSpec(context.Background(), vm.GetName())
				Expect(err).ToNot(HaveOccurred())
				Expect(expandedVm.Spec).To(Equal(vm.Spec))
			})

			DescribeTable("[test_id:TODO] should return VirtualMachine with instancetype expanded", func(matcherFn func() *v1.InstancetypeMatcher) {
				vm := libvmi.NewVirtualMachine(libvmi.New())
				vm.Spec.Instancetype = matcherFn()

				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expandedVm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).
					GetWithExpandedSpec(context.Background(), vm.GetName())
				Expect(err).ToNot(HaveOccurred())
				Expect(expandedVm.Spec.Instancetype).To(BeNil(), "Expanded VM should not have InstancetypeMatcher")
				Expect(expandedVm.Spec.Template.Spec.Domain.CPU).To(Equal(expectedCpu), "VM should have instancetype expanded")
			},
				Entry("with VirtualMachineInstancetype", instancetypeMatcherFn),
				Entry("with VirtualMachineClusterInstancetype", clusterInstancetypeMatcherFn),
			)
		})

		Context("with passed VM in request", func() {
			It("[test_id:TODO] should return unchanged VirtualMachine, if instancetype is not used", func() {
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())

				expandedVm, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(expandedVm.Spec).To(Equal(vm.Spec))
			})

			DescribeTable("[test_id:TODO] should return VirtualMachine with instancetype expanded", func(matcherFn func() *v1.InstancetypeMatcher) {
				vm := libvmi.NewVirtualMachine(libvmi.New())
				vm.Spec.Instancetype = matcherFn()

				expandedVm, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(expandedVm.Spec.Instancetype).To(BeNil(), "Expanded VM should not have InstancetypeMatcher")
				Expect(expandedVm.Spec.Template.Spec.Domain.CPU).To(Equal(expectedCpu), "VM should have instancetype expanded")
			},
				Entry("with VirtualMachineInstancetype", instancetypeMatcherFn),
				Entry("with VirtualMachineClusterInstancetype", clusterInstancetypeMatcherFn),
			)

			DescribeTable("[test_id:TODO] should fail, if referenced instancetype does not exist", func(matcher *v1.InstancetypeMatcher) {
				vm := libvmi.NewVirtualMachine(libvmi.New())
				vm.Spec.Instancetype = matcher

				_, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(matcher.Kind + ".instancetype.kubevirt.io \"" + matcher.Name + "\" not found"))
			},
				Entry("with VirtualMachineInstancetype", &v1.InstancetypeMatcher{Name: "nonexisting-instancetype", Kind: instancetypeapi.PluralResourceName}),
				Entry("with VirtualMachineClusterInstancetype", &v1.InstancetypeMatcher{Name: "nonexisting-clusterinstancetype", Kind: instancetypeapi.ClusterPluralResourceName}),
			)

			DescribeTable("[test_id:TODO] should fail, if instancetype expansion hits a conflict", func(matcherFn func() *v1.InstancetypeMatcher) {
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
				vm.Spec.Instancetype = matcherFn()

				_, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(conflict.New("spec.template.spec.domain.resources.requests.memory").Error()))
			},
				Entry("with VirtualMachineInstancetype", instancetypeMatcherFn),
				Entry("with VirtualMachineClusterInstancetype", clusterInstancetypeMatcherFn),
			)

			DescribeTable("[test_id:TODO] should fail, if VM and endpoint namespace are different", func(matcherFn func() *v1.InstancetypeMatcher) {
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
				vm.Spec.Instancetype = matcherFn()
				vm.Namespace = "madethisup"

				_, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(nil)).ForVirtualMachine(vm)
				Expect(err).To(HaveOccurred())
				errMsg := fmt.Sprintf("VM namespace must be empty or %s", testsuite.GetTestNamespace(nil))
				Expect(err).To(MatchError(errMsg))
			},
				Entry("with VirtualMachineInstancetype", instancetypeMatcherFn),
				Entry("with VirtualMachineClusterInstancetype", clusterInstancetypeMatcherFn),
			)
		})
	})

	Context("preference", func() {
		var (
			preference        *instancetypev1beta1.VirtualMachinePreference
			clusterPreference *instancetypev1beta1.VirtualMachineClusterPreference

			preferenceMatcher        *v1.PreferenceMatcher
			clusterPreferenceMatcher *v1.PreferenceMatcher

			preferenceMatcherFn = func() *v1.PreferenceMatcher {
				return preferenceMatcher
			}
			clusterPreferenceMatcherFn = func() *v1.PreferenceMatcher {
				return clusterPreferenceMatcher
			}
		)

		BeforeEach(func() {
			var err error
			preference = instancetypebuilder.NewPreference()
			preference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
				PreferredAutoattachSerialConsole: pointer.P(true),
			}
			preference, err = virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
				Create(context.Background(), preference, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			preferenceMatcher = &v1.PreferenceMatcher{
				Name: preference.Name,
				Kind: instancetypeapi.SingularPreferenceResourceName,
			}

			clusterPreference = instancetypebuilder.NewClusterPreference()
			clusterPreference.Spec.Devices = &instancetypev1beta1.DevicePreferences{
				PreferredAutoattachSerialConsole: pointer.P(true),
			}
			clusterPreference, err = virtClient.VirtualMachineClusterPreference().
				Create(context.Background(), clusterPreference, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			clusterPreferenceMatcher = &v1.PreferenceMatcher{
				Name: clusterPreference.Name,
				Kind: instancetypeapi.ClusterSingularPreferenceResourceName,
			}
		})

		AfterEach(func() {
			Expect(virtClient.VirtualMachinePreference(testsuite.GetTestNamespace(preference)).
				Delete(context.Background(), preference.Name, metav1.DeleteOptions{})).To(Succeed())
			Expect(virtClient.VirtualMachineClusterPreference().
				Delete(context.Background(), clusterPreference.Name, metav1.DeleteOptions{})).To(Succeed())
		})

		Context("with existing VM", func() {
			It("[test_id:TODO] should return unchanged VirtualMachine, if preference is not used", func() {
				// Using NewCirros() here to have some data in spec.
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())

				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expandedVm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).
					GetWithExpandedSpec(context.Background(), vm.GetName())
				Expect(err).ToNot(HaveOccurred())
				Expect(expandedVm.Spec).To(Equal(vm.Spec))
			})

			DescribeTable("[test_id:TODO] should return VirtualMachine with preference expanded", func(matcherFn func() *v1.PreferenceMatcher) {
				// Using NewCirros() here to have some data in spec.
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
				vm.Spec.Preference = matcherFn()

				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				expandedVm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).
					GetWithExpandedSpec(context.Background(), vm.GetName())
				Expect(err).ToNot(HaveOccurred())
				Expect(expandedVm.Spec.Preference).To(BeNil(), "Expanded VM should not have InstancetypeMatcher")
				Expect(*expandedVm.Spec.Template.Spec.Domain.Devices.AutoattachSerialConsole).To(BeTrue(), "VM should have preference expanded")
			},
				Entry("with VirtualMachinePreference", preferenceMatcherFn),
				Entry("with VirtualMachineClusterPreference", clusterPreferenceMatcherFn),
			)
		})

		Context("with passed VM in request", func() {
			It("[test_id:TODO] should return unchanged VirtualMachine, if preference is not used", func() {
				// Using NewCirros() here to have some data in spec.
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())

				expandedVm, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(expandedVm.Spec).To(Equal(vm.Spec))
			})

			DescribeTable("[test_id:TODO] should return VirtualMachine with preference expanded", func(matcherFn func() *v1.PreferenceMatcher) {
				// Using NewCirros() here to have some data in spec.
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
				vm.Spec.Preference = matcherFn()

				expandedVm, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(expandedVm.Spec.Preference).To(BeNil(), "Expanded VM should not have InstancetypeMatcher")
				Expect(*expandedVm.Spec.Template.Spec.Domain.Devices.AutoattachSerialConsole).To(BeTrue(), "VM should have preference expanded")
			},
				Entry("with VirtualMachinePreference", preferenceMatcherFn),
				Entry("with VirtualMachineClusterPreference", clusterPreferenceMatcherFn),
			)

			DescribeTable("[test_id:TODO] should fail, if referenced preference does not exist", func(matcher *v1.PreferenceMatcher) {
				// Using NewCirros() here to have some data in spec.
				vm := libvmi.NewVirtualMachine(libvmifact.NewCirros())
				vm.Spec.Preference = matcher

				_, err := virtClient.ExpandSpec(testsuite.GetTestNamespace(vm)).ForVirtualMachine(vm)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(matcher.Kind + ".instancetype.kubevirt.io \"" + matcher.Name + "\" not found"))
			},
				Entry("with VirtualMachinePreference", &v1.PreferenceMatcher{Name: "nonexisting-preference", Kind: instancetypeapi.PluralPreferenceResourceName}),
				Entry("with VirtualMachineClusterPreference", &v1.PreferenceMatcher{Name: "nonexisting-clusterpreference", Kind: instancetypeapi.ClusterPluralPreferenceResourceName}),
			)
		})
	})
}))
