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

package network

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/virtio"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

type DomainConfigurator struct {
	domainAttachmentByInterfaceName map[string]string
	useLaunchSecuritySEV            bool
	useLaunchSecurityPV             bool
}

type option func(*DomainConfigurator)

func NewDomainConfigurator(options ...option) DomainConfigurator {
	var configurator DomainConfigurator

	for _, f := range options {
		f(&configurator)
	}

	return configurator
}

func (d DomainConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	var domainInterfaces []api.Interface

	nonAbsentIfaces := netvmispec.FilterInterfacesSpec(vmi.Spec.Domain.Devices.Interfaces, func(iface v1.Interface) bool {
		return iface.State != v1.InterfaceStateAbsent
	})
	nonAbsentNets := netvmispec.FilterNetworksByInterfaces(vmi.Spec.Networks, nonAbsentIfaces)

	networks := indexNetworksByName(nonAbsentNets)

	for i, iface := range nonAbsentIfaces {
		_, isExist := networks[iface.Name]
		if !isExist {
			return fmt.Errorf("failed to find network %s", iface.Name)
		}

		if (iface.Binding != nil && d.domainAttachmentByInterfaceName[iface.Name] != string(v1.Tap)) || iface.SRIOV != nil {
			continue
		}

		ifaceType := getInterfaceType(&nonAbsentIfaces[i])
		domainIface := api.Interface{
			Model: &api.Model{
				Type: translateModel(vmi.Spec.Domain.Devices.UseVirtioTransitional, ifaceType, vmi.Spec.Architecture),
			},
			Alias: api.NewUserDefinedAlias(iface.Name),
		}

		if queueCount := uint(calculateNetworkQueues(vmi, ifaceType)); queueCount != 0 {
			domainIface.Driver = &api.InterfaceDriver{Name: "vhost", Queues: &queueCount}
		}

		// Add a pciAddress if specified
		if iface.PciAddress != "" {
			addr, err := device.NewPciAddressField(iface.PciAddress)
			if err != nil {
				return fmt.Errorf("failed to configure interface %s: %v", iface.Name, err)
			}
			domainIface.Address = addr
		}

		if iface.ACPIIndex > 0 {
			domainIface.ACPI = &api.ACPI{Index: uint(iface.ACPIIndex)}
		}

		if d.domainAttachmentByInterfaceName[iface.Name] == string(v1.Tap) {
			// use "ethernet" interface type, since we're using pre-configured tap devices
			// https://libvirt.org/formatdomain.html#elementsNICSEthernet
			domainIface.Type = "ethernet"
			if iface.BootOrder != nil {
				domainIface.BootOrder = &api.BootOrder{Order: *iface.BootOrder}
			} else if arch.NewConverter(vmi.Spec.Architecture).IsROMTuningSupported() {
				domainIface.Rom = &api.Rom{Enabled: "no"}
			}
		}

		if d.useLaunchSecuritySEV || d.useLaunchSecurityPV {
			if arch.NewConverter(vmi.Spec.Architecture).IsROMTuningSupported() {
				// It's necessary to disable the iPXE option ROM as iPXE is not aware of SEV
				domainIface.Rom = &api.Rom{Enabled: "no"}
			}
			if ifaceType == v1.VirtIO {
				if domainIface.Driver != nil {
					domainIface.Driver.IOMMU = "on"
				} else {
					domainIface.Driver = &api.InterfaceDriver{Name: "vhost", IOMMU: "on"}
				}
			}
		}

		if iface.State == v1.InterfaceStateLinkDown {
			domainIface.LinkState = &api.LinkState{State: "down"}
		}
		domainInterfaces = append(domainInterfaces, domainIface)
	}

	domain.Spec.Devices.Interfaces = domainInterfaces
	return nil
}

func WithDomainAttachmentByInterfaceName(domainAttachmentByInterfaceName map[string]string) option {
	return func(d *DomainConfigurator) {
		d.domainAttachmentByInterfaceName = domainAttachmentByInterfaceName
	}
}

func WithUseLaunchSecuritySEV(useLaunchSecuritySEV bool) option {
	return func(d *DomainConfigurator) {
		d.useLaunchSecuritySEV = useLaunchSecuritySEV
	}
}

func WithUseLaunchSecurityPV(useLaunchSecurityPV bool) option {
	return func(d *DomainConfigurator) {
		d.useLaunchSecurityPV = useLaunchSecurityPV
	}
}

func getInterfaceType(iface *v1.Interface) string {
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

func calculateNetworkQueues(vmi *v1.VirtualMachineInstance, ifaceType string) uint32 {
	if ifaceType != v1.VirtIO {
		return 0
	}
	return NetworkQueuesCapacity(vmi)
}

func translateModel(useVirtioTransitional *bool, bus string, archString string) string {
	if bus == v1.VirtIO {
		return virtio.InterpretTransitionalModelType(useVirtioTransitional, archString)
	}
	return bus
}
