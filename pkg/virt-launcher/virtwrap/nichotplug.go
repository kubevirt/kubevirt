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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package virtwrap

import (
	"encoding/xml"
	"fmt"
	"strings"

	"kubevirt.io/kubevirt/pkg/network/namescheme"

	"libvirt.org/go/libvirt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/util"
)

type vmConfigurator interface {
	SetupPodNetworkPhase2(domain *api.Domain, networksToPlug []v1.Network) error
}

type virtIOInterfaceManager struct {
	dom          cli.VirDomain
	configurator vmConfigurator
}

const (
	// ReservedInterfaces represents the number of interfaces the domain
	// should reserve for future hotplug additions.
	ReservedInterfaces            = 4
	LibvirtInterfaceLinkStateUP   = "up"
	LibvirtInterfaceLinkStateDown = "down"
)

func newVirtIOInterfaceManager(
	libvirtClient cli.VirDomain,
	configurator vmConfigurator,
) *virtIOInterfaceManager {
	return &virtIOInterfaceManager{
		dom:          libvirtClient,
		configurator: configurator,
	}
}

func (vim *virtIOInterfaceManager) hotplugVirtioInterface(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain, updatedDomain *api.Domain) error {
	for _, network := range networksToHotplugWhoseInterfacesAreNotInTheDomain(vmi, indexedDomainInterfaces(currentDomain)) {
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
		vmiIface := netvmispec.LookupInterfaceByName(vmi.Spec.Domain.Devices.Interfaces, relevantIface.Alias.GetName())
		if vmiIface != nil && vmiIface.State == v1.InterfaceStateLinkDown {
			relevantIface.LinkState = &api.LinkState{State: LibvirtInterfaceLinkStateDown}
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

// normalizeDomainLinkState anything other than down is considered up
func normalizeDomainLinkState(iface *api.Interface) v1.InterfaceState {
	if iface.LinkState != nil && iface.LinkState.State == LibvirtInterfaceLinkStateDown {
		return v1.InterfaceStateLinkDown
	}
	return v1.InterfaceStateLinkUp
}

// normalizeVMLinkState anything other than down is considered up
func normalizeVMLinkState(iface *v1.Interface) v1.InterfaceState {
	if iface.State == v1.InterfaceStateLinkDown {
		return iface.State
	}
	return v1.InterfaceStateLinkUp
}

func requiredLinkAction(vmiIface v1.Interface, domIface api.Interface) v1.InterfaceState {
	normalizedVMLinkState := normalizeVMLinkState(&vmiIface)
	normalizedDomainLinkState := normalizeDomainLinkState(&domIface)
	switch {
	case normalizedVMLinkState == v1.InterfaceStateLinkUp && normalizedDomainLinkState == v1.InterfaceStateLinkDown:
		return v1.InterfaceStateLinkUp
	case normalizedVMLinkState == v1.InterfaceStateLinkDown && normalizedDomainLinkState == v1.InterfaceStateLinkUp:
		return v1.InterfaceStateLinkDown
	}
	return ""
}

func (vim *virtIOInterfaceManager) updateDomainWithLinkState(domIfaceToPatch *api.Interface, targetState v1.InterfaceState) error {
	if targetState == "" {
		return nil
	}
	log.Log.Infof("preparing to set link %s for %s", domIfaceToPatch.Alias.GetName(), targetState)
	linkTargetStateInDomain := LibvirtInterfaceLinkStateUP
	//seems obvious but mapping kubevirt terminology to libvirt
	if targetState == v1.InterfaceStateLinkDown {
		linkTargetStateInDomain = LibvirtInterfaceLinkStateDown
	}
	domIfaceToPatch.LinkState = &api.LinkState{State: linkTargetStateInDomain}
	ifaceXML, err := xml.Marshal(domIfaceToPatch)
	if err != nil {
		return err
	}
	if err = vim.dom.UpdateDeviceFlags(strings.ToLower(string(ifaceXML)), affectDeviceLiveAndConfigLibvirtFlags); err != nil {
		log.Log.Reason(err).Errorf("libvirt failed to set interface %s to %s, %v", domIfaceToPatch.Alias.GetName(), targetState, err)
		return err
	}
	return nil
}
func (vim *virtIOInterfaceManager) setLinkUpDownVirtioInterface(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain) error {
	domainIfacesByAlias := indexedDomainInterfaces(currentDomain)
	for _, vmiIface := range vmi.Spec.Domain.Devices.Interfaces {
		domIface, ok := domainIfacesByAlias[vmiIface.Name]
		if !ok {
			return fmt.Errorf("could not find domain interface %s for %s", vmiIface.Name, currentDomain.Name)
		}
		if err := vim.updateDomainWithLinkState(&domIface, requiredLinkAction(vmiIface, domIface)); err != nil {
			return err
		}
	}
	return nil
}

func (vim *virtIOInterfaceManager) hotUnplugVirtioInterface(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain) error {
	for _, domainIface := range interfacesToHotUnplug(vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, currentDomain.Spec.Devices.Interfaces) {
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

func interfacesToHotUnplug(vmiSpecInterfaces []v1.Interface, vmiSpecNets []v1.Network, domainSpecInterfaces []api.Interface) []api.Interface {
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

func networksToHotplugWhoseInterfacesAreNotInTheDomain(vmi *v1.VirtualMachineInstance, indexedDomainIfaces map[string]api.Interface) []v1.Network {
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

// withNetworkIfacesResources adds network interfaces as placeholders to the domain spec
// to trigger the addition of the dependent resources/devices (e.g. PCI controllers).
// As its last step, it reads the generated configuration and removes the network interfaces
// so none will be created with the domain creation.
// The dependent devices are left in the configuration, to allow future hotplug.
func withNetworkIfacesResources(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec, f func(v *v1.VirtualMachineInstance, s *api.DomainSpec) (cli.VirDomain, error)) (cli.VirDomain, error) {
	domainSpecWithIfacesResource := appendPlaceholderInterfacesToTheDomain(vmi, domainSpec)
	dom, err := f(vmi, domainSpecWithIfacesResource)
	if err != nil {
		return nil, err
	}

	if len(domainSpec.Devices.Interfaces) == len(domainSpecWithIfacesResource.Devices.Interfaces) {
		return dom, nil
	}

	domainSpecWithoutIfacePlaceholders, err := util.GetDomainSpecWithFlags(dom, libvirt.DOMAIN_XML_INACTIVE)
	if err != nil {
		return nil, err
	}
	domainSpecWithoutIfacePlaceholders.Devices.Interfaces = domainSpec.Devices.Interfaces
	// Only the devices are taken into account because some parameters are not assured to be returned when
	// getting the domain spec (e.g. the `qemu:commandline` section).
	domainSpecWithoutIfacePlaceholders.Devices.DeepCopyInto(&domainSpec.Devices)

	return f(vmi, domainSpec)
}

func appendPlaceholderInterfacesToTheDomain(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) *api.DomainSpec {
	if len(vmi.Spec.Domain.Devices.Interfaces) == 0 {
		return domainSpec
	}
	if val := vmi.Annotations[v1.PlacePCIDevicesOnRootComplex]; val == "true" {
		return domainSpec
	}
	domainSpecWithIfacesResource := domainSpec.DeepCopy()
	interfacePlaceholderCount := ReservedInterfaces - len(vmi.Spec.Domain.Devices.Interfaces)
	for i := 0; i < interfacePlaceholderCount; i++ {
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
