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
 * Copyright The KubeVirt Authors.
 *
 */
package vm_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"go.uber.org/mock/gomock"
	k8sv1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/kubevirt/fake"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	webhook "kubevirt.io/kubevirt/pkg/instancetype/webhooks/vm"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
)

type admitHandler interface {
	ApplyToVM(*virtv1.VirtualMachine) (
		*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		[]metav1.StatusCause,
	)
	Check(*v1beta1.VirtualMachineInstancetypeSpec,
		*v1beta1.VirtualMachinePreferenceSpec,
		*virtv1.VirtualMachineInstanceSpec,
	) (conflict.Conflicts, error)
}

var _ = Describe("Instance type and Preference VirtualMachine Admitter", func() {
	Context("Given a valid VirtualMachine", func() {
		var (
			vm         *virtv1.VirtualMachine
			virtClient *kubecli.MockKubevirtClient
			admitter   admitHandler
		)

		const (
			instancetypeName = "instancetypeName"
			preferenceName   = "preferenceName"
			unknownResource  = "unknown"
		)

		BeforeEach(func() {
			ctrl := gomock.NewController(GinkgoT())
			virtClient = kubecli.NewMockKubevirtClient(ctrl)

			fakeKubevirtV1 := fake.NewSimpleClientset().KubevirtV1()
			fakeInstancetypeV1beta1 := fake.NewSimpleClientset().InstancetypeV1beta1()

			virtClient.EXPECT().VirtualMachine(metav1.NamespaceDefault).Return(
				fakeKubevirtV1.VirtualMachines(metav1.NamespaceDefault)).AnyTimes()

			virtClient.EXPECT().VirtualMachineInstance(metav1.NamespaceDefault).Return(
				fakeKubevirtV1.VirtualMachineInstances(metav1.NamespaceDefault)).AnyTimes()

			virtClient.EXPECT().VirtualMachineInstancetype(k8sv1.NamespaceDefault).Return(
				fakeInstancetypeV1beta1.VirtualMachineInstancetypes(metav1.NamespaceDefault)).AnyTimes()

			virtClient.EXPECT().VirtualMachineClusterInstancetype().Return(
				fakeInstancetypeV1beta1.VirtualMachineClusterInstancetypes()).AnyTimes()

			virtClient.EXPECT().VirtualMachinePreference(k8sv1.NamespaceDefault).Return(
				fakeInstancetypeV1beta1.VirtualMachinePreferences(metav1.NamespaceDefault)).AnyTimes()

			virtClient.EXPECT().VirtualMachineClusterPreference().Return(
				fakeInstancetypeV1beta1.VirtualMachineClusterPreferences()).AnyTimes()

			// TODO(lyarwood): Add WithNamespace to libinstancetype.builder and use here
			testInstancetype := &v1beta1.VirtualMachineInstancetype{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instancetypeName,
					Namespace: metav1.NamespaceDefault,
				},
				Spec: v1beta1.VirtualMachineInstancetypeSpec{
					CPU: v1beta1.CPUInstancetype{
						Guest: uint32(2),
					},
					Memory: v1beta1.MemoryInstancetype{
						Guest: resource.MustParse("128Mi"),
					},
				},
			}
			_, err := virtClient.VirtualMachineInstancetype(
				metav1.NamespaceDefault).Create(context.Background(), testInstancetype, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			// TODO(lyarwood): Add WithNamespace to libinstancetype.builder and use here
			testPreference := &v1beta1.VirtualMachinePreference{
				ObjectMeta: metav1.ObjectMeta{
					Name:      preferenceName,
					Namespace: metav1.NamespaceDefault,
				},
				Spec: v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Cores),
					},
				},
			}

			_, err = virtClient.VirtualMachinePreference(
				metav1.NamespaceDefault).Create(context.Background(), testPreference, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vm = libvmi.NewVirtualMachine(
				libvmi.New(libvmi.WithNamespace(metav1.NamespaceDefault)),
				libvmi.WithInstancetype(instancetypeName),
				libvmi.WithPreference(preferenceName),
			)

			admitter = webhook.NewAdmitter(virtClient)
		})

		It("should reject if instancetype fails to apply to VMI", func() {
			vm.Spec.Template.Spec.Domain = virtv1.DomainSpec{
				CPU: &virtv1.CPU{
					Sockets: 1,
				},
				Memory: &virtv1.Memory{
					Guest: pointer.P(resource.MustParse("1Gi")),
				},
			}

			instancetypeSpec, preferenceSpec, causes := admitter.ApplyToVM(vm)
			Expect(instancetypeSpec).To(BeNil())
			Expect(preferenceSpec).To(BeNil())
			Expect(causes).To(ContainElements(
				[]metav1.StatusCause{
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "VM field(s) spec.template.spec.domain.cpu.sockets conflicts with selected instance type",
						Field:   "spec.template.spec.domain.cpu.sockets",
					},
					{
						Type:    metav1.CauseTypeFieldValueInvalid,
						Message: "VM field(s) spec.template.spec.domain.memory.guest conflicts with selected instance type",
						Field:   "spec.template.spec.domain.memory.guest",
					},
				},
			))
		})

		It("should reject if preference requirements are not met", func() {
			testPreference, err := virtClient.VirtualMachinePreference(
				metav1.NamespaceDefault).Get(context.Background(), preferenceName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			testPreference.Spec.Requirements = &v1beta1.PreferenceRequirements{
				CPU: &v1beta1.CPUPreferenceRequirement{
					Guest: 10,
				},
			}

			conflicts, err := admitter.Check(nil, &testPreference.Spec, &vm.Spec.Template.Spec)
			Expect(err).To(MatchError("no CPU resources provided by VirtualMachine, preference requires 10 vCPU"))
			Expect(conflicts).To(ContainElements(conflict.New("spec", "template", "spec", "domain", "cpu")))
		})

		const (
			instancetypeCPUGuestPath       = "instancetype.spec.cpu.guest"
			spreadAcrossSocketsCoresErrFmt = "%d vCPUs provided by the instance type are not divisible by the " +
				"Spec.PreferSpreadSocketToCoreRatio or Spec.CPU.PreferSpreadOptions.Ratio of %d provided by the preference"
			spreadAcrossCoresThreadsErrFmt        = "%d vCPUs provided by the instance type are not divisible by the number of threads per core %d"
			spreadAcrossSocketsCoresThreadsErrFmt = "%d vCPUs provided by the instance type are not divisible by the number of threads per core " +
				"%d and Spec.PreferSpreadSocketToCoreRatio or Spec.CPU.PreferSpreadOptions.Ratio of %d"
		)

		DescribeTable("should reject if PreferSpread requested with",
			func(vCPUs uint32, expectedPreferenceSpec v1beta1.VirtualMachinePreferenceSpec, expectedMessage string) {
				testPreference, err := virtClient.VirtualMachinePreference(
					metav1.NamespaceDefault).Get(context.Background(), preferenceName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				testPreference.Spec = expectedPreferenceSpec

				_, err = virtClient.VirtualMachinePreference(vm.Namespace).Update(context.Background(), testPreference, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())

				testInstancetype, err := virtClient.VirtualMachineInstancetype(
					metav1.NamespaceDefault).Get(context.Background(), instancetypeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				testInstancetype.Spec.CPU.Guest = vCPUs

				_, err = virtClient.VirtualMachineInstancetype(vm.Namespace).Update(context.Background(), testInstancetype, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())

				instancetypeSpec, preferenceSpec, causes := admitter.ApplyToVM(vm)
				Expect(instancetypeSpec).To(BeNil())
				Expect(preferenceSpec).To(BeNil())
				Expect(causes).To(ContainElement(metav1.StatusCause{
					Type:    metav1.CauseTypeFieldValueInvalid,
					Message: expectedMessage,
					Field:   "instancetype.spec.cpu.guest",
				}))
			},
			Entry("3 vCPUs, default of SpreadAcrossSocketsCores and default SocketCoreRatio of 2 with spread",
				uint32(3),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 3, 2),
			),
			Entry("3 vCPUs, default of SpreadAcrossSocketsCores and default SocketCoreRatio of 2 with preferSpread",
				uint32(3),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 3, 2),
			),
			Entry("2 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 3 with spread",
				uint32(2),
				v1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 2, 3),
			),
			Entry("2 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 3 with preferSpread",
				uint32(2),
				v1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 2, 3),
			),
			Entry("2 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via SpreadOptions of 3 with spread",
				uint32(2),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 2, 3),
			),
			Entry("2 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via SpreadOptions of 3 with preferSpread",
				uint32(2),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 2, 3),
			),
			Entry("4 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 3 with spread",
				uint32(4),
				v1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 4, 3),
			),
			Entry("4 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 3 with preferSpread",
				uint32(4),
				v1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(3),
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 4, 3),
			),
			Entry("4 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via SpreadOptions of 3 with spread",
				uint32(4),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 4, 3),
			),
			Entry("4 vCPUs, default of SpreadAcrossSocketsCores and SocketCoreRatio via SpreadOptions of 3 with preferSpread",
				uint32(4),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Ratio: pointer.P(uint32(3)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresErrFmt, 4, 3),
			),
			Entry("3 vCPUs and SpreadAcrossCoresThreads with spread",
				uint32(3),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, 3, 2),
			),
			Entry("3 vCPUs and SpreadAcrossCoresThreads with preferSpread",
				uint32(3),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, 3, 2),
			),
			Entry("5 vCPUs and SpreadAcrossCoresThreads with spread",
				uint32(5),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, 5, 2),
			),
			Entry("5 vCPUs and SpreadAcrossCoresThreads with preferSpread",
				uint32(5),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossCoresThreadsErrFmt, 5, 2),
			),
			Entry("5 vCPUs, SpreadAcrossSocketsCoresThreads and default SocketCoreRatio of 2 with spread",
				uint32(5),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 5, 2, 2),
			),
			Entry("5 vCPUs, SpreadAcrossSocketsCoresThreads and default SocketCoreRatio of 2 with preferSpread",
				uint32(5),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 5, 2, 2),
			),
			Entry("6 vCPUs, SpreadAcrossSocketsCoresThreads and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 4 with spread",
				uint32(6),
				v1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(4),
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 6, 2, 4),
			),
			Entry("6 vCPUs, SpreadAcrossSocketsCoresThreads and SocketCoreRatio via PreferSpreadSocketToCoreRatio of 4 with preferSpread",
				uint32(6),
				v1beta1.VirtualMachinePreferenceSpec{
					PreferSpreadSocketToCoreRatio: uint32(4),
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 6, 2, 4),
			),
			Entry("6 vCPUs, SpreadAcrossSocketsCoresThreads and SocketCoreRatio via SpreadOptions of 4 with spread",
				uint32(6),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.Spread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(4)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 6, 2, 4),
			),
			Entry("6 vCPUs, SpreadAcrossSocketsCoresThreads and SocketCoreRatio via SpreadOptions of 4 with preferSpread",
				uint32(6),
				v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: pointer.P(v1beta1.DeprecatedPreferSpread),
						SpreadOptions: &v1beta1.SpreadOptions{
							Across: pointer.P(v1beta1.SpreadAcrossSocketsCoresThreads),
							Ratio:  pointer.P(uint32(4)),
						},
					},
				},
				fmt.Sprintf(spreadAcrossSocketsCoresThreadsErrFmt, 6, 2, 4),
			),
		)

		DescribeTable("should admit VM with preference using preferSpread and without instancetype",
			func(preferredCPUTopology v1beta1.PreferredCPUTopology) {
				vm.Spec.Instancetype = nil

				testPreference, err := virtClient.VirtualMachinePreference(
					metav1.NamespaceDefault).Get(context.Background(), preferenceName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				testPreference.Spec.CPU.PreferredCPUTopology = pointer.P(preferredCPUTopology)

				_, err = virtClient.VirtualMachinePreference(vm.Namespace).Update(context.Background(), testPreference, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())

				instancetypeSpec, preferenceSpec, causes := admitter.ApplyToVM(vm)
				Expect(instancetypeSpec).To(BeNil())
				Expect(preferenceSpec).ToNot(BeNil())
				Expect(causes).To(BeNil())
			},
			Entry("with spread", v1beta1.Spread),
			Entry("with preferSpread", v1beta1.DeprecatedPreferSpread),
		)

		DescribeTable("should admit when", func(vm *virtv1.VirtualMachine) {
			instancetypeSpec, preferenceSpec, causes := admitter.ApplyToVM(vm)
			Expect(instancetypeSpec).To(BeNil())
			Expect(preferenceSpec).To(BeNil())
			Expect(causes).To(BeNil())
		},
			Entry("VirtualMachineInstancetype is referenced but not found",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(k8sv1.NamespaceDefault),
					),
					libvmi.WithInstancetype("unknown"),
				),
			),
			Entry("VirtualMachineClusterInstancetype is referenced but not found",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(k8sv1.NamespaceDefault),
					),
					libvmi.WithClusterInstancetype("unknown"),
				),
			),
			Entry("VirtualMachinePreference is referenced but not found",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(k8sv1.NamespaceDefault),
					),
					libvmi.WithPreference("unknown"),
				),
			),
			Entry("VirtualMachineClusterPreference is referenced but not found",
				libvmi.NewVirtualMachine(
					libvmi.New(
						libvmi.WithNamespace(k8sv1.NamespaceDefault),
					),
					libvmi.WithClusterPreference("unknown"),
				),
			),
		)
	})
})
