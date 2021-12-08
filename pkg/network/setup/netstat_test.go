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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("netstat", func() {
	const (
		iface0 = "iface0"
		iface1 = "iface1"
	)

	var netStat *netsetup.NetStat
	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		netStat = netsetup.NewNetStat(&interfaceCacheFactoryStatusStub{})

		vmi = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{UID: "123"}}
	})

	It("run status with no domain", func() {
		Expect(netStat.UpdateStatus(vmi, nil)).To(Succeed())
	})

	Context("with volatile cache", func() {
		const (
			primaryNetworkName = "primary"
			primaryPodIPv4     = "1.1.1.1"
			primaryPodIPv6     = "fd10:244::8c4c"

			secondaryNetworkName = "secondary"
			secondaryPodIPv4     = "1.1.1.2"
			secondaryPodIPv6     = "fd10:244::8c4e"
		)

		BeforeEach(func() {
			vmi.Spec.Networks = []v1.Network{
				{
					Name:          primaryNetworkName,
					NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
				},
				{
					Name: secondaryNetworkName,
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{
							NetworkName: "test.network",
						},
					},
				},
			}

			podCacheInterface := makePodCacheInterface(primaryNetworkName, primaryPodIPv4, primaryPodIPv6)
			netStat.CachePodInterfaceVolatileData(vmi, primaryNetworkName, podCacheInterface)

			podCacheSecondaryInterface := makePodCacheInterface(secondaryNetworkName, secondaryPodIPv4, secondaryPodIPv6)
			netStat.CachePodInterfaceVolatileData(vmi, secondaryNetworkName, podCacheSecondaryInterface)
		})

		It("run status and expect two interfaces/networks to be reported (without guest-agent)", func() {
			netStat.UpdateStatus(vmi, &api.Domain{})

			Expect(vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, "", ""),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryPodIPv4, secondaryPodIPv6}, "", ""),
			}), "the pod IP/s should be reported in the status")

			Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, primaryNetworkName)).To(BeTrue())
			Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, secondaryNetworkName)).To(BeTrue())
		})

		It("run status and expect 2 interfaces to be reported based on guest-agent data", func() {
			// Guest data collected by the guest-agent
			const (
				primaryGaIPv4 = "2.2.2.1"
				primaryGaIPv6 = "fd20:244::8c4c"

				secondaryGaIPv4 = "2.2.2.2"
				secondaryGaIPv6 = "fd20:244::8c4e"

				primaryMAC   = "1C:CE:C0:01:BE:E7"
				secondaryMAC = "1C:CE:C0:01:BE:E9"
			)

			// Guest agent data is collected and placed in the DomainStatus.
			// During status update, this data is overriding the one from the domain spec and cache.
			domain := &api.Domain{
				Spec: api.DomainSpec{Devices: api.Devices{Interfaces: []api.Interface{
					newDomainSpecIface(primaryNetworkName, primaryMAC),
					newDomainSpecIface(secondaryNetworkName, secondaryMAC),
				}}},
				Status: api.DomainStatus{Interfaces: []api.InterfaceStatus{
					newDomainStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC),
					newDomainStatusIface(secondaryNetworkName, []string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC),
				}},
			}

			netStat.UpdateStatus(vmi, domain)

			Expect(vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC, ""),
				newVMIStatusIface(secondaryNetworkName, []string{secondaryGaIPv4, secondaryGaIPv6}, secondaryMAC, ""),
			}), "the guest-agent IP/s should be reported in the status")

			Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, primaryNetworkName)).To(BeTrue())
			Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, secondaryNetworkName)).To(BeTrue())
		})

		It("run status and expect an interfaces (with masquerade) to be reported based on pod & guest-agent data", func() {
			// Guest data collected by the guest-agent
			const (
				primaryGaIPv4 = "2.2.2.1"
				primaryGaIPv6 = "fd20:244::8c4c"

				primaryMAC = "1C:CE:C0:01:BE:E7"
			)

			// Guest agent data is collected and placed in the DomainStatus.
			// During status update, this data is overriding the one from the domain spec and cache.
			domain := &api.Domain{
				Spec: api.DomainSpec{Devices: api.Devices{Interfaces: []api.Interface{
					newDomainSpecIface(primaryNetworkName, primaryMAC),
				}}},
				Status: api.DomainStatus{Interfaces: []api.InterfaceStatus{
					newDomainStatusIface(primaryNetworkName, []string{primaryGaIPv4, primaryGaIPv6}, primaryMAC),
				}},
			}

			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{newVMISpecIfaceWithMasqueradeBinding(primaryNetworkName)}

			netStat.UpdateStatus(vmi, domain)

			Expect(vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
				newVMIStatusIface(primaryNetworkName, []string{primaryPodIPv4, primaryPodIPv6}, primaryMAC, ""),
			}), "the pod IP/s should be reported in the status")

			Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, primaryNetworkName)).To(BeTrue())
		})
	})

	It("runs teardown that clears volatile cache", func() {
		data := &cache.PodCacheInterface{}
		netStat.CachePodInterfaceVolatileData(vmi, iface0, data)
		netStat.CachePodInterfaceVolatileData(vmi, iface1, data)

		netStat.Teardown(vmi)

		Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, iface0)).To(BeFalse())
		Expect(netStat.PodInterfaceVolatileDataIsCached(vmi, iface1)).To(BeFalse())
	})

	It("should update existing interface status with MAC from the domain", func() {
		const (
			primaryNetworkName = "primary"

			origIPv4 = "1.1.1.1"
			origMAC  = "C0:01:BE:E7:15:G0:0D"

			newDomainMAC = "1C:CE:C0:01:BE:E7"
		)

		vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
			{
				IP:   origIPv4,
				IPs:  []string{origIPv4},
				MAC:  origMAC,
				Name: primaryNetworkName,
			},
		}

		domain := &api.Domain{
			Spec: api.DomainSpec{Devices: api.Devices{Interfaces: []api.Interface{
				newDomainSpecIface(primaryNetworkName, newDomainMAC),
			}}},
		}

		netStat.UpdateStatus(vmi, domain)

		Expect(vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(primaryNetworkName, []string{origIPv4}, newDomainMAC, ""),
		}), "the pod IP/s should be reported in the status")
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

		vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
			{
				IP:   origIPv4,
				IPs:  []string{origIPv4, origIPv6},
				MAC:  origMAC,
				Name: primaryNetworkName,
			},
		}

		domain := &api.Domain{
			Spec: api.DomainSpec{Devices: api.Devices{Interfaces: []api.Interface{
				newDomainSpecIface(primaryNetworkName, origMAC),
			}}},
			Status: api.DomainStatus{Interfaces: []api.InterfaceStatus{
				newDomainStatusIface(primaryNetworkName, []string{newGaIPv4, newGaIPv6}, origMAC),
			}},
		}

		netStat.UpdateStatus(vmi, domain)

		Expect(vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(primaryNetworkName, []string{newGaIPv4, newGaIPv6}, origMAC, ""),
		}), "the pod IP/s should be reported in the status")
	})

	It("should add a new interface based on the domain spec", func() {
		const (
			existingNetworkName = "primary"

			existingIPv4 = "1.1.1.1"
			existingIPv6 = "fd10:1111::1111"
			existingMAC  = "C0:01:BE:E7:15:G0:0D"

			newNetworkName = "secondary"
			newDomainMAC   = "22:22:22:22:22:22"
		)

		vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
			{
				IP:   existingIPv4,
				IPs:  []string{existingIPv4, existingIPv6},
				MAC:  existingMAC,
				Name: existingNetworkName,
			},
		}

		domain := &api.Domain{
			Spec: api.DomainSpec{Devices: api.Devices{Interfaces: []api.Interface{
				newDomainSpecIface(existingNetworkName, existingMAC),
				newDomainSpecIface(newNetworkName, newDomainMAC),
			}}},
		}

		netStat.UpdateStatus(vmi, domain)

		Expect(vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(existingNetworkName, []string{existingIPv4, existingIPv6}, existingMAC, ""),
			newVMIStatusIface(newNetworkName, nil, newDomainMAC, ""),
		}), "the new interface should be reported in the status")
	})

	It("should replace a non-named interface with new data (with name) from the domain", func() {
		const (
			networkName  = "primary"
			existingIPv4 = "1.1.1.1"
			existingMAC  = "C0:01:BE:E7:15:G0:0D"
			newDomainMAC = "22:22:22:22:22:22"
		)

		vmi.Spec.Networks = []v1.Network{
			{
				Name:          "other_name",
				NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{}},
			},
			{
				Name:          networkName,
				NetworkSource: v1.NetworkSource{Pod: &v1.PodNetwork{}},
			},
		}

		vmi.Status.Interfaces = []v1.VirtualMachineInstanceNetworkInterface{
			{IP: existingIPv4, MAC: existingMAC},
		}

		domain := &api.Domain{
			Spec: api.DomainSpec{Devices: api.Devices{Interfaces: []api.Interface{
				newDomainSpecIface(networkName, newDomainMAC),
			}}},
		}

		netStat.UpdateStatus(vmi, domain)

		Expect(vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(networkName, nil, newDomainMAC, ""),
		}), "the new interface should be reported in the status")
	})

	It("should report SR-IOV interface with MAC and network name, based on VMI spec and guest-agent data", func() {
		const (
			networkName    = "sriov-network"
			NADName        = "sriov-nad"
			ifaceMAC       = "C0:01:BE:E7:15:G0:0D"
			guestIfaceName = "eth1"
		)

		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
			{
				Name:                   networkName,
				InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &v1.InterfaceSRIOV{}},
				MacAddress:             ifaceMAC,
			},
		}

		vmi.Spec.Networks = []v1.Network{
			{Name: networkName, NetworkSource: v1.NetworkSource{Multus: &v1.MultusNetwork{NetworkName: NADName}}},
		}

		domain := &api.Domain{
			Status: api.DomainStatus{Interfaces: []api.InterfaceStatus{
				{Mac: ifaceMAC, InterfaceName: guestIfaceName},
			}},
		}

		netStat.UpdateStatus(vmi, domain)

		Expect(vmi.Status.Interfaces).To(Equal([]v1.VirtualMachineInstanceNetworkInterface{
			newVMIStatusIface(networkName, nil, ifaceMAC, guestIfaceName),
		}), "the SR-IOV interface should be reported in the status, associated to the network")
	})
})

type interfaceCacheFactoryStatusStub struct {
	podInterfaceCacheStore podInterfaceCacheStoreStatusStub
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

type podInterfaceCacheStoreStatusStub struct{ failRemove bool }

func (p podInterfaceCacheStoreStatusStub) Read(iface string) (*cache.PodCacheInterface, error) {
	return &cache.PodCacheInterface{Iface: &v1.Interface{Name: "net-name"}}, nil
}

func (p podInterfaceCacheStoreStatusStub) Write(iface string, cacheInterface *cache.PodCacheInterface) error {
	return nil
}

func (p podInterfaceCacheStoreStatusStub) Remove() error {
	if p.failRemove {
		return fmt.Errorf("remove failed")
	}
	return nil
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

func newDomainStatusIface(name string, IPs []string, mac string) api.InterfaceStatus {
	var ip string
	if len(IPs) > 0 {
		ip = IPs[0]
	}
	return api.InterfaceStatus{
		Name: name,
		Ip:   ip,
		IPs:  IPs,
		Mac:  mac,
	}
}

func newVMIStatusIface(name string, IPs []string, mac, ifaceName string) v1.VirtualMachineInstanceNetworkInterface {
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
