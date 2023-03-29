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

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netsetup "kubevirt.io/kubevirt/pkg/network/setup"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

func hotplugVirtioInterface(vmi *v1.VirtualMachineInstance, dom cli.VirDomain, currentDomain *api.Domain, updatedDomain *api.Domain) error {
	vmConfigurator := netsetup.NewVMNetworkConfigurator(vmi, cache.CacheCreator{})
	for _, network := range networksToHotplugWhoseInterfacesAreNotInTheDomain(vmi, indexedDomainInterfaces(currentDomain)) {
		log.Log.Infof("will hot plug %s", network.Name)

		if err := vmConfigurator.SetupPodNetworkPhase2(updatedDomain, []v1.Network{network}); err != nil {
			return err
		}

		relevantIface := domainInterfaceFromNetwork(updatedDomain, network)
		if relevantIface == nil {
			return fmt.Errorf("could not retrieve the api.Interface object from the dummy domain")
		}

		log.Log.Infof("will hot plug %s with MAC %s", network.Name, *relevantIface.MAC)
		ifaceXML, err := xml.Marshal(relevantIface)
		if err != nil {
			return err
		}

		if err := dom.AttachDeviceFlags(strings.ToLower(string(ifaceXML)), affectLiveAndConfigLibvirtFlags); err != nil {
			log.Log.Reason(err).Errorf("libvirt failed to attach interface %s: %v", network.Name, err)
			return err
		}
	}
	return nil
}

func domainInterfaceFromNetwork(domain *api.Domain, network v1.Network) *api.Interface {
	for _, iface := range domain.Spec.Devices.Interfaces {
		if iface.Alias.GetName() == network.Name {
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

			return netvmispec.ContainsInfoSource(
				ifaceStatus.InfoSource, netvmispec.InfoSourceMultusStatus,
			) && !exists
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
