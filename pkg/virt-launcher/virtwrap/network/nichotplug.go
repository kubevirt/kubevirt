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
	"encoding/xml"
	"fmt"
	"strings"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

type VmConfigurator interface {
	SetupPodNetworkPhase2(domain *api.Domain, networksToPlug []v1.Network) error
}

type virtIOInterfaceManager struct {
	dom          domainClient
	configurator VmConfigurator
}

const (
	LibvirtInterfaceLinkStateDown         = "down"
	affectDeviceLiveAndConfigLibvirtFlags = libvirt.DOMAIN_DEVICE_MODIFY_LIVE | libvirt.DOMAIN_DEVICE_MODIFY_CONFIG
)

func NewVirtIOInterfaceManager(
	libvirtClient domainClient,
	configurator VmConfigurator,
) *virtIOInterfaceManager {
	return &virtIOInterfaceManager{
		dom:          libvirtClient,
		configurator: configurator,
	}
}

func (vim *virtIOInterfaceManager) HotplugVirtioInterface(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain, updatedDomain *api.Domain) error {
	for _, network := range NetworksToHotplugWhoseInterfacesAreNotInTheDomain(vmi, indexedDomainInterfaces(currentDomain)) {
		log.Log.Infof("will hot plug %s", network.Name)

		if err := vim.configurator.SetupPodNetworkPhase2(updatedDomain, []v1.Network{network}); err != nil {
			return err
		}

		relevantIface := lookupDomainInterfaceByName(updatedDomain.Spec.Devices.Interfaces, network.Name)
		if relevantIface == nil {
			return fmt.Errorf("could not retrieve the api.Interface object from the dummy domain")
		}

		ifaceMAC := ""
		if relevantIface.MAC != nil {
			ifaceMAC = relevantIface.MAC.MAC
		}
		log.Log.Infof("will hot plug %q with MAC %q", network.Name, ifaceMAC)
		ifaceXML, err := xml.Marshal(relevantIface)
		if err != nil {
			return err
		}

		if err := vim.dom.AttachDeviceFlags(strings.ToLower(string(ifaceXML)), affectDeviceLiveAndConfigLibvirtFlags); err != nil {
			log.Log.Reason(err).Errorf("libvirt failed to attach interface %s: %v", network.Name, err)
			return err
		}
	}
	return nil
}

func (vim *virtIOInterfaceManager) UpdateDomainLinkState(currentDomain, desiredDomain *api.Domain) error {

	currentDomainIfacesByAlias := indexedDomainInterfaces(currentDomain)
	for _, desiredIface := range desiredDomain.Spec.Devices.Interfaces {
		curIface, ok := currentDomainIfacesByAlias[desiredIface.Alias.GetName()]
		if !ok {
			continue
		}

		if !isLinkStateEqual(curIface, desiredIface) {
			curIface.LinkState = desiredIface.LinkState
			if err := vim.updateIfaceInDomain(&curIface); err != nil {
				return err
			}
		}

	}
	return nil
}

func (vim *virtIOInterfaceManager) updateIfaceInDomain(domIfaceToUpdate *api.Interface) error {
	log.Log.Infof("preparing to update link state to interface %q", domIfaceToUpdate.Alias.GetName())
	ifaceXML, err := xml.Marshal(domIfaceToUpdate)
	if err != nil {
		return err
	}

	if err = vim.dom.UpdateDeviceFlags(strings.ToLower(string(ifaceXML)), affectDeviceLiveAndConfigLibvirtFlags); err != nil {
		log.Log.Reason(err).Errorf("libvirt failed to set link state to interface %s , %v", domIfaceToUpdate.Alias.GetName(), err)
		return err
	}
	return nil
}

func (vim *virtIOInterfaceManager) hotUnplugVirtioInterface(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain) error {
	for _, domainIface := range InterfacesToHotUnplug(vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, currentDomain.Spec.Devices.Interfaces) {
		log.Log.Infof("preparing to hot-unplug %s", domainIface.Alias.GetName())

		ifaceXML, err := xml.Marshal(domainIface)
		if err != nil {
			return err
		}

		if derr := vim.dom.DetachDeviceFlags(strings.ToLower(string(ifaceXML)), affectDeviceLiveAndConfigLibvirtFlags); derr != nil {
			log.Log.Reason(derr).Errorf("libvirt failed to detach interface %s: %v", domainIface.Alias.GetName(), derr)
			return derr
		}
	}
	return nil
}

