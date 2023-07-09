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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"

	dutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	netsriov "kubevirt.io/kubevirt/pkg/network/sriov"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("netstat", func() {
	var setup testSetup

	BeforeEach(func() {
		setup = newTestSetup()
	})

	AfterEach(func() { Expect(setup.Cleanup()).To(Succeed()) })

	It("run status with no domain", func() {
		Expect(setup.NetStat.UpdateStatus(setup.Vmi, nil)).To(Succeed())
	})

	It("volatile cache is updated based on non-volatile cache", func() {
		const (
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
		)

		Expect(
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, ""),
				primaryPodIPv4,
			),
		).To(Succeed())

		Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

		Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeTrue())
	})

	Context("with volatile cache", func() {
		const (
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
			primaryPodIPv6     = "fd10:244::8c4c"
			primaryGaIPv4      = "2.2.2.1"
			primaryGaIPv6      = "fd20:244::8c4c"
			primaryMAC         = "1C:CE:C0:01:BE:E7"
			primaryIfaceName   = "eth0"

			secondaryNetworkName = "secondary"
			secondaryPodIPv4     = "1.1.1.2"
			secondaryPodIPv6     = "fd10:244::8c4e"
			secondaryGaIPv4      = "2.2.2.2"
			secondaryGaIPv6      = "fd20:244::8c4e"
			secondaryMAC         = "1C:CE:C0:01:BE:E9"
			secondaryIfaceName   = "eth1"
		)

		BeforeEach(func() {
			setup = newTestSetupWithVolatileCache()
		})

		It("run status and expect two interfaces/networks to be reported (without guest-agent)", func() {
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, ""),
					primaryPodIPv4, primaryPodIPv6,
				),
			).To(Succeed())
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(secondaryNetworkName),
					newVMISpecMultusNetwork(secondaryNetworkName),
					newDomainSpecIface(secondaryNetworkName, ""),
					secondaryPodIPv4, secondaryPodIPv6,
				),
			).To(Succeed())

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, "", "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryPodIPv4, secondaryPodIPv6}, "", "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
			}), "the pod IP/s should be reported in the status")

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeTrue())
			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, secondaryNetworkName)).To(BeTrue())
		})

		It("run status and expect interface/network to be reported with the right queue count (without guest-agent)", func() {
			var queueCount uint = 8
			domainSpecInterface := newDomainSpecIface(primaryNetworkName, "")
			domainSpecInterface.Driver = &api.InterfaceDriver{Name: "vhost", Queues: &queueCount}

			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					domainSpecInterface,
					primaryPodIPv4, primaryPodIPv6,
				),
			).To(Succeed())

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, "", "", netvmispec.InfoSourceDomain, int32(queueCount)),
			}), "queue count and the pod IP/s should be reported in the status")

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeTrue())
		})

		It("run status and expect 2 interfaces to be reported based on guest-agent data", func() {
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, primaryMAC),
					primaryPodIPv4, primaryPodIPv6,
				),
			).To(Succeed())
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(secondaryNetworkName),
					newVMISpecMultusNetwork(secondaryNetworkName),
					newDomainSpecIface(secondaryNetworkName, secondaryMAC),
					secondaryPodIPv4, secondaryPodIPv6,
				),
			).To(Succeed())

			setup.addGuestAgentInterfaces(
				newDomainStatusIface([]string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName),
				newDomainStatusIface([]string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC, secondaryIfaceName),
			)

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName, netvmispec.InfoSourceDomainAndGA, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC, secondaryIfaceName, netvmispec.InfoSourceDomainAndGA, netsetup.DefaultInterfaceQueueCount),
			}), "the guest-agent IP/s should be reported in the status")

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeTrue())
			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, secondaryNetworkName)).To(BeTrue())
		})

		It("run status and expect 2 interfaces to be reported based on multus status and guest-agent data", func() {
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, primaryMAC),
					primaryPodIPv4, primaryPodIPv6,
				),
			).To(Succeed())
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(secondaryNetworkName),
					newVMISpecMultusNetwork(secondaryNetworkName),
					newDomainSpecIface(secondaryNetworkName, secondaryMAC),
					secondaryPodIPv4, secondaryPodIPv6,
				),
			).To(Succeed())

			setup.addGuestAgentInterfaces(
				newDomainStatusIface([]string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName),
				newDomainStatusIface([]string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC, secondaryIfaceName),
			)

			setup.Vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{Name: primaryNetworkName, InfoSource: netvmispec.InfoSourceMultusStatus},
				{Name: secondaryNetworkName, InfoSource: netvmispec.InfoSourceMultusStatus},
				// Interfaces that exist in the status but are not detected from the domain or GA are dropped.
				{Name: "foo", InfoSource: netvmispec.InfoSourceMultusStatus},
			}

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			infoSourceDomainGAMultus := netvmispec.NewInfoSource(
				netvmispec.InfoSourceDomain, netvmispec.InfoSourceGuestAgent, netvmispec.InfoSourceMultusStatus)
			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName, infoSourceDomainGAMultus, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC, secondaryIfaceName, infoSourceDomainGAMultus, netsetup.DefaultInterfaceQueueCount),
			}), "the guest-agent IP/s should be reported in the status")

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeTrue())
			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, secondaryNetworkName)).To(BeTrue())
		})

		It("run status and expect an interfaces (with masquerade) to be reported based on pod & guest-agent data", func() {
			// Guest data collected by the guest-agent
			const (
				primaryGaIPv4 = "2.2.2.1"
				primaryGaIPv6 = "fd20:244::8c4c"

				primaryMAC = "1C:CE:C0:01:BE:E7"
			)

			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithMasqueradeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, primaryMAC),
					primaryPodIPv4, primaryPodIPv6,
				),
			).To(Succeed())
			setup.addGuestAgentInterfaces(
				newDomainStatusIface([]string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, "eth0"),
			)

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, primaryMAC, "eth0", netvmispec.InfoSourceDomainAndGA, netsetup.DefaultInterfaceQueueCount),
			}), "the pod IP/s should be reported in the status")

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeTrue())
		})

		It("should update existing interface status with MAC from the domain", func() {
			const (
				origMAC      = "C0:01:BE:E7:15:G0:0D"
				newDomainMAC = "1C:CE:C0:01:BE:E7"
			)

			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, newDomainMAC),
					primaryPodIPv4, primaryPodIPv6,
				),
			).To(Succeed())

			setup.Vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{
					IP:   primaryPodIPv4,
					IPs:  []string{primaryPodIPv4, primaryPodIPv6},
					MAC:  origMAC,
					Name: primaryNetworkName,
				},
			}

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, newDomainMAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
			}), "the pod IP/s should be reported in the status")
		})

		It("runs teardown that clears volatile cache", func() {
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, ""),
					primaryPodIPv4, primaryPodIPv6,
				),
			).To(Succeed())
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(secondaryNetworkName),
					newVMISpecMultusNetwork(secondaryNetworkName),
					newDomainSpecIface(secondaryNetworkName, ""),
					secondaryPodIPv4, secondaryPodIPv6,
				),
			).To(Succeed())

			setup.NetStat.Teardown(setup.Vmi)

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeFalse())
			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, secondaryNetworkName)).To(BeFalse())
		})
	})

	It("should update existing interface status with IP from the guest-agent", func() {
		const (
			primaryNetworkName = "primary"
			primaryIfaceName   = "eth0"

			origIPv4 = "1.1.1.1"
			origIPv6 = "fd10:1111::1111"
			origMAC  = "C0:01:BE:E7:15:G0:0D"

			newGaIPv4 = "2.2.2.2"
			newGaIPv6 = "fd20:2222::2222"
		)

		Expect(
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, origMAC),
				origIPv4, origIPv6,
			),
		).To(Succeed())
		setup.Vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
			{
				IP:   origIPv4,
				IPs:  []string{origIPv4, origIPv6},
				MAC:  origMAC,
				Name: primaryNetworkName,
			},
		}

		setup.addGuestAgentInterfaces(
			newDomainStatusIface([]string{newGaIPv4, newGaIPv6}, origMAC, primaryIfaceName),
		)

		Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

		Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(primaryNetworkName, []string{newGaIPv4, newGaIPv6}, origMAC, primaryIfaceName, netvmispec.InfoSourceDomainAndGA, netsetup.DefaultInterfaceQueueCount),
		}), "the pod IP/s should be reported in the status")
	})

	It("should report SR-IOV interface when guest-agent is inactive and no other interface exists", func() {
		const (
			networkName = "sriov-network"
			ifaceMAC    = "C0:01:BE:E7:15:G0:0D"
		)

		sriovIface := newVMISpecIfaceWithSRIOVBinding(networkName)
		// The MAC is specified intentionally to illustrate that it is not reported if the GA is not present.
		sriovIface.MacAddress = ifaceMAC
		setup.addSRIOVNetworkInterface(
			sriovIface,
			newVMISpecMultusNetwork(networkName),
		)

		Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

		Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(networkName, nil, ifaceMAC, "", netvmispec.InfoSourceDomain, netsetup.UnknownInterfaceQueueCount),
		}), "the SR-IOV interface should be reported in the status.")
	})

	It("should report SR-IOV interface when guest-agent is inactive and a regular interface exists", func() {
		const (
			networkName        = "sriov-network"
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
		)

		Expect(
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, ""),
				primaryPodIPv4,
			),
		).To(Succeed())

		sriovIface := newVMISpecIfaceWithSRIOVBinding(networkName)
		setup.addSRIOVNetworkInterface(
			sriovIface,
			newVMISpecMultusNetwork(networkName),
		)

		Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

		Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4}, "", "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
			newVMIStatusIface(networkName, nil, "", "", netvmispec.InfoSourceDomain, netsetup.UnknownInterfaceQueueCount),
		}), "the SR-IOV interface should be reported in the status.")
	})

	It("should report SR-IOV interface with MAC and network name, based on VMI spec and guest-agent data", func() {
		const (
			networkName    = "sriov-network"
			ifaceMAC       = "C0:01:BE:E7:15:G0:0D"
			guestIfaceName = "eth1"
		)

		sriovIface := newVMISpecIfaceWithSRIOVBinding(networkName)
		sriovIface.MacAddress = ifaceMAC
		setup.addSRIOVNetworkInterface(
			sriovIface,
			newVMISpecMultusNetwork(networkName),
		)
		setup.addGuestAgentInterfaces(
			newDomainStatusIface(nil, ifaceMAC, guestIfaceName),
		)

		Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

		Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(networkName, nil, ifaceMAC, guestIfaceName, netvmispec.InfoSourceDomainAndGA, netsetup.UnknownInterfaceQueueCount),
		}), "the SR-IOV interface should be reported in the status, associated to the network")
	})

	When("the desired state (VMI spec) is not in sync with the state in the guest (guest-agent)", func() {
		const (
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
			primaryPodIPv6     = "fd10:244::8c4c"
			primaryGaIPv4      = "2.2.2.1"
			primaryGaIPv6      = "fd20:244::8c4c"
			primaryMAC         = "1C:CE:C0:01:BE:E7"
			primaryIfaceName   = "eth0"

			secondaryNetworkName = "secondary"
			secondaryPodIPv4     = "1.1.1.2"
			secondaryPodIPv6     = "fd10:244::8c4e"
			secondaryGaIPv4      = secondaryPodIPv4
			secondaryGaIPv6      = secondaryPodIPv6
			secondaryMAC         = "1C:CE:C0:01:BE:E9"
			secondaryIfaceName   = "eth1"

			newMAC1 = "fd20:000::0001"
			newMAC2 = "fd20:000::0002"
		)

		BeforeEach(func() {
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithMasqueradeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, primaryMAC),
					primaryPodIPv4, primaryPodIPv6,
				),
			).To(Succeed())
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(secondaryNetworkName),
					newVMISpecMultusNetwork(secondaryNetworkName),
					newDomainSpecIface(secondaryNetworkName, secondaryMAC),
					secondaryPodIPv4, secondaryPodIPv6,
				),
			).To(Succeed())
		})

		It("reports masquerade and bridge interfaces with their MAC changed in the guest", func() {
			setup.addGuestAgentInterfaces(
				newDomainStatusIface([]string{primaryGaIPv4, primaryGaIPv6}, newMAC1, primaryIfaceName),
				newDomainStatusIface([]string{secondaryGaIPv4, secondaryGaIPv6}, newMAC2, secondaryIfaceName),
			)
			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(ConsistOf([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, primaryMAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface(secondaryNetworkName, nil, secondaryMAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface("", []string{primaryGaIPv4, primaryGaIPv6}, newMAC1, primaryIfaceName, netvmispec.InfoSourceGuestAgent, netsetup.UnknownInterfaceQueueCount),
				newVMIStatusIface("", []string{secondaryGaIPv4, secondaryGaIPv6}, newMAC2, secondaryIfaceName, netvmispec.InfoSourceGuestAgent, netsetup.UnknownInterfaceQueueCount),
			}))
		})

		It("reports a new interface that appeared in the guest", func() {
			const (
				newGaIPv4    = "3.3.3.3"
				newGaIPv6    = "fd20:333::3333"
				newIfaceName = "eth3"
			)
			setup.addGuestAgentInterfaces(
				newDomainStatusIface([]string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName),
				newDomainStatusIface([]string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC, secondaryIfaceName),
				newDomainStatusIface([]string{newGaIPv4, newGaIPv6}, newMAC1, newIfaceName),
			)
			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(ConsistOf([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, primaryMAC, primaryIfaceName, netvmispec.InfoSourceDomainAndGA, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryPodIPv4, secondaryPodIPv6}, secondaryMAC, secondaryIfaceName, netvmispec.InfoSourceDomainAndGA, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface("", []string{newGaIPv4, newGaIPv6}, newMAC1, newIfaceName, netvmispec.InfoSourceGuestAgent, netsetup.UnknownInterfaceQueueCount),
			}))
		})

		It("reports that an interface is not seen in the guest", func() {
			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(ConsistOf([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, primaryMAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryPodIPv4, secondaryPodIPv6}, secondaryMAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
			}))
		})
	})

	Context("backward compatability", func() {
		const (
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
			primaryGaIPv4      = "2.2.2.1"
			primaryMAC         = "1C:CE:C0:01:BE:E7"
			primaryIfaceName   = "eth0"
		)

		It("reports no infoSource when virt-launcher is old and only the domain data exists (but GA is active)", func() {
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, primaryMAC),
					primaryPodIPv4,
				),
			).To(Succeed())

			// The existence of an empty interfaceName is the outcome of an old virt-launcher merging the domain and
			// GA data, including the domain-only data in.
			primaryIface := newDomainStatusIface(nil, primaryMAC, "")
			setup.addGuestAgentInterfaces(primaryIface)

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, nil, primaryMAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
			}))
		})

		It("reports no infoSource when virt-launcher is old and both the domain & GA data exists", func() {
			Expect(
				setup.addNetworkInterface(
					newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
					newVMISpecPodNetwork(primaryNetworkName),
					newDomainSpecIface(primaryNetworkName, primaryMAC),
					primaryPodIPv4,
				),
			).To(Succeed())

			// The existence of an interfaceName is the outcome of an old virt-launcher merging the domain and
			// GA data, where an association could be made between the domain and the guest agent report.
			// Note: This is correct for new virt-launchers as well.
			primaryIface := newDomainStatusIface([]string{primaryGaIPv4}, primaryMAC, primaryIfaceName)
			setup.addGuestAgentInterfaces(primaryIface)

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryGaIPv4}, primaryMAC, primaryIfaceName, netvmispec.InfoSourceDomainAndGA, netsetup.DefaultInterfaceQueueCount),
			}))
		})
	})

	Describe("primary interface position in Status.Interfaces", func() {
		const (
			prNetworkName   = "primary"
			secNetworkName1 = "secondary1"
			secNetworkName2 = "secondary2"
			podIP           = "1.1.1.1"
			MAC             = "1C:CE:C0:01:BE:E7"
			MAC1            = "1C:CE:C0:01:BE:E8"
			MAC2            = "1C:CE:C0:01:BE:E9"
		)

		const (
			PRIMARY_IFACE_IND = iota
			SECONDARY_IFACE1_IND
			SECONDARY_IFACE2_IND
		)

		var vmiSpecIfaces []v1.Interface
		var vmiSpecNetworks []v1.Network
		var domainSpecIfaces []api.Interface

		BeforeEach(func() {
			vmiSpecIfaces = []v1.Interface{
				newVMISpecIfaceWithMasqueradeBinding(prNetworkName),
				newVMISpecIfaceWithBridgeBinding(secNetworkName1),
				newVMISpecIfaceWithBridgeBinding(secNetworkName2),
			}
			vmiSpecNetworks = []v1.Network{
				newVMISpecPodNetwork(prNetworkName),
				newVMISpecMultusNetwork(secNetworkName1),
				newVMISpecMultusNetwork(secNetworkName2),
			}
			domainSpecIfaces = []api.Interface{
				newDomainSpecIface(prNetworkName, MAC),
				newDomainSpecIface(secNetworkName1, MAC1),
				newDomainSpecIface(secNetworkName2, MAC2),
			}
		})

		DescribeTable("verify primary interface is always first in Status.Interfaces list", func(ifaceIndexArr []int) {
			for index := range ifaceIndexArr {
				Expect(setup.addNetworkInterface(
					vmiSpecIfaces[index],
					vmiSpecNetworks[index],
					domainSpecIfaces[index],
					podIP,
				)).To(Succeed())
			}

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(prNetworkName, []string{podIP}, MAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface(secNetworkName1, []string{podIP}, MAC1, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
				newVMIStatusIface(secNetworkName2, []string{podIP}, MAC2, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
			}))
		},
			Entry("primary interface defined first in spec", []int{PRIMARY_IFACE_IND, SECONDARY_IFACE1_IND, SECONDARY_IFACE2_IND}),
			Entry("primary interface defined last in spec", []int{SECONDARY_IFACE1_IND, SECONDARY_IFACE2_IND, PRIMARY_IFACE_IND}),
			Entry("primary interface defined in the middle in spec", []int{SECONDARY_IFACE1_IND, PRIMARY_IFACE_IND, SECONDARY_IFACE2_IND}),
		)
	})

	It("run status and expect 1 attached iface & 1 detached iface to be reported based on multus status and guest-agent data", func() {
		const (
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
			primaryPodIPv6     = "fd10:244::8c4c"
			primaryGaIPv4      = "2.2.2.1"
			primaryGaIPv6      = "fd20:244::8c4c"
			primaryMAC         = "1C:CE:C0:01:BE:E7"
			primaryIfaceName   = "eth0"

			secondaryNetworkName = "secondary"
		)
		Expect(
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, primaryMAC),
				primaryPodIPv4, primaryPodIPv6,
			),
		).To(Succeed())

		setup.Vmi.Spec.Domain.Devices.Interfaces = append(setup.Vmi.Spec.Domain.Devices.Interfaces,
			newVMISpecIfaceWithBridgeBinding(secondaryNetworkName))
		setup.Vmi.Spec.Networks = append(setup.Vmi.Spec.Networks, newVMISpecMultusNetwork(secondaryNetworkName))

		setup.addGuestAgentInterfaces(
			newDomainStatusIface([]string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName),
		)
		setup.Vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
			{Name: primaryNetworkName, InfoSource: netvmispec.InfoSourceMultusStatus},
			{Name: secondaryNetworkName, InfoSource: netvmispec.InfoSourceMultusStatus},
		}

		Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

		infoSourceDomainGAMultus := netvmispec.NewInfoSource(
			netvmispec.InfoSourceDomain, netvmispec.InfoSourceGuestAgent, netvmispec.InfoSourceMultusStatus)
		Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName, infoSourceDomainGAMultus, netsetup.DefaultInterfaceQueueCount),
			newVMIStatusIface(secondaryNetworkName, nil, "", "", netvmispec.InfoSourceMultusStatus, 0),
		}), "primary and secondary ifaces should exist in status, where secondary iface have multus-status only")
	})

	It("run status and expect iface that doesn't exist in VMI spec to NOT be reported", func() {
		const (
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
			primaryPodIPv6     = "fd10:244::8c4c"
			primaryGaIPv4      = "2.2.2.1"
			primaryGaIPv6      = "fd20:244::8c4c"
			primaryMAC         = "1C:CE:C0:01:BE:E7"
			primaryIfaceName   = "eth0"

			secondaryNetworkName = "secondary"
		)

		Expect(
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, primaryMAC),
				primaryPodIPv4, primaryPodIPv6,
			),
		).To(Succeed())

		setup.addGuestAgentInterfaces(
			newDomainStatusIface([]string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName),
		)

		setup.Vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
			{Name: primaryNetworkName, InfoSource: netvmispec.InfoSourceMultusStatus},
			{Name: secondaryNetworkName, InfoSource: netvmispec.InfoSourceMultusStatus},
		}

		Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

		infoSourceDomainGAMultus := netvmispec.NewInfoSource(
			netvmispec.InfoSourceDomain, netvmispec.InfoSourceGuestAgent, netvmispec.InfoSourceMultusStatus)
		Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, primaryIfaceName, infoSourceDomainGAMultus, netsetup.DefaultInterfaceQueueCount),
		}), "only primary should exist in status since secondary iface not exist in spec")
	})

	Context("misc scenario", func() {
		const (
			networkName = "primary"
			MAC         = "1C:CE:C0:01:BE:E7"
		)

		It("has interface in domain spec but not in VMI spec", func() {
			setup.Domain.Spec.Devices.Interfaces = append(setup.Domain.Spec.Devices.Interfaces, newDomainSpecIface(networkName, MAC))

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(networkName, nil, MAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
			}))
		})

		It("has interface in VMI spec but not in domain spec", func() {
			setup.Vmi.Spec.Domain.Devices.Interfaces = append(setup.Vmi.Spec.Domain.Devices.Interfaces, newVMISpecIfaceWithBridgeBinding(networkName))
			setup.Vmi.Spec.Networks = append(setup.Vmi.Spec.Networks, newVMISpecPodNetwork(networkName))

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(BeEmpty())
		})

		It("has interface in VMI and domain specs, but not in filesystem cache", func() {
			setup.Vmi.Spec.Domain.Devices.Interfaces = append(setup.Vmi.Spec.Domain.Devices.Interfaces, newVMISpecIfaceWithBridgeBinding(networkName))
			setup.Vmi.Spec.Networks = append(setup.Vmi.Spec.Networks, newVMISpecPodNetwork(networkName))
			setup.Domain.Spec.Devices.Interfaces = append(setup.Domain.Spec.Devices.Interfaces, newDomainSpecIface(networkName, MAC))

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(networkName, nil, MAC, "", netvmispec.InfoSourceDomain, netsetup.DefaultInterfaceQueueCount),
			}))
		})

		It("has interface only in cache but not in any spec", func() {
			const (
				podIPv4 = "1.1.1.1"
				podIPv6 = "fd10:244::8c4c"
			)

			Expect(setup.addFSCacheInterface(networkName, podIPv4, podIPv6)).To(Succeed())

			Expect(setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)).To(Succeed())

			Expect(setup.Vmi.Status.Interfaces).To(BeEmpty())
		})
	})
})

