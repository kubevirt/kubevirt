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
 */

package ifacehook

import (
	"fmt"
	"strings"

	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/network/namescheme"
	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

// HasheIfaceNameHook handles CPU pinning adjustments for dedicated CPU migrations
func HasheIfaceNameHook(vmi *v1.VirtualMachineInstance, domain *libvirtxml.Domain) error {
	if !namescheme.HasOrdinalSecondaryIfaces(vmi.Spec.Networks, vmi.Status.Interfaces) {
		return nil
	}
	domainIfacesByAlias := indexedDomainInterfacesByAlias(domain)
	for _, network := range vmi.Spec.Networks {

		if netvmispec.IsSecondaryMultusNetwork(network) {
			// wasteful
			tapIfaceName := link.GenerateTapDeviceName(namescheme.GenerateHashedInterfaceName(network.Name), network)
			fmt.Printf("DEBUG: before update %s  %+v\n", network.Name, domainIfacesByAlias[network.Name].Target.Dev)
			*(domainIfacesByAlias[network.Name].Target) = libvirtxml.DomainInterfaceTarget{Dev: tapIfaceName, Managed: "no"}
			fmt.Printf("DEBUG: after update %s  %+v\n\n", network.Name, domainIfacesByAlias[network.Name].Target.Dev)
		}
	}

	log.Log.Object(vmi).Info("HasheIfaceNameHook: processing completed")
	return nil
}

func indexedDomainInterfacesByAlias(domain *libvirtxml.Domain) map[string]*libvirtxml.DomainInterface {
	domainInterfaces := map[string]*libvirtxml.DomainInterface{}

	for i, iface := range domain.Devices.Interfaces {
		domainInterfaces[strings.TrimPrefix(iface.Alias.Name, api.UserAliasPrefix)] = &domain.Devices.Interfaces[i]
	}
	return domainInterfaces
}
