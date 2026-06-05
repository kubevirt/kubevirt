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

package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	network "kubevirt.io/kubevirt/pkg/network/setup"

	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmistatus "kubevirt.io/kubevirt/pkg/libvmi/status"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
)

var _ = Describe("Network setup filters", func() {
	Context("FilterNetsForVMStartup", func() {
		It("Should return a list non-absent networks", func() {
			const absentNetName = "absent-net"
			absentIface := libvmi.InterfaceDeviceWithBridgeBinding(absentNetName)
			absentIface.State = v1.InterfaceStateAbsent

			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(absentIface),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(absentNetName, "somenad")),
			)

			Expect(network.FilterNetsForVMStartup(vmi)).To(Equal([]v1.Network{*v1.DefaultPodNetwork()}))
		})
	})

	Context("FilterNetsForLiveUpdate", func() {
		const (
			net1Name = "net1"
			net2Name = "net2"
			nad1Name = "nad1"
			nad2Name = "nad2"
		)

		multusAndDomainInfoSource := vmispec.NewInfoSource(vmispec.InfoSourceMultusStatus, vmispec.InfoSourceDomain)

		It("Should return an empty list when the VMI has no networks", func() {
			vmi := libvmi.New(libvmi.WithAutoAttachPodInterface(false))
			Expect(network.FilterNetsForLiveUpdate(vmi)).To(BeEmpty())
		})

		It("Should return an empty list when interface status is not reported", func() {
			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net1Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(net1Name, nad1Name)),
			)

			Expect(network.FilterNetsForLiveUpdate(vmi)).To(BeEmpty())
		})

		It("Should return an empty list when there are no networks to hot plug/unplug", func() {
			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net1Name)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net2Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(net1Name, nad1Name)),
				libvmi.WithNetwork(libvmi.MultusNetwork(net2Name, nad2Name)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       net1Name,
							InfoSource: multusAndDomainInfoSource,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       net2Name,
							InfoSource: multusAndDomainInfoSource,
						}),
					),
				),
			)

			Expect(network.FilterNetsForLiveUpdate(vmi)).To(BeEmpty())
		})

		It("Should return a network to hotplug when the interface exists in pod but not in the domain", func() {
			multusAndDomainInfoSource := vmispec.NewInfoSource(vmispec.InfoSourceMultusStatus, vmispec.InfoSourceDomain)
			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net1Name)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net2Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(net1Name, nad1Name)),
				libvmi.WithNetwork(libvmi.MultusNetwork(net2Name, nad2Name)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       net1Name,
							InfoSource: multusAndDomainInfoSource,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       net2Name,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
					),
				),
			)

			Expect(network.FilterNetsForLiveUpdate(vmi)).To(Equal([]v1.Network{*libvmi.MultusNetwork(net2Name, nad2Name)}))
		})

		It("Should return a network to hotunplug when its interface is marked as absent and not in the domain", func() {
			absentIface1 := libvmi.InterfaceDeviceWithBridgeBinding(net1Name)
			absentIface1.State = v1.InterfaceStateAbsent

			absentIface2 := libvmi.InterfaceDeviceWithBridgeBinding(net2Name)
			absentIface2.State = v1.InterfaceStateAbsent

			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(absentIface1),
				libvmi.WithInterface(absentIface2),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(net1Name, nad1Name)),
				libvmi.WithNetwork(libvmi.MultusNetwork(net2Name, nad2Name)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       net1Name,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       net2Name,
							InfoSource: multusAndDomainInfoSource,
						}),
					),
				),
			)

			Expect(network.FilterNetsForLiveUpdate(vmi)).To(Equal([]v1.Network{*libvmi.MultusNetwork(net1Name, nad1Name)}))
		})

		It("Should return networks to hotplug and hotunplug", func() {
			absentIface := libvmi.InterfaceDeviceWithBridgeBinding(net1Name)
			absentIface.State = v1.InterfaceStateAbsent

			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(absentIface),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithBridgeBinding(net2Name)),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(net1Name, nad1Name)),
				libvmi.WithNetwork(libvmi.MultusNetwork(net2Name, nad2Name)),
				libvmistatus.WithStatus(
					libvmistatus.New(
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       "default",
							InfoSource: vmispec.InfoSourceDomain,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       net1Name,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
						libvmistatus.WithInterfaceStatus(v1.VirtualMachineInstanceNetworkInterface{
							Name:       net2Name,
							InfoSource: vmispec.InfoSourceMultusStatus,
						}),
					),
				),
			)

			Expect(network.FilterNetsForLiveUpdate(vmi)).To(ConsistOf(
				*libvmi.MultusNetwork(net1Name, nad1Name),
				*libvmi.MultusNetwork(net2Name, nad2Name),
			))
		})
	})

	Context("FilterNetsForMigrationTarget", func() {
		It("Should return a list of all networks - no matter their interface state", func() {
			const absentNetName = "absent-net"
			absentIface := libvmi.InterfaceDeviceWithBridgeBinding(absentNetName)
			absentIface.State = v1.InterfaceStateAbsent

			vmi := libvmi.New(
				libvmi.WithInterface(*v1.DefaultBridgeNetworkInterface()),
				libvmi.WithInterface(absentIface),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithNetwork(libvmi.MultusNetwork(absentNetName, "somenad")),
			)

			Expect(network.FilterNetsForMigrationTarget(vmi)).To(Equal(
				[]v1.Network{
					*v1.DefaultPodNetwork(),
					*libvmi.MultusNetwork(absentNetName, "somenad"),
				},
			))
		})
	})
})
