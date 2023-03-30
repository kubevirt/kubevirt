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
	for _, networkName := range networksToHotplugWhoseInterfacesAreNotInTheDomain(vmi.Status.Interfaces, currentDomain.Spec.Devices.Interfaces) {
		log.Log.Infof("will hot plug %s", networkName)

		network := netvmispec.LookupNetworkByName(vmi.Spec.Networks, networkName)
		if err := vmConfigurator.SetupPodNetworkPhase2(updatedDomain, []v1.Network{*network}); err != nil {
			return err
		}

		relevantIface := lookupInterfaceName(currentDomain.Spec.Devices.Interfaces, networkName)
		if relevantIface == nil {
			return fmt.Errorf("could not retrieve %q Interface object from domain", networkName)
		}

		log.Log.Infof("will hot plug %s with MAC %s", networkName, relevantIface.MAC)
		ifaceXML, err := xml.Marshal(relevantIface)
		if err != nil {
			return err
		}

		if err := dom.AttachDeviceFlags(strings.ToLower(string(ifaceXML)), affectLiveAndConfigLibvirtFlags); err != nil {
			log.Log.Reason(err).Errorf("libvirt failed to attach interface %s: %v", networkName, err)
			return err
		}
	}
	return nil
}

func networksToHotplugWhoseInterfacesAreNotInTheDomain(vmiStatusInterfaces []v1.VirtualMachineInstanceNetworkInterface, domainInterfaces []api.Interface) []string {
	var networks []string
	lookupDomainIfaceWithName := indexedDomainInterfaces(domainInterfaces)
	for _, vmiStatusIface := range vmiStatusInterfaces {
		_, ifaceAttachedToDomain := lookupDomainIfaceWithName[vmiStatusIface.Name]
		ifaceExistInPod := netvmispec.ContainsInfoSource(vmiStatusIface.InfoSource, netvmispec.InfoSourceMultusStatus)

		if ifaceExistInPod && !ifaceAttachedToDomain {
			networks = append(networks, vmiStatusIface.Name)
		}
	}

	return networks
}

func indexedDomainInterfaces(ifaces []api.Interface) map[string]api.Interface {
	domainInterfaces := map[string]api.Interface{}
	for _, iface := range ifaces {
		domainInterfaces[iface.Alias.GetName()] = iface
	}
	return domainInterfaces
}

func lookupInterfaceName(ifaces []api.Interface, name string) *api.Interface {
	for _, iface := range ifaces {
		if iface.Alias.GetName() == name {
			i := iface
			return &i
		}
	}

	return nil
}
