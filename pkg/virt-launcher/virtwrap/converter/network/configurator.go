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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/device"
)

type DomainConfigurator struct {
	domainAttachmentByInterfaceName map[string]string
	useLaunchSecuritySEV            bool
	useLaunchSecurityPV             bool
	isROMTuningSupported            bool
	virtioModel                     string
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

		domainIface, err := d.configureInterface(&nonAbsentIfaces[i], vmi)
		if err != nil {
			return err
		}

		domainInterfaces = append(domainInterfaces, domainIface)
	}

	domain.Spec.Devices.Interfaces = domainInterfaces
	return nil
}

func (d DomainConfigurator) configureInterface(iface *v1.Interface, vmi *v1.VirtualMachineInstance) (api.Interface, error) {
	var builderOptions []builderOption

	useLaunchSecurity := d.useLaunchSecuritySEV || d.useLaunchSecurityPV

	ifaceType := getInterfaceType(iface)
	modelType := ifaceType
	if ifaceType == v1.VirtIO {
		modelType = d.virtioModel

		builderOptions = append(builderOptions, withDriver(newVirtioDriver(vmi, useLaunchSecurity)))
	}

	if iface.PciAddress != "" {
		addr, err := device.NewPciAddressField(iface.PciAddress)
		if err != nil {
			return api.Interface{}, fmt.Errorf("failed to configure interface %s: %v", iface.Name, err)
		}
		builderOptions = append(builderOptions, withPCIAddress(addr))
	}

	domainIface := newDomainInterface(iface.Name, modelType, builderOptions...)

	if iface.ACPIIndex > 0 {
		domainIface.ACPI = &api.ACPI{Index: uint(iface.ACPIIndex)}
	}

	if d.domainAttachmentByInterfaceName[iface.Name] == string(v1.Tap) {
		// use "ethernet" interface type, since we're using pre-configured tap devices
		// https://libvirt.org/formatdomain.html#elementsNICSEthernet
		domainIface.Type = "ethernet"
		if iface.BootOrder != nil {
			domainIface.BootOrder = &api.BootOrder{Order: *iface.BootOrder}
		} else if d.isROMTuningSupported {
			domainIface.Rom = &api.Rom{Enabled: "no"}
		}
	}

	if useLaunchSecurity {
		if d.isROMTuningSupported {
			// It's necessary to disable the iPXE option ROM as iPXE is not aware of SEV
			domainIface.Rom = &api.Rom{Enabled: "no"}
		}
	}

	if iface.State == v1.InterfaceStateLinkDown {
		domainIface.LinkState = &api.LinkState{State: "down"}
	}
	return domainIface, nil
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

func WithROMTuningSupport(isROMTuningSupported bool) option {
	return func(d *DomainConfigurator) {
		d.isROMTuningSupported = isROMTuningSupported
	}
}

func WithVirtioModel(virtioModel string) option {
	return func(d *DomainConfigurator) {
		d.virtioModel = virtioModel
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

func newVirtioDriver(vmi *v1.VirtualMachineInstance, requiresIOMMU bool) *api.InterfaceDriver {
	var driver *api.InterfaceDriver
	queueCount := uint(NetworkQueuesCapacity(vmi))

	if queueCount > 0 || requiresIOMMU {
		driver = &api.InterfaceDriver{Name: "vhost"}
		if queueCount > 0 {
			driver.Queues = &queueCount
		}
		if requiresIOMMU {
			driver.IOMMU = "on"
		}
	}

	return driver
}
