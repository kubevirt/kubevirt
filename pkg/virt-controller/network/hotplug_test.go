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
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-controller/network"
)

var _ = Describe("Network interface hot{un}plug", func() {
	const (
		testNetworkName1 = "testnet1"
		testNetworkName2 = "testnet2"

		ordinal = true
	)

	DescribeTable("apply dynamic interface request on VMI",
		func(vmiForVM, currentVMI, expectedVMI *v1.VirtualMachineInstance, hasOrdinalIfaces bool) {
			vm := virtualMachineFromVMI(currentVMI.Name, vmiForVM)
			updatedVMI := network.ApplyDynamicIfaceRequestOnVMI(vm, currentVMI, hasOrdinalIfaces)
			Expect(updatedVMI.Networks).To(Equal(expectedVMI.Spec.Networks))
			Expect(updatedVMI.Domain.Devices.Interfaces).To(Equal(expectedVMI.Spec.Domain.Devices.Interfaces))
		},
		Entry("when the are no interfaces to hotplug/unplug",
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			false),
		Entry("when a bridge binding interface has to be hotplugged",
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(),
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			!ordinal),
		Entry("when an SRIOV  binding interface has to be hotplugged",
			libvmi.New(
				libvmi.WithInterface(sriovInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(),
			libvmi.New(
				libvmi.WithInterface(sriovInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			!ordinal),
		Entry("when an interface has to be hotplugged but it has no SRIOV or bridge binding",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: testNetworkName1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(),
			libvmi.New(),
			!ordinal),
		Entry("when an interface has to be hotplugged but it is absent",
			libvmi.New(
				libvmi.WithInterface(bridgeAbsentInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(),
			libvmi.New(),
			!ordinal),
		Entry("when an interface has to be hotunplugged",
			libvmi.New(
				libvmi.WithInterface(bridgeAbsentInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(
				libvmi.WithInterface(bridgeAbsentInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			!ordinal),
		Entry("when an interface has to be hotunplugged but it has ordinal name",
			libvmi.New(
				libvmi.WithInterface(bridgeAbsentInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			ordinal),
		Entry("when one interface has to be plugged and other hotunplugged",
			libvmi.New(
				libvmi.WithInterface(bridgeAbsentInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				libvmi.WithInterface(bridgeInterface(testNetworkName2)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
			),
			libvmi.New(
				libvmi.WithInterface(bridgeInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
			),
			libvmi.New(
				libvmi.WithInterface(bridgeAbsentInterface(testNetworkName1)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				libvmi.WithInterface(bridgeInterface(testNetworkName2)),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
			),
			!ordinal),
	)

	DescribeTable("spec interfaces",
		func(specIfaces []v1.Interface, statusIfaces []v1.VirtualMachineInstanceNetworkInterface,
			expectedInterfaces []v1.Interface, expectedNetworks []v1.Network) {
			var testNetworks []v1.Network
			for _, iface := range specIfaces {
				testNetworks = append(testNetworks, v1.Network{Name: iface.Name})
			}
			testStatusIfaces := vmispec.IndexInterfaceStatusByName(statusIfaces,
				func(i v1.VirtualMachineInstanceNetworkInterface) bool { return true })

			ifaces, networks := network.ClearDetachedInterfaces(specIfaces, testNetworks, testStatusIfaces)

			Expect(ifaces).To(Equal(expectedInterfaces))
			Expect(networks).To(Equal(expectedNetworks))
		},
		Entry("should remain, given non-absent interfaces, and no associated status ifaces (i.e.: plug pending)",
			[]v1.Interface{{Name: "blue"}, {Name: "red"}},
			[]v1.VirtualMachineInstanceNetworkInterface{},
			[]v1.Interface{{Name: "blue"}, {Name: "red"}},
			[]v1.Network{{Name: "blue"}, {Name: "red"}},
		),
		Entry("should remain, given non-absent interfaces, and associated status ifaces (i.e.: plugged iface)",
			[]v1.Interface{{Name: "blue"}, {Name: "red"}},
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: "blue"}, {Name: "red"}},
			[]v1.Interface{{Name: "blue"}, {Name: "red"}},
			[]v1.Network{{Name: "blue"}, {Name: "red"}},
		),
		Entry("should remain, given absent iface and associated status ifaces (i.e.: unplug pending)",
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}, {Name: "red"}},
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: "blue"}, {Name: "red"}},
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}, {Name: "red"}},
			[]v1.Network{{Name: "blue"}, {Name: "red"}},
		),
		Entry("should be cleared, given absent iface and no associated status iface (i.e.: unplugged iface)",
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}, {Name: "red"}},
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: "red"}},
			[]v1.Interface{{Name: "red"}},
			[]v1.Network{{Name: "red"}},
		),
		Entry("should remain, given status iface and no associated iface in spec",
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInstanceNetworkInterface{{Name: "RED"}},
			[]v1.Interface{{Name: "blue"}},
			[]v1.Network{{Name: "blue"}},
		),
	)
})

func bridgeInterface(name string) v1.Interface {
	return v1.Interface{Name: name, InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}
}

func sriovInterface(name string) v1.Interface {
	return v1.Interface{Name: name, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}
}

func bridgeAbsentInterface(name string) v1.Interface {
	iface := bridgeInterface(name)
	iface.State = v1.InterfaceStateAbsent
	return iface
}

func virtualMachineFromVMI(name string, vmi *v1.VirtualMachineInstance) *v1.VirtualMachine {
	started := true
	vm := &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1", UID: "vm-uid"},
		Spec: v1.VirtualMachineSpec{
			Running: &started,
			Template: &v1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return vm
}
