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

const (
	//nolint:gosec //linter is confusing passt for password
	passtLogFilePath  = "/var/run/kubevirt/passt.log"
	passtBackendPasst = "passt"
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

	if iface.ACPIIndex > 0 {
		builderOptions = append(builderOptions, withACPIIndex(uint(iface.ACPIIndex)))
	}

	if iface.MacAddress != "" {
		builderOptions = append(builderOptions, withMACAddress(iface.MacAddress))
	}

	switch {
	case d.domainAttachmentByInterfaceName[iface.Name] == string(v1.Tap):
		builderOptions = append(builderOptions, d.tapBindingOptions(iface, useLaunchSecurity)...)

	case iface.PasstBinding != nil:
		passtOpts, err := d.passtBindingOptions(iface, vmi)
		if err != nil {
			return api.Interface{}, err
		}
		builderOptions = append(builderOptions, passtOpts...)

	default:
		return api.Interface{}, fmt.Errorf("invalid configuration for interface %s", iface.Name)
	}

	return newDomainInterface(iface.Name, modelType, builderOptions...), nil
}

func (d DomainConfigurator) tapBindingOptions(iface *v1.Interface, useLaunchSecurity bool) []builderOption {
	// use "ethernet" interface type, since we're using pre-configured tap devices
	// https://libvirt.org/formatdomain.html#elementsNICSEthernet
	opts := []builderOption{withIfaceType("ethernet")}

	if iface.BootOrder != nil {
		opts = append(opts, withBootOrder(*iface.BootOrder))
	}

	if d.isROMTuningSupported && (iface.BootOrder == nil || useLaunchSecurity) {
		opts = append(opts, withROMDisabled())
	}

	if iface.State == v1.InterfaceStateLinkDown {
		opts = append(opts, withLinkStateDown())
	}

	return opts
}

func (d DomainConfigurator) passtBindingOptions(iface *v1.Interface, vmi *v1.VirtualMachineInstance) ([]builderOption, error) {
	ifaceStatus := netvmispec.LookupInterfaceStatusByName(vmi.Status.Interfaces, iface.Name)
	if ifaceStatus == nil || ifaceStatus.PodInterfaceName == "" {
		return nil, fmt.Errorf("pod interface name not found in vmi %s status, for interface %s",
			vmi.Name, iface.Name)
	}
	return []builderOption{
		withIfaceType("vhostuser"),
		withSource(api.InterfaceSource{Device: ifaceStatus.PodInterfaceName}),
		withBackend(api.InterfaceBackend{Type: passtBackendPasst, LogFile: passtLogFilePath}),
		withPortForward(generatePasstPortForward(iface, vmi)),
	}, nil
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
