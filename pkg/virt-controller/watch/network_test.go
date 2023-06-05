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

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("Network interface hot{un}plug", func() {
	const (
		expectNoChange = false
		expectToChange = !expectNoChange

		testNetworkName1 = "testnet1"
		testNetworkName2 = "testnet2"
	)
	DescribeTable("calculate if changes are required",

		func(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod, expIfaces []v1.Interface, expNets []v1.Network, expToChange bool) {
			ifaces, nets, exists := calculateDynamicInterfaces(vmi, pod)
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
				libvmi.WithInterface(v1.Interface{Name: testNetworkName1}),
				libvmi.WithInterface(v1.Interface{Name: testNetworkName2}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName1}),
				libvmi.WithNetwork(&v1.Network{Name: testNetworkName2}),
				withInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{Name: testNetworkName1}),
			),
			&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
					networkv1.NetworkStatusAnnot: `[
						{"interface":"net1", "name":"red-net", "namespace": "default"}
					]`,
				}},
			},
			[]v1.Interface{{Name: testNetworkName1}, {Name: testNetworkName2}},
			[]v1.Network{{Name: testNetworkName1}, {Name: testNetworkName2}},
			expectToChange,
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
		Entry("when a vmi interface has state set to `absent`, but pod iface name is ordinal",
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
						{"interface":"net1", "name":"red-net", "namespace": "default"},
						{"interface":"net2", "name":"blue-net", "namespace": "default"}
					]`,
				}},
			},
			nil,
			nil,
			expectNoChange,
		),
		Entry("when vmi interfaces have an interface to hotplug and one to hot-unplug, given ordinal names",
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
						{"interface":"net1", "name":"red-net", "namespace": "default"}
					]`,
				}},
			},
			[]v1.Interface{{Name: testNetworkName1, State: v1.InterfaceStateAbsent}, {Name: testNetworkName2}},
			[]v1.Network{{Name: testNetworkName1}, {Name: testNetworkName2}},
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

	DescribeTable("VM status interfaces requests",
		func(ifaces []v1.Interface, ifaceRequests, expectedIfaceRequests []v1.VirtualMachineInterfaceRequest) {
			vm := kubecli.NewMinimalVM("test")
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
			vm.Spec.Template.Spec.Domain.Devices.Interfaces = ifaces
			vm.Status.InterfaceRequests = ifaceRequests

			trimDoneInterfaceRequests(vm)

			Expect(vm.Status.InterfaceRequests).To(Equal(expectedIfaceRequests))
		},
		Entry("have request removed on successful hotplug",
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{},
		),
		Entry("keep interface request for pending hotplug",
			[]v1.Interface{},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
		),
		Entry("have request removed on successful unplug",
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{},
		),
		Entry("keep interface request for pending unplug",
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
		),
	)
})

func withInterfaceStatus(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Status.Interfaces = append(
			vmi.Status.Interfaces, ifaceStatus,
		)
	}
}
