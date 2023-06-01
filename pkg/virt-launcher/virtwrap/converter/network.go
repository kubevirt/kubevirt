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
	"net"
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
	isVirtioNetProhibited, err := c.IsVirtIONetProhibited()
	if err != nil {
		return nil, err
	}

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

		if iface.SRIOV != nil {
			continue
		}

		ifaceType := GetInterfaceType(&nonAbsentIfaces[i])
		domainIface := api.Interface{
			Model: &api.Model{
				Type: translateModel(vmi.Spec.Domain.Devices.UseVirtioTransitional, ifaceType),
			},
			Alias: api.NewUserDefinedAlias(iface.Name),
		}

		// if AllowEmulation unset and at least one NIC model is virtio,
		// /dev/vhost-net must be present as we should have asked for it.
		if ifaceType == v1.VirtIO && isVirtioNetProhibited {
			return nil, fmt.Errorf("In-kernel virtio-net device emulation '/dev/vhost-net' not present")
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
		} else if iface.Slirp != nil {
			domainIface.Type = "user"

			// Create network interface
			initializeQEMUCmdAndQEMUArg(domain)

			// TODO: (seba) Need to change this if multiple interface can be connected to the same network
			// append the ports from all the interfaces connected to the same network
			err := createSlirpNetwork(iface, *net, domain)
			if err != nil {
				return nil, err
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
		} else if iface.Passt != nil {
			domain.Spec.Devices.Emulator = "/usr/bin/qrap"
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
	if iface.Slirp != nil {
		// Slirp configuration works only with e1000 or rtl8139
		if iface.Model != "e1000" && iface.Model != "rtl8139" {
			log.Log.Infof("The network interface type of %s was changed to e1000 due to unsupported interface type by qemu slirp network", iface.Name)
			return "e1000"
		}
		return iface.Model
	}
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

func createSlirpNetwork(iface v1.Interface, network v1.Network, domain *api.Domain) error {
	qemuArg := api.Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}

	err := configVMCIDR(&qemuArg, network)
	if err != nil {
		return err
	}

	err = configDNSSearchName(&qemuArg)
	if err != nil {
		return err
	}

	err = configPortForward(&qemuArg, iface)
	if err != nil {
		return err
	}

	domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-netdev"})
	domain.Spec.QEMUCmd.QEMUArg = append(domain.Spec.QEMUCmd.QEMUArg, qemuArg)

	return nil
}

func CalculateNetworkQueues(vmi *v1.VirtualMachineInstance, ifaceType string) uint32 {
	if !isTrue(vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue) || ifaceType != v1.VirtIO {
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

func configPortForward(qemuArg *api.Arg, iface v1.Interface) error {
	if iface.Ports == nil {
		return nil
	}

	// Can't be duplicated ports forward or the qemu process will crash
	configuredPorts := make(map[string]struct{}, 0)
	for _, forwardPort := range iface.Ports {

		if forwardPort.Port == 0 {
			return fmt.Errorf("Port must be configured")
		}

		if forwardPort.Protocol == "" {
			forwardPort.Protocol = api.DefaultProtocol
		}

		portConfig := fmt.Sprintf("%s-%d", forwardPort.Protocol, forwardPort.Port)
		if _, ok := configuredPorts[portConfig]; !ok {
			qemuArg.Value += fmt.Sprintf(",hostfwd=%s::%d-:%d", strings.ToLower(forwardPort.Protocol), forwardPort.Port, forwardPort.Port)
			configuredPorts[portConfig] = struct{}{}
		}
	}

	return nil
}

func configVMCIDR(qemuArg *api.Arg, network v1.Network) error {
	vmNetworkCIDR := ""
	if network.Pod.VMNetworkCIDR != "" {
		_, _, err := net.ParseCIDR(network.Pod.VMNetworkCIDR)
		if err != nil {
			return fmt.Errorf("Failed parsing CIDR %s", network.Pod.VMNetworkCIDR)
		}
		vmNetworkCIDR = network.Pod.VMNetworkCIDR
	} else {
		vmNetworkCIDR = api.DefaultVMCIDR
	}

	// Insert configuration to qemu commandline
	qemuArg.Value += fmt.Sprintf(",net=%s", vmNetworkCIDR)

	return nil
}

func configDNSSearchName(qemuArg *api.Arg) error {
	_, dnsDoms, err := GetResolvConfDetailsFromPod()
	if err != nil {
		return err
	}

	for _, dom := range dnsDoms {
		qemuArg.Value += fmt.Sprintf(",dnssearch=%s", dom)
	}
	return nil
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
