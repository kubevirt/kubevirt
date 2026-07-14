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

package vmliveupdate_test

import (
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/vmliveupdate"
)

var _ = Describe("IsRestartRequired", func() {
	const (
		defaultNetworkName = "default"

		secondaryNetName1 = "foo"
		secondaryNADName1 = "foo-nad"

		secondaryNetName2 = "bar"
		secondaryNADName2 = "bar-nad"
	)

	DescribeTable("should not require restart when there is no change", func(vmi *v1.VirtualMachineInstance) {
		vm := libvmi.NewVirtualMachine(vmi).DeepCopy()

		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeFalse())
	},
		Entry("Without interfaces and networks",
			libvmi.New(libvmi.WithAutoAttachPodInterface(false))),
		Entry("With interfaces and networks",
			libvmi.New(
				libvmi.WithInterface(libvmi.NewInterface(defaultNetworkName, libvmi.WithMasqueradeBinding())),
				libvmi.WithInterface(libvmi.NewInterface(secondaryNetName1, libvmi.WithBridgeBinding())),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName1, secondaryNADName1)),
			),
		),
	)

	It("should not require restart when networks are added", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(defaultNetworkName, libvmi.WithMasqueradeBinding())),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vm := libvmi.NewVirtualMachine(vmi).DeepCopy()

		vm.Spec.Template.Spec.Domain.Devices.Interfaces = append(
			vm.Spec.Template.Spec.Domain.Devices.Interfaces,
			libvmi.NewInterface(secondaryNetName1, libvmi.WithBridgeBinding()),
		)

		vm.Spec.Template.Spec.Networks = append(
			vm.Spec.Template.Spec.Networks,
			*libvmi.MultusNetwork(secondaryNetName1, secondaryNADName1),
		)
		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeFalse())
	})

	DescribeTable("should not require restart when interface state changes", func(current, desired v1.InterfaceState) {
		iface := libvmi.NewInterface(secondaryNetName1, libvmi.WithBridgeBinding())
		iface.State = current

		vmi := libvmi.New(
			libvmi.WithInterface(iface),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName1, secondaryNADName1)),
		)

		vm := libvmi.NewVirtualMachine(vmi).DeepCopy()
		vm.Spec.Template.Spec.Domain.Devices.Interfaces[0].State = desired

		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeFalse())
	},
		Entry("From empty to empty", v1.InterfaceState(""), v1.InterfaceState("")),
		Entry("From empty to absent", v1.InterfaceState(""), v1.InterfaceStateAbsent),
		Entry("From empty to up", v1.InterfaceState(""), v1.InterfaceStateLinkUp),
		Entry("From empty to down", v1.InterfaceState(""), v1.InterfaceStateLinkDown),
		Entry("From up to empty", v1.InterfaceStateLinkUp, v1.InterfaceState("")),
		Entry("From up to absent", v1.InterfaceStateLinkUp, v1.InterfaceStateAbsent),
		Entry("From up to up", v1.InterfaceStateLinkUp, v1.InterfaceStateLinkUp),
		Entry("From up to down", v1.InterfaceStateLinkUp, v1.InterfaceStateLinkDown),
		Entry("From down to empty", v1.InterfaceStateLinkDown, v1.InterfaceState("")),
		Entry("From down to absent", v1.InterfaceStateLinkDown, v1.InterfaceStateAbsent),
		Entry("From down to up", v1.InterfaceStateLinkDown, v1.InterfaceStateLinkUp),
		Entry("From down to down", v1.InterfaceStateLinkDown, v1.InterfaceStateLinkDown),
	)

	It("should not require restart when secondary NICs are hotplugged", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(defaultNetworkName, libvmi.WithMasqueradeBinding())),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		vm := libvmi.NewVirtualMachine(vmi).DeepCopy()

		ifacesToHotplug := []v1.Interface{
			libvmi.NewInterface(secondaryNetName1, libvmi.WithBridgeBinding()),
			libvmi.NewInterface(secondaryNetName2, libvmi.WithSRIOVBinding()),
		}

		netsToHotplug := []v1.Network{
			*libvmi.MultusNetwork(secondaryNetName1, secondaryNADName1),
			*libvmi.MultusNetwork(secondaryNetName2, secondaryNADName2),
		}

		vm.Spec.Template.Spec.Domain.Devices.Interfaces = append(
			vm.Spec.Template.Spec.Domain.Devices.Interfaces,
			ifacesToHotplug...,
		)

		vm.Spec.Template.Spec.Networks = append(vm.Spec.Template.Spec.Networks, netsToHotplug...)

		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeFalse())
	})

	It("should not require restart when interfaces or networks order is changed", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(defaultNetworkName, libvmi.WithMasqueradeBinding())),
			libvmi.WithInterface(libvmi.NewInterface(secondaryNetName1, libvmi.WithBridgeBinding())),
			libvmi.WithInterface(libvmi.NewInterface(secondaryNetName2, libvmi.WithBridgeBinding())),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName1, secondaryNADName1)),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName2, secondaryNADName2)),
		)

		vm := libvmi.NewVirtualMachine(vmi).DeepCopy()

		slices.Reverse(vm.Spec.Template.Spec.Domain.Devices.Interfaces)
		slices.Reverse(vm.Spec.Template.Spec.Networks)

		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeFalse())
	})

	It("should require restart when interface binding changes", func() {
		iface := libvmi.NewInterface(secondaryNetName1, libvmi.WithBridgeBinding())

		vmi := libvmi.New(
			libvmi.WithInterface(iface),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName1, secondaryNADName1)),
		)

		vm := libvmi.NewVirtualMachine(vmi).DeepCopy()
		vm.Spec.Template.Spec.Domain.Devices.Interfaces[0] = libvmi.NewInterface(defaultNetworkName, libvmi.WithMasqueradeBinding())

		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeTrue())
	})

	DescribeTable("should require restart when network source changes", func(
		current,
		desired v1.Network,
	) {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(defaultNetworkName, libvmi.WithMasqueradeBinding())),
			libvmi.WithNetwork(&current),
		)

		vm := libvmi.NewVirtualMachine(vmi).DeepCopy()
		vm.Spec.Template.Spec.Networks[0] = desired

		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeTrue())
	},
		Entry("From Pod to Multus", *v1.DefaultPodNetwork(),
			*libvmi.MultusNetwork("default", secondaryNADName1)),
		Entry("From Multus to Pod", *libvmi.MultusNetwork("default", secondaryNADName1),
			*v1.DefaultPodNetwork()),
	)

	DescribeTable("should set restart requirement for NAD name changes based on migratability",
		func(conditions []v1.VirtualMachineInstanceCondition, expectedRestart bool) {
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.NewInterface(secondaryNetName1, libvmi.WithBridgeBinding())),
				libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName1, secondaryNADName1)),
			)
			vmi.Status.Conditions = conditions

			vm := libvmi.NewVirtualMachine(vmi).DeepCopy()
			vm.Spec.Template.Spec.Networks[0] = *libvmi.MultusNetwork(secondaryNetName1, secondaryNADName2)

			Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(Equal(expectedRestart))
		},
		Entry("when VM is migratable", []v1.VirtualMachineInstanceCondition{
			{Type: v1.VirtualMachineInstanceIsMigratable, Status: k8sv1.ConditionTrue},
		}, false),
		Entry("when VM is non-migratable", []v1.VirtualMachineInstanceCondition{
			{Type: v1.VirtualMachineInstanceIsMigratable, Status: k8sv1.ConditionFalse},
		}, true),
		Entry("when IsMigratable condition is absent", nil, true),
	)

	It("Should require restart when interfaces and networks are removed", func() {
		vmi := libvmi.New(
			libvmi.WithInterface(libvmi.NewInterface(defaultNetworkName, libvmi.WithMasqueradeBinding())),
			libvmi.WithInterface(libvmi.NewInterface(secondaryNetName1, libvmi.WithBridgeBinding())),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithNetwork(libvmi.MultusNetwork(secondaryNetName1, secondaryNADName1)),
		)

		vm := libvmi.NewVirtualMachine(vmi).DeepCopy()
		vm.Spec.Template.Spec.Domain.Devices.Interfaces = vm.Spec.Template.Spec.Domain.Devices.Interfaces[:1]
		vm.Spec.Template.Spec.Networks = vm.Spec.Template.Spec.Networks[:1]

		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeTrue())
	})

	It("should require restart when pod network is added to networkless VM", func() {
		vmi := libvmi.New()

		vm := libvmi.NewVirtualMachine(
			libvmi.New(
				libvmi.WithInterface(libvmi.NewInterface(defaultNetworkName, libvmi.WithMasqueradeBinding())),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			),
		)

		Expect(vmliveupdate.IsRestartRequired(vm, vmi)).To(BeTrue())
	})
})