type testSetup struct {
	Vmi     *v1.VirtualMachineInstance
	Domain  *api.Domain
	NetStat *netsetup.NetStat

	cacheCreator  *tempCacheCreator
	podIfaceCache cache.PodInterfaceCache

	// There are two types of caches used: virt-launcher/pod filesystem & virt-handler in-memory (volatile).
	// volatileCache flag marks that the setup should also populate the volatile cache when a network interface is added.
	volatileCache bool
}

func newTestSetupWithVolatileCache() testSetup {
	setup := newTestSetup()
	setup.volatileCache = true
	return setup
}

func newTestSetup() testSetup {
	var cacheCreator tempCacheCreator
	const uid = "123"
	vmi := &v1.VirtualMachineInstance{}
	vmi.UID = uid
	dutils.MockDefaultOwnershipManager()

	return testSetup{
		Vmi:           vmi,
		Domain:        &api.Domain{},
		NetStat:       netsetup.NewNetStateWithCustomFactory(&cacheCreator),
		cacheCreator:  &cacheCreator,
		podIfaceCache: cache.NewPodInterfaceCache(&cacheCreator, uid),
	}
}

// addNetworkInterface is adding a regular[*] network configuration which adds a vNIC.
// This consist of 4 entities and an optional pod volatile cache:
// - vmi spec interface
// - vmi spec network
// - domain spec interface
// - virt-launcher/pod filesystem cache
//
// [*] Non SR-IOV
//
// Guest Agent interface report is not included and if required should be added through `addGuestAgentInterfaces`.
func (t *testSetup) addNetworkInterface(vmiIface v1.Interface, vmiNetwork v1.Network, domainIface api.Interface, podIPs ...string) error {
	if !(vmiIface.Name == vmiNetwork.Name && vmiIface.Name == domainIface.Alias.GetName()) {
		panic("network name must be the same")
	}
	t.Vmi.Spec.Domain.Devices.Interfaces = append(t.Vmi.Spec.Domain.Devices.Interfaces, vmiIface)
	t.Vmi.Spec.Networks = append(t.Vmi.Spec.Networks, vmiNetwork)

	t.Domain.Spec.Devices.Interfaces = append(t.Domain.Spec.Devices.Interfaces, domainIface)

	if err := t.addFSCacheInterface(vmiNetwork.Name, podIPs...); err != nil {
		return err
	}

	if t.volatileCache {
		podCacheInterface := makePodCacheInterface(vmiNetwork.Name, podIPs...)
		t.NetStat.CachePodInterfaceVolatileData(t.Vmi, vmiNetwork.Name, podCacheInterface)
	}
	return nil
}

