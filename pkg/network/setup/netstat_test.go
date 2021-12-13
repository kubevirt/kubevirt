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
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("netstat", func() {
	var setup testSetup

	BeforeEach(func() {
		setup = newTestSetup()
	})

	It("run status with no domain", func() {
		Expect(setup.NetStat.UpdateStatus(setup.Vmi, nil)).To(Succeed())
	})

	Context("with volatile cache", func() {
		const (
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
			primaryPodIPv6     = "fd10:244::8c4c"
			primaryGaIPv4      = "2.2.2.1"
			primaryGaIPv6      = "fd20:244::8c4c"
			primaryMAC         = "1C:CE:C0:01:BE:E7"

			secondaryNetworkName = "secondary"
			secondaryPodIPv4     = "1.1.1.2"
			secondaryPodIPv6     = "fd10:244::8c4e"
			secondaryGaIPv4      = "2.2.2.2"
			secondaryGaIPv6      = "fd20:244::8c4e"
			secondaryMAC         = "1C:CE:C0:01:BE:E9"
		)

		BeforeEach(func() {
			setup = newTestSetupWithVolatileCache()
		})

		It("run status and expect two interfaces/networks to be reported (without guest-agent)", func() {
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, ""),
				primaryPodIPv4, primaryPodIPv6,
			)
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(secondaryNetworkName),
				newVMISpecMultusNetwork(secondaryNetworkName),
				newDomainSpecIface(secondaryNetworkName, ""),
				secondaryPodIPv4, secondaryPodIPv6,
			)

			setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, "", "", netvmispec.InfoSourceDomain),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryPodIPv4, secondaryPodIPv6}, "", "", netvmispec.InfoSourceDomain),
			}), "the pod IP/s should be reported in the status")

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeTrue())
			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, secondaryNetworkName)).To(BeTrue())
		})

		It("run status and expect 2 interfaces to be reported based on guest-agent data", func() {
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, primaryMAC),
				primaryPodIPv4, primaryPodIPv6,
			)
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(secondaryNetworkName),
				newVMISpecMultusNetwork(secondaryNetworkName),
				newDomainSpecIface(secondaryNetworkName, secondaryMAC),
				secondaryPodIPv4, secondaryPodIPv6,
			)

			setup.addGuestAgentInterfaces(
				newDomainStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, "", netvmispec.InfoSourceDomainAndGA),
				newDomainStatusIface(secondaryNetworkName, []string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC, "", netvmispec.InfoSourceDomainAndGA),
			)

			setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, "", netvmispec.InfoSourceDomainAndGA),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC, "", netvmispec.InfoSourceDomainAndGA),
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

			setup.addNetworkInterface(
				newVMISpecIfaceWithMasqueradeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, primaryMAC),
				primaryPodIPv4, primaryPodIPv6,
			)
			setup.addGuestAgentInterfaces(
				newDomainStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, "eth0", netvmispec.InfoSourceDomainAndGA),
			)

			setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, primaryMAC, "eth0", netvmispec.InfoSourceDomainAndGA),
			}), "the pod IP/s should be reported in the status")

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeTrue())
		})

		It("should update existing interface status with MAC from the domain", func() {
			const (
				origMAC      = "C0:01:BE:E7:15:G0:0D"
				newDomainMAC = "1C:CE:C0:01:BE:E7"
			)

			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, newDomainMAC),
				primaryPodIPv4, primaryPodIPv6,
			)

			setup.Vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
				{
					IP:   primaryPodIPv4,
					IPs:  []string{primaryPodIPv4, primaryPodIPv6},
					MAC:  origMAC,
					Name: primaryNetworkName,
				},
			}

			setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)

			Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, newDomainMAC, "", netvmispec.InfoSourceDomain),
			}), "the pod IP/s should be reported in the status")
		})

		It("runs teardown that clears volatile cache", func() {
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
				newVMISpecPodNetwork(primaryNetworkName),
				newDomainSpecIface(primaryNetworkName, ""),
				primaryPodIPv4, primaryPodIPv6,
			)
			setup.addNetworkInterface(
				newVMISpecIfaceWithBridgeBinding(secondaryNetworkName),
				newVMISpecMultusNetwork(secondaryNetworkName),
				newDomainSpecIface(secondaryNetworkName, ""),
				secondaryPodIPv4, secondaryPodIPv6,
			)

			setup.NetStat.Teardown(setup.Vmi)

			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, primaryNetworkName)).To(BeFalse())
			Expect(setup.NetStat.PodInterfaceVolatileDataIsCached(setup.Vmi, secondaryNetworkName)).To(BeFalse())
		})
	})

	It("should update existing interface status with IP from the guest-agent", func() {
		const (
			primaryNetworkName = "primary"

			origIPv4 = "1.1.1.1"
			origIPv6 = "fd10:1111::1111"
			origMAC  = "C0:01:BE:E7:15:G0:0D"

			newGaIPv4 = "2.2.2.2"
			newGaIPv6 = "fd20:2222::2222"
		)

		setup.addNetworkInterface(
			newVMISpecIfaceWithBridgeBinding(primaryNetworkName),
			newVMISpecPodNetwork(primaryNetworkName),
			newDomainSpecIface(primaryNetworkName, origMAC),
			origIPv4, origIPv6,
		)
		setup.Vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
			{
				IP:   origIPv4,
				IPs:  []string{origIPv4, origIPv6},
				MAC:  origMAC,
				Name: primaryNetworkName,
			},
		}

		setup.addGuestAgentInterfaces(
			newDomainStatusIface(primaryNetworkName, []string{newGaIPv4, newGaIPv6}, origMAC, "", netvmispec.InfoSourceDomainAndGA),
		)

		setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)

		Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(primaryNetworkName, []string{newGaIPv4, newGaIPv6}, origMAC, "", netvmispec.InfoSourceDomainAndGA),
		}), "the pod IP/s should be reported in the status")
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
			newDomainStatusIface("", nil, ifaceMAC, guestIfaceName, netvmispec.InfoSourceGuestAgent),
		)

		setup.NetStat.UpdateStatus(setup.Vmi, setup.Domain)

		Expect(setup.Vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(networkName, nil, ifaceMAC, guestIfaceName, netvmispec.InfoSourceGuestAgent),
		}), "the SR-IOV interface should be reported in the status, associated to the network")
	})
})

