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

	"kubevirt.io/kubevirt/pkg/apimachinery/resource"
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
		log.Log.Infof("will hot plug %q with MAC %q", network.Name, ifaceMAC)
		ifaceXML, err := xml.Marshal(relevantIface)
		if err != nil {
			return err
		}

		if err := vim.dom.AttachDeviceFlags(strings.ToLower(string(ifaceXML)), affectLiveAndConfigLibvirtFlags); err != nil {
			log.Log.Reason(err).Errorf("libvirt failed to attach interface %s: %v", network.Name, err)
			return err
		}
	}
	return nil
}

func (vim *virtIOInterfaceManager) hotUnplugVirtioInterface(vmi *v1.VirtualMachineInstance, currentDomain *api.Domain) error {
	for _, domainIface := range interfacesToHotUnplug(vmi.Spec.Domain.Devices.Interfaces, currentDomain.Spec.Devices.Interfaces) {
		log.Log.Infof("preparing to hot-unplug %s", domainIface.Alias.GetName())

		ifaceXML, err := xml.Marshal(domainIface)
		if err != nil {
			return err
		}

		if derr := vim.dom.DetachDeviceFlags(strings.ToLower(string(ifaceXML)), affectLiveAndConfigLibvirtFlags); derr != nil {
			log.Log.Reason(derr).Errorf("libvirt failed to detach interface %s: %v", domainIface.Alias.GetName(), derr)
			return derr
		}
	}
	return nil
}

func interfacesToHotUnplug(vmiSpecInterfaces []v1.Interface, domainSpecInterfaces []api.Interface) []api.Interface {
	ifaces2remove := netvmispec.FilterInterfacesSpec(vmiSpecInterfaces, func(i v1.Interface) bool {
		return i.State == v1.InterfaceStateAbsent
	})
	var domainIfacesToRemove []api.Interface
	for _, vmiIface := range ifaces2remove {
		if domainIface := lookupDomainInterfaceByName(domainSpecInterfaces, vmiIface.Name); domainIface != nil {
			if hasDeviceWithHashedTapName(domainIface.Target, vmiIface) {
				domainIfacesToRemove = append(domainIfacesToRemove, *domainIface)
			}
		}
	}
	return domainIfacesToRemove
}

func hasDeviceWithHashedTapName(target *api.InterfaceTarget, vmiIface v1.Interface) bool {
	return target != nil &&
		target.Device == virtnetlink.GenerateTapDeviceName(namescheme.GenerateHashedInterfaceName(vmiIface.Name))
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
	interfacesToHoplug := netvmispec.IndexInterfacesFromStatus(
		vmi.Status.Interfaces,
		func(ifaceStatus v1.VirtualMachineInstanceNetworkInterface) bool {
			_, exists := indexedDomainIfaces[ifaceStatus.Name]
			vmiSpecIface := netvmispec.LookupInterfaceByName(vmi.Spec.Domain.Devices.Interfaces, ifaceStatus.Name)

			return netvmispec.ContainsInfoSource(
				ifaceStatus.InfoSource, netvmispec.InfoSourceMultusStatus,
			) && !exists && vmiSpecIface.State != v1.InterfaceStateAbsent
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
	domainSpecWithoutIfacePlaceholders.DeepCopyInto(domainSpec)

	return f(vmi, domainSpecWithoutIfacePlaceholders)
}

func appendPlaceholderInterfacesToTheDomain(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) *api.DomainSpec {
	domainSpecWithIfacesResource := domainSpec.DeepCopy()
	requests := resource.ExtendedResourceList{ResourceList: vmi.Spec.Domain.Resources.Requests}
	reqInterfaces := int(requests.Interface().Value())
	interfacePlaceholderCount := reqInterfaces - len(vmi.Spec.Domain.Devices.Interfaces)
	for i := 0; i < interfacePlaceholderCount; i++ {
		domainSpecWithIfacesResource.Devices.Interfaces = append(
			domainSpecWithIfacesResource.Devices.Interfaces,
			newInterfacePlaceholder(i, converter.InterpretTransitionalModelType(vmi.Spec.Domain.Devices.UseVirtioTransitional)),
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
