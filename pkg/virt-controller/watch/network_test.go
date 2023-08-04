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
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/api"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/pointer"
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
			ifaces, nets, exists := calculateDynamicInterfaces(vmi)
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
		func(vmIfaces, vmiIfaces []v1.Interface, ifaceRequests, expectedIfaceRequests []v1.VirtualMachineInterfaceRequest) {
			vmi := api.NewMinimalVMI("test")
			vmi.Spec.Domain.Devices.Interfaces = vmiIfaces
			vm := VirtualMachineFromVMI("test", vmi, true)
			vm.Spec.Template.Spec.Domain.Devices.Interfaces = vmIfaces
			vm.Status.InterfaceRequests = ifaceRequests

			trimDoneInterfaceRequests(vm, vmi)

			Expect(vm.Status.InterfaceRequests).To(Equal(expectedIfaceRequests))
		},
		Entry("have request removed on successful hotplug",
			[]v1.Interface{{Name: "blue"}},
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{},
		),
		Entry("keep interface request for pending hotplug",
			[]v1.Interface{},
			[]v1.Interface{},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
		),
		Entry("keep interface request for pending hotplug, when VMI spec has not been updated yet",
			[]v1.Interface{{Name: "blue"}},
			[]v1.Interface{},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
		),
		Entry("when VMI has been updated but the VM isn't, hotplug request is ignored and kept in the queue",
			[]v1.Interface{},
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
		),
		Entry("have request removed on successful unplug",
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}},
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{},
		),
		Entry("keep interface request for pending unplug",
			[]v1.Interface{{Name: "blue"}},
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
		),
		Entry("keep interface request for pending unplug, when VMI spec has not been updated yet",
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}},
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
		),
		Entry("when VMI has been updated but the VM isn't, unplug request is ignored and kept in the queue",
			[]v1.Interface{{Name: "blue"}},
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
		),
	)

	DescribeTable("spec interfaces",
		func(specIfaces []v1.Interface, statusIfaces []v1.VirtualMachineInstanceNetworkInterface,
			expectedInterfaces []v1.Interface, expectedNetworks []v1.Network) {
			var testNetworks []v1.Network
			for _, iface := range specIfaces {
				testNetworks = append(testNetworks, v1.Network{Name: iface.Name})
			}
			testStatusIfaces := vmispec.IndexInterfacesFromStatus(statusIfaces,
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

	DescribeTable("Stopped VM status interfaces requests",
		func(ifaces []v1.Interface, ifaceRequests, expectedIfaceRequests []v1.VirtualMachineInterfaceRequest) {
			vm := kubecli.NewMinimalVM("test")
			vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
			vm.Spec.Template.Spec.Domain.Devices.Interfaces = ifaces
			vm.Status.InterfaceRequests = ifaceRequests
			vm.Spec.Running = pointer.P(false)

			trimDoneInterfaceRequests(vm, nil)

			Expect(vm.Status.InterfaceRequests).To(Equal(expectedIfaceRequests))
		},
		Entry("request removed on successful unplug",
			[]v1.Interface{{Name: "blue", State: v1.InterfaceStateAbsent}},
			[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{},
		),
		Entry("request removed on successful hotplug",
			[]v1.Interface{{Name: "blue"}},
			[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
			[]v1.VirtualMachineInterfaceRequest{},
		),
	)

	Context("handle dynamic interface requests", func() {
		var clusterConfig *virtconfig.ClusterConfig

		BeforeEach(func() {
			clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
		})

		DescribeTable("for a VM with no interfaces",
			func(requests []v1.VirtualMachineInterfaceRequest, expectedInterfaces []v1.Interface, expectedNetworks []v1.Network, vmi *v1.VirtualMachineInstance) {
				vm := kubecli.NewMinimalVM("test")
				vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
				vm.Status.InterfaceRequests = requests

				Expect(handleDynamicInterfaceRequests(vm, vmi, clusterConfig)).To(Succeed())
				Expect(vm.Spec.Template.Spec.Domain.Devices.Interfaces).To(Equal(expectedInterfaces))
				Expect(vm.Spec.Template.Spec.Networks).To(Equal(expectedNetworks))
			},
			Entry("no request",
				[]v1.VirtualMachineInterfaceRequest{},
				nil,
				nil,
				libvmi.New(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(v1.Interface{Name: "default"}),
				),
			),
			Entry("one hot plug request, a default interface should be added to the VM",
				[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
				[]v1.Interface{
					{Name: "default", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
					{Name: "blue", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				},
				[]v1.Network{
					*v1.DefaultPodNetwork(),
					{Name: "blue", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
				},
				libvmi.New(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(v1.Interface{Name: "default"}),
				),
			),
			Entry("one hot plug request, VMI has no pod network",
				[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
				[]v1.Interface{
					{Name: "blue", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				},
				[]v1.Network{
					{Name: "blue", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
				},
				libvmi.New(),
			),
			Entry("one hot unplug request",
				[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "blue"}}},
				nil,
				nil,
				libvmi.New(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(v1.Interface{Name: "default"}),
				),
			),
		)

		DescribeTable("for a VM with existing interfaces",
			func(requests []v1.VirtualMachineInterfaceRequest, expectedInterfaces []v1.Interface, expectedNetworks []v1.Network) {
				vm := kubecli.NewMinimalVM("test")
				vm.Spec.Template = &v1.VirtualMachineInstanceTemplateSpec{}
				vm.Spec.Template.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "oldNet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}}
				vm.Spec.Template.Spec.Networks = []v1.Network{{Name: "oldNet", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}}}
				vm.Status.InterfaceRequests = requests

				vmi := libvmi.New(
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithInterface(v1.Interface{Name: "default"}),
					libvmi.WithNetwork(&v1.Network{Name: "oldNet", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}}),
					libvmi.WithInterface(v1.Interface{Name: "oldNet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}}),
				)
				Expect(handleDynamicInterfaceRequests(vm, vmi, clusterConfig)).To(Succeed())
				Expect(vm.Spec.Template.Spec.Domain.Devices.Interfaces).To(Equal(expectedInterfaces))
				Expect(vm.Spec.Template.Spec.Networks).To(Equal(expectedNetworks))
			},
			Entry("no requests",
				[]v1.VirtualMachineInterfaceRequest{},
				[]v1.Interface{
					{Name: "oldNet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				},
				[]v1.Network{
					{Name: "oldNet", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
				},
			),
			Entry("one hot plug request",
				[]v1.VirtualMachineInterfaceRequest{{AddInterfaceOptions: &v1.AddInterfaceOptions{Name: "blue"}}},
				[]v1.Interface{
					{Name: "oldNet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
					{Name: "blue", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}},
				},
				[]v1.Network{
					{Name: "oldNet", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
					{Name: "blue", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
				},
			),
			Entry("one hot unplug request",
				[]v1.VirtualMachineInterfaceRequest{{RemoveInterfaceOptions: &v1.RemoveInterfaceOptions{Name: "oldNet"}}},
				[]v1.Interface{
					{Name: "oldNet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Bridge: &v1.InterfaceBridge{}}, State: v1.InterfaceStateAbsent},
				},
				[]v1.Network{
					{Name: "oldNet", NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}}},
				},
			),
		)
	})
})

func withInterfaceStatus(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Status.Interfaces = append(
			vmi.Status.Interfaces, ifaceStatus,
		)
	}
}