type interfaceCacheFactoryStatusStub struct {
	podInterfaceCacheStore podInterfaceCacheStoreStatusStub
}

func newInterfaceCacheFactoryStub() *interfaceCacheFactoryStatusStub {
	return &interfaceCacheFactoryStatusStub{
		podInterfaceCacheStore: podInterfaceCacheStoreStatusStub{
			data: map[string]*cache.PodCacheInterface{},
		},
	}
}

func (i interfaceCacheFactoryStatusStub) CacheForVMI(vmi *v1.VirtualMachineInstance) cache.PodInterfaceCacheStore {
	return i.podInterfaceCacheStore
}
func (i interfaceCacheFactoryStatusStub) CacheDomainInterfaceForPID(pid string) cache.DomainInterfaceStore {
	return nil
}
func (i interfaceCacheFactoryStatusStub) CacheDHCPConfigForPid(pid string) cache.DHCPConfigStore {
	return nil
}

type podInterfaceCacheStoreStatusStub struct {
	data       map[string]*cache.PodCacheInterface
	failRemove bool
}

func (p podInterfaceCacheStoreStatusStub) Read(iface string) (*cache.PodCacheInterface, error) {
	if d, exists := p.data[iface]; exists {
		return &cache.PodCacheInterface{Iface: d.Iface}, nil
	}
	return &cache.PodCacheInterface{}, nil
}

func (p podInterfaceCacheStoreStatusStub) Write(iface string, cacheInterface *cache.PodCacheInterface) error {
	p.data[iface] = cacheInterface
	return nil
}

func (p podInterfaceCacheStoreStatusStub) Remove() error {
	if p.failRemove {
		return fmt.Errorf("remove failed")
	}
	return nil
}

type testSetup struct {
	Vmi     *v1.VirtualMachineInstance
	Domain  *api.Domain
	NetStat *netsetup.NetStat

	ifaceFSCacheFactory *interfaceCacheFactoryStatusStub

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
	vmi := &v1.VirtualMachineInstance{}
	vmi.UID = "123"
	ifaceFSCacheFactory := newInterfaceCacheFactoryStub()
	return testSetup{
		Vmi:                 vmi,
		Domain:              &api.Domain{},
		NetStat:             netsetup.NewNetStat(ifaceFSCacheFactory),
		ifaceFSCacheFactory: ifaceFSCacheFactory,
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
func (t *testSetup) addNetworkInterface(vmiIface v1.Interface, vmiNetwork v1.Network, domainIface api.Interface, podIPs ...string) {
	if !(vmiIface.Name == vmiNetwork.Name && vmiIface.Name == domainIface.Alias.GetName()) {
		panic("network name must be the same")
	}
	t.Vmi.Spec.Domain.Devices.Interfaces = append(t.Vmi.Spec.Domain.Devices.Interfaces, vmiIface)
	t.Vmi.Spec.Networks = append(t.Vmi.Spec.Networks, vmiNetwork)

	t.Domain.Spec.Devices.Interfaces = append(t.Domain.Spec.Devices.Interfaces, domainIface)

	t.addFSCacheInterface(vmiNetwork.Name, podIPs...)

	if t.volatileCache {
		podCacheInterface := makePodCacheInterface(vmiNetwork.Name, podIPs...)
		t.NetStat.CachePodInterfaceVolatileData(t.Vmi, vmiNetwork.Name, podCacheInterface)
	}
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
}

// addGuestAgentInterfaces adds guest agent data.
// Guest agent data is collected and placed in the DomainStatus.
// During status update, this data is overriding the one from the domain spec and cache.
func (t *testSetup) addGuestAgentInterfaces(interfaces ...api.InterfaceStatus) {
	t.Domain.Status.Interfaces = append(t.Domain.Status.Interfaces, interfaces...)
}

func (t *testSetup) addFSCacheInterface(name string, podIPs ...string) {
	t.ifaceFSCacheFactory.CacheForVMI(nil).Write(name, makePodCacheInterface(name, podIPs...))
}

func makePodCacheInterface(networkName string, podIPs ...string) *cache.PodCacheInterface {
	return &cache.PodCacheInterface{
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

func newDomainStatusIface(name string, IPs []string, mac, interfaceName string, infoSource string) api.InterfaceStatus {
	var ip string
	if len(IPs) > 0 {
		ip = IPs[0]
	}
	return api.InterfaceStatus{
		Name:          name,
		Ip:            ip,
		IPs:           IPs,
		Mac:           mac,
		InterfaceName: interfaceName,
		InfoSource:    infoSource,
	}
}

func newVMIStatusIface(name string, IPs []string, mac, ifaceName string, infoSource string) v1.VirtualMachineInstanceNetworkInterface {
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
