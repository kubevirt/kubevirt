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
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

func hotplugVirtioInterface(vmi *v1.VirtualMachineInstance, converterContext *converter.ConverterContext, dom cli.VirDomain, domain *api.Domain) error {
	vmConfigurator := netsetup.NewVMNetworkConfigurator(vmi, cache.CacheCreator{})
	indexedInterfaces := netvmispec.IndexInterfaceSpecByName(vmi.Spec.Domain.Devices.Interfaces)
	for _, network := range netvmispec.NetworksToHotplugWhosePodIfacesAreReady(vmi) {
		log.Log.Infof("will hot plug %s", network.Name)

		ifaceToHotplug, wasFound := indexedInterfaces[network.Name]
		if !wasFound {
			return fmt.Errorf("could not find a matching interface for network: %s", network.Name)
		}

		domainInterfaces, err := converter.CreateDomainInterfaces(vmi, domain, converterContext, ifaceToHotplug)
		if err != nil {
			return err
		}

		domain.Spec.Devices.Interfaces = append(domain.Spec.Devices.Interfaces, domainInterfaces...)
		if err := vmConfigurator.SetupPodNetworkPhase2(domain, network); err != nil {
			return err
		}

		relevantIface := domainInterfaceFromNetwork(domain, network)
		if relevantIface == nil {
			return fmt.Errorf("could not retrieve the api.Interface object from the dummy domain")
		}

		log.Log.Infof("will hot plug %s with MAC %s", network.Name, *relevantIface.MAC)
		ifaceXML, err := xml.Marshal(relevantIface)
		if err != nil {
			return err
		}

		if err := dom.AttachDevice(strings.ToLower(string(ifaceXML))); err != nil {
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
