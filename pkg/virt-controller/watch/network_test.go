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

package watch

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Network interface hot{un}plug", func() {
	const (
		expectNoChange = false
		expectToChange = !expectNoChange

		testNetworkName1 = "testnet1"
		testNetworkName2 = "testnet2"
		testNetworkName3 = "testnet3"
		testNetworkName4 = "testnet4"

		ordinal = true
	)
	DescribeTable("calculate if changes are required",

		func(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod, expIfaces []v1.Interface, expNets []v1.Network, expToChange bool) {
			ifaces, nets, exists := calculateInterfacesAndNetworksForMultusAnnotationUpdate(vmi)
			Expect(ifaces).To(Equal(expIfaces))
			Expect(nets).To(Equal(expNets))
			Expect(exists).To(Equal(expToChange))
		},
		Entry("when no interfaces exist, change is not required", libvmi.New(), nil, nil, nil, expectNoChange),
		Entry("when vmi interfaces match pod multus annotation and status, change is not required",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: testNetworkName1}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName1}),
			),
			&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					networkv1.NetworkStatusAnnot: `[
						{"interface":"net1", "name":"red-net", "namespace": "default"}
					]`,
				}},
			}, nil, nil, expectNoChange,
		),
		Entry("when vmi interfaces have an extra interface which requires hotplug",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: testNetworkName1, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}),
				libvmi.WithInterface(v1.Interface{Name: testNetworkName2}),
				libvmi.WithInterface(v1.Interface{Name: testNetworkName3}),
				libvmi.WithInterface(v1.Interface{Name: testNetworkName4, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName3}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName4}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName1}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName2}),
			),
			&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					networkv1.NetworkStatusAnnot: `[
						{"interface":"net1", "name":"red-net", "namespace": "default"},
						{"interface":"net2", "name":"blue-net", "namespace": "default"}
					]`,
				}},
			},
			[]v1.Interface{{Name: testNetworkName1, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}, {Name: testNetworkName2}, {Name: testNetworkName3}},
			[]v1.Network{{Name: testNetworkName1}, {Name: testNetworkName2}, {Name: testNetworkName3}},
			expectToChange,
		),
		Entry("when vmi interfaces have an extra SRIOV interface which requires hotplug, change is not required since SRIOV hotplug to a pod is not supported",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: testNetworkName1}),
				libvmi.WithInterface(v1.Interface{Name: testNetworkName2, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}),
				libvmi.WithInterface(v1.Interface{Name: testNetworkName3, InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}}}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName3}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName1}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName2}),
			),
			&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					networkv1.NetworkStatusAnnot: `[
						{"interface":"net1", "name":"red-net", "namespace": "default"},
						{"interface":"net2", "name":"blue-net", "namespace": "default"}
					]`,
				}},
			},
			nil,
			nil,
			expectNoChange,
		),
		Entry("when a vmi interface has state set to `absent`, requiring hotunplug",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: testNetworkName1}),
				libvmi.WithInterface(v1.Interface{Name: testNetworkName2, State: v1.InterfaceStateAbsent}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName1}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName2}),
			),
			&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					networkv1.NetworkStatusAnnot: `[
						{"interface":"pod1", "name":"red-net", "namespace": "default"},
						{"interface":"pod2", "name":"blue-net", "namespace": "default"}
					]`,
				}},
			},
			[]v1.Interface{{Name: testNetworkName1}},
			[]v1.Network{{Name: testNetworkName1}},
			expectToChange,
		),
		Entry("when vmi interfaces have an interface to hotplug and one to hot-unplug, given hashed names",
			libvmi.New(
				libvmi.WithInterface(v1.Interface{Name: testNetworkName1, State: v1.InterfaceStateAbsent}),
				libvmi.WithInterface(v1.Interface{Name: testNetworkName2}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName1}),
			),
			&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					networkv1.NetworkStatusAnnot: `[
						{"interface":"pod1a2b3c", "name":"red-net", "namespace": "default"}
					]`,
				}},
			},
			[]v1.Interface{{Name: testNetworkName2}},
			[]v1.Network{{Name: testNetworkName2}},
			expectToChange,
		),
	)
	DescribeTable("apply dynamic interface request on VMI",
		func(vmiForVM, currentVMI, expectedVMI *v1.VirtualMachineInstance, hasOrdinalIfaces bool) {
			vm := VirtualMachineFromVMI(currentVMI.Name, vmiForVM, true)
			updatedVMI := applyDynamicIfaceRequestOnVMI(vm, currentVMI, hasOrdinalIfaces)
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
				libvmi.WithInterface(v1.Interface{Name: testNetworkName1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Macvtap: &v1.InterfaceMacvtap{}}}),
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

			ifaces, networks := clearDetachedInterfaces(specIfaces, testNetworks, testStatusIfaces)

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

func withInterfaceStatus(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Status.Interfaces = append(
			vmi.Status.Interfaces, ifaceStatus,
		)
	}
}