func InterfacesToHotUnplug(vmiSpecInterfaces []v1.Interface, vmiSpecNets []v1.Network, domainSpecInterfaces []api.Interface) []api.Interface {
	ifaces2remove := netvmispec.FilterInterfacesSpec(vmiSpecInterfaces, func(iface v1.Interface) bool {
		return iface.State == v1.InterfaceStateAbsent
	})

	networksByName := netvmispec.IndexNetworkSpecByName(vmiSpecNets)
	var domainIfacesToRemove []api.Interface
	for _, vmiIface := range ifaces2remove {
		if domainIface := lookupDomainInterfaceByName(domainSpecInterfaces, vmiIface.Name); domainIface != nil {
			if hasDeviceWithHashedTapName(domainIface.Target, vmiIface, networksByName[vmiIface.Name]) {
				domainIfacesToRemove = append(domainIfacesToRemove, *domainIface)
			}
		}
	}
	return domainIfacesToRemove
}

func hasDeviceWithHashedTapName(target *api.InterfaceTarget, vmiIface v1.Interface, vmiNet v1.Network) bool {
	return target != nil &&
		target.Device == virtnetlink.GenerateTapDeviceName(namescheme.GenerateHashedInterfaceName(vmiIface.Name), vmiNet)
}

func lookupDomainInterfaceByName(domainIfaces []api.Interface, networkName string) *api.Interface {
	for _, iface := range domainIfaces {
		if iface.Alias.GetName() == networkName {
			return &iface
		}
	}
	return nil
}

func NetworksToHotplugWhoseInterfacesAreNotInTheDomain(vmi *v1.VirtualMachineInstance, indexedDomainIfaces map[string]api.Interface) []v1.Network {
	var networksToHotplug []v1.Network
	interfacesToHoplug := netvmispec.IndexInterfaceStatusByName(
		vmi.Status.Interfaces,
		func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool {
			_, exists := indexedDomainIfaces[ifaceStatus.Name]
			vmiSpecIface := netvmispec.LookupInterfaceByName(vmi.Spec.Domain.Devices.Interfaces, ifaceStatus.Name)

			return netvmispec.ContainsInfoSource(
				ifaceStatus.InfoSource, netvmispec.InfoSourceMultusStatus,
			) && !exists && vmiSpecIface.State != v1.InterfaceStateAbsent && vmiSpecIface.SRIOV == nil
		},
	)

	for netName, network := range netvmispec.IndexNetworkSpecByName(vmi.Spec.Networks) {
		if _, isAttachmentToBeHotplugged := interfacesToHoplug[netName]; isAttachmentToBeHotplugged {
			networksToHotplug = append(networksToHotplug, network)
		}
	}

	return networksToHotplug
}

func indexedDomainInterfaces(domain *api.Domain) map[string]api.Interface {
	domainInterfaces := map[string]api.Interface{}
	for _, iface := range domain.Spec.Devices.Interfaces {
		domainInterfaces[iface.Alias.GetName()] = iface
	}
	return domainInterfaces
}

func AppendPlaceholderInterfacesToTheDomain(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec, count int) *api.DomainSpec {
	domainSpecWithIfacesResource := domainSpec.DeepCopy()
	for i := 0; i < count; i++ {
		domainSpecWithIfacesResource.Devices.Interfaces = append(
			domainSpecWithIfacesResource.Devices.Interfaces,
			newInterfacePlaceholder(i, converter.InterpretTransitionalModelType(vmi.Spec.Domain.Devices.UseVirtioTransitional, vmi.Spec.Architecture)),
		)
	}
	return domainSpecWithIfacesResource
}

func newInterfacePlaceholder(index int, modelType string) api.Interface {
	return api.Interface{
		Type:  "ethernet",
		Model: &api.Model{Type: modelType},
		Target: &api.InterfaceTarget{
			Device:  fmt.Sprintf("placeholder-%d", index),
			Managed: "no",
		},
	}
}

func isLinkStateEqual(iface1, iface2 api.Interface) bool {
	if iface1.LinkState == nil && iface2.LinkState == nil {
		return true
	}

	if iface1.LinkState == nil || iface2.LinkState == nil {
		return false
	}

	return iface1.LinkState.State == iface2.LinkState.State
}