// addSRIOVNetworkInterface is adding a SR-IOV network configuration which adds a hostdevice to the guest.
// This consist of 2 entities:
// - vmi spec interface
// - vmi spec network
//
// Guest Agent interface report is not included and if required should be added through `addGuestAgentInterfaces`.
func (t *testSetup) addSRIOVNetworkInterface(vmiIface v1.Interface, vmiNetwork v1.Network) {
	if vmiIface.Name != vmiNetwork.Name {
		panic("network name must be the same")
	}
	t.Vmi.Spec.Domain.Devices.Interfaces = append(t.Vmi.Spec.Domain.Devices.Interfaces, vmiIface)
	t.Vmi.Spec.Networks = append(t.Vmi.Spec.Networks, vmiNetwork)

	t.Domain.Spec.Devices.HostDevices = append(t.Domain.Spec.Devices.HostDevices, api.HostDevice{
		Alias: api.NewUserDefinedAlias(netsriov.AliasPrefix + vmiNetwork.Name),
	})
}

// addGuestAgentInterfaces adds guest agent data.
// Guest agent data is collected and placed in the DomainStatus.
// During status update, this data is overriding the one from the domain spec and cache.
func (t *testSetup) addGuestAgentInterfaces(interfaces ...api.InterfaceStatus) {
	t.Domain.Status.Interfaces = append(t.Domain.Status.Interfaces, interfaces...)
}

