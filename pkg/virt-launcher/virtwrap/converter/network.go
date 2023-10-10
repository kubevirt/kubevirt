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

package converter

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"

	"kubevirt.io/kubevirt/pkg/network/dns"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

func CreateDomainInterfaces(vmi *v1.VirtualMachineInstance, domain *api.Domain, c *ConverterContext) ([]api.Interface, error) {
	var domainInterfaces []api.Interface

	nonAbsentIfaces := netvmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	nonAbsentNets := netvmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, nonAbsentIfaces)

	networks := indexNetworksByName(nonAbsentNets)

	for i, iface := range nonAbsentIfaces {
		net, isExist := networks[iface.Name]
		if !isExist {
			return nil, fmt.Errorf("failed to find network %s", iface.Name)
		}

		if iface.Binding != nil || iface.SRIOV != nil || iface.Slirp != nil {
			continue
		}

		ifaceType := GetInterfaceType(&nonAbsentIfaces[i])
		domainIface := api.Interface{
			Model: &api.Model{
				Type: translateModel(vmi.Spec.Domain.Devices.UseVirtioTransitional, ifaceType),
			},
			Alias: api.NewUserDefinedAlias(iface.Name),
		}

		if queueCount := uint(CalculateNetworkQueues(vmi, ifaceType)); queueCount != 0 {
			domainIface.Driver = &api.InterfaceDriver{Name: "vhost", Queues: &queueCount}
		}

		// Add a pciAddress if specified
		if iface.PciAddress != "" {
			addr, err := device.NewPciAddressField(iface.PciAddress)
			if err != nil {
				return nil, fmt.Errorf("failed to configure interface %s: %v", iface.Name, err)
			}
			domainIface.Address = addr
		}

		if iface.ACPIIndex > 0 {
			domainIface.ACPI = &api.ACPI{Index: uint(iface.ACPIIndex)}
		}

		if iface.Bridge != nil || iface.Masquerade != nil {
			// TODO:(ihar) consider abstracting interface type conversion /
			// detection into drivers

			// use "ethernet" interface type, since we're using pre-configured tap devices
			// https://libvirt.org/formatdomain.html#elementsNICSEthernet
			domainIface.Type = "ethernet"
			if iface.BootOrder != nil {
				domainIface.BootOrder = &api.BootOrder{Order: *iface.BootOrder}
			} else {
				domainIface.Rom = &api.Rom{Enabled: "no"}
			}
		} else if iface.Macvtap != nil {
			if net.Multus == nil {
				return nil, fmt.Errorf("macvtap interface %s requires Multus meta-cni", iface.Name)
			}

			domainIface.Type = "ethernet"
			if iface.BootOrder != nil {
				domainIface.BootOrder = &api.BootOrder{Order: *iface.BootOrder}
			} else {
				domainIface.Rom = &api.Rom{Enabled: "no"}
			}
		}

		if c.UseLaunchSecurity {
			// It's necessary to disable the iPXE option ROM as iPXE is not aware of SEV
			domainIface.Rom = &api.Rom{Enabled: "no"}
			if ifaceType == v1.VirtIO {
				if domainIface.Driver != nil {
					domainIface.Driver.IOMMU = "on"
				} else {
					domainIface.Driver = &api.InterfaceDriver{Name: "vhost", IOMMU: "on"}
				}
			}
		}
		domainInterfaces = append(domainInterfaces, domainIface)
	}

	return domainInterfaces, nil
}

func GetInterfaceType(iface *v1.Interface) string {
	if iface.Model != "" {
		return iface.Model
	}
	return v1.VirtIO
}

func indexNetworksByName(networks []v1.Network) map[string]*v1.Network {
	netsByName := map[string]*v1.Network{}
	for _, network := range networks {
		netsByName[network.Name] = network.DeepCopy()
	}
	return netsByName
}

func CalculateNetworkQueues(vmi *v1.VirtualMachineInstance, ifaceType string) uint32 {
	if ifaceType != v1.VirtIO {
		return 0
	}
	return NetworkQueuesCapacity(vmi)
}

func NetworkQueuesCapacity(vmi *v1.VirtualMachineInstance) uint32 {
	if !isTrue(vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue) {
		return 0
	}

	cpuTopology := vcpu.GetCPUTopology(vmi)
	queueNumber := vcpu.CalculateRequestedVCPUs(cpuTopology)

	if queueNumber > multiQueueMaxQueues {
		log.Log.V(3).Infof("Capped the number of queues to be the current maximum of tap device queues: %d", multiQueueMaxQueues)
		queueNumber = multiQueueMaxQueues
	}
	return queueNumber
}

func isTrue(networkInterfaceMultiQueue *bool) bool {
	return (networkInterfaceMultiQueue != nil) && (*networkInterfaceMultiQueue)
}

// returns nameservers [][]byte, searchdomains []string, error
func GetResolvConfDetailsFromPod() ([][]byte, []string, error) {
	// #nosec No risk for path injection. resolvConf is static "/etc/resolve.conf"
	b, err := os.ReadFile(resolvConf)
	if err != nil {
		return nil, nil, err
	}

	nameservers, err := dns.ParseNameservers(string(b))
	if err != nil {
		return nil, nil, err
	}

	searchDomains, err := dns.ParseSearchDomains(string(b))
	if err != nil {
		return nil, nil, err
	}

	log.Log.Reason(err).Infof("Found nameservers in %s: %s", resolvConf, bytes.Join(nameservers, []byte{' '}))
	log.Log.Reason(err).Infof("Found search domains in %s: %s", resolvConf, strings.Join(searchDomains, " "))

	return nameservers, searchDomains, err
}

func translateModel(useVirtioTransitional *bool, bus string) string {
	if bus == v1.VirtIO {
		return InterpretTransitionalModelType(useVirtioTransitional)
	}
	return bus
}
