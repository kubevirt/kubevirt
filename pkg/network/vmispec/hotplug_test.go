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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package vmispec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("utilitary funcs to identify attachments to hotplug", func() {
	Context("NetworksToHotplugWhosePodIfacesAreReady", func() {
		const (
			guestIfaceName = "eno123"
			nadName        = "nad1"
			networkName    = "n1"
		)
		DescribeTable("NetworksToHotplugWhosePodIfacesAreReady", func(vmi *v1.VirtualMachineInstance, networksToHotplug ...v1.Network) {
			Expect(vmispec.NetworksToHotplugWhosePodIfacesAreReady(vmi)).To(ConsistOf(networksToHotplug))
		},
			Entry("VMI without networks in spec does not have anything to hotplug", libvmi.New()),
			Entry("VMI with networks in spec, but not marked as ready in the status are *not* subject to hotplug",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
					libvmi.WithNetwork(libvmi.MultusNetwork(networkName, nadName)),
				),
			),
			Entry("VMI with networks in spec, marked as ready in the status, but not yet available in the domain *is* subject to hotplug",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
					libvmi.WithNetwork(libvmi.MultusNetwork(networkName, nadName)),
					libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
						Name: networkName, InterfaceName: guestIfaceName, InfoSource: vmispec.InfoSourceMultusStatus,
					}))),
				),
				v1.Network{
					Name: networkName,
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: nadName,
						},
					},
				},
			),
			Entry("VMI with networks in spec, marked as ready in the status, but already present in the domain *not* subject to hotplug",
				libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(networkName)),
					libvmi.WithNetwork(libvmi.MultusNetwork(networkName, nadName)),
					libvmistatus.WithStatus(libvmistatus.New(libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
						Name: networkName, InterfaceName: guestIfaceName, InfoSource: vmispec.NewInfoSource(vmispec.InfoSourceDomain, vmispec.InfoSourceMultusStatus),
					}))),
				),
			),
		)
	})

	Context("CalculateInterfacesAndNetworksForMultusAnnotationUpdate", func() {
		const (
			expectNoChange = false
			expectToChange = !expectNoChange

			testNetworkName1 = "testnet1"
			testNetworkName2 = "testnet2"
			testNetworkName3 = "testnet3"
			testNetworkName4 = "testnet4"
		)
		DescribeTable("calculate if changes are required",
			func(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod, expIfaces []v1.Interface, expNets []v1.Network, expToChange bool) {
				ifaces, nets, exists := vmispec.CalculateInterfacesAndNetworksForMultusAnnotationUpdate(vmi)
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
	})
})

func withInterfaceStatus(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Status.Interfaces = append(
			vmi.Status.Interfaces, ifaceStatus,
		)
	}
}