func (t *testSetup) addFSCacheInterface(name string, podIPs ...string) error {
	c, err := t.podIfaceCache.IfaceEntry(name)
	if err != nil {
		return err
	}
	return c.Write(makePodCacheInterface(name, podIPs...))
}

func (t *testSetup) Cleanup() error {
	return t.cacheCreator.New("").Delete()
}

func makePodCacheInterface(networkName string, podIPs ...string) *cache.PodIfaceCacheData {
	return &cache.PodIfaceCacheData{
		Iface: &v1.Interface{
			Name: networkName,
		},
		PodIP:  podIPs[0],
		PodIPs: podIPs,
	}
}

func newDomainSpecIface(alias, mac string) api.Interface {
	return api.Interface{
		Alias: api.NewUserDefinedAlias(alias),
		MAC:   &api.MAC{MAC: mac},
	}
}

func newDomainStatusIface(IPs []string, mac, interfaceName string) api.InterfaceStatus {
	var ip string
	if len(IPs) > 0 {
		ip = IPs[0]
	}
	return api.InterfaceStatus{
		Ip:            ip,
		IPs:           IPs,
		Mac:           mac,
		InterfaceName: interfaceName,
	}
}

func newVMIStatusIface(name string, IPs []string, mac, ifaceName string, infoSource string, queueCount int32) v1.VirtualMachineInstanceNetworkInterface {
	var ip string
	if len(IPs) > 0 {
		ip = IPs[0]
	}
	return v1.VirtualMachineInstanceNetworkInterface{
		Name:          name,
		InterfaceName: ifaceName,
		IP:            ip,
		IPs:           IPs,
		MAC:           mac,
		InfoSource:    infoSource,
		QueueCount:    queueCount,
	}
}

func newVMISpecIfaceWithMasqueradeBinding(name string) v1.Interface {
	return v1.Interface{
		Name: name,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Masquerade: &v1.InterfaceMasquerade{},
		},
	}
}
func newVMISpecIfaceWithBridgeBinding(name string) v1.Interface {
	return v1.Interface{
		Name: name,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			Bridge: &v1.InterfaceBridge{},
		},
	}
}

func newVMISpecIfaceWithSRIOVBinding(name string) v1.Interface {
	return v1.Interface{
		Name: name,
		InterfaceBindingMethod: v1.InterfaceBindingMethod{
			SRIOV: &v1.InterfaceSRIOV{},
		},
	}
}

func newVMISpecPodNetwork(name string) v1.Network {
	return v1.Network{Name: name, NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}}}
}

func newVMISpecMultusNetwork(name string) v1.Network {
	return v1.Network{
		Name: name,
		NetworkSource: v1.NetworkSource{
			Multus: &v1.MultusNetwork{
				NetworkName: "test.network",
			}},
	}
}
