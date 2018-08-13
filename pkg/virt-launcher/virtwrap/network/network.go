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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

/*
 ATTENTION: Rerun code generators when interface signatures are modified.
*/

import (
	"fmt"

	"kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	podInterface        = "eth0"
	defaultDNS          = "8.8.8.8"
	resolvConf          = "/etc/resolv.conf"
	defaultSearchDomain = "cluster.local"
	domainSearchPrefix  = "search"
	nameserverPrefix    = "nameserver"
)

var interfaceCacheFile = "/var/run/kubevirt-private/interface-cache-%s.json"
var NetworkInterfaceFactory = getNetworkClass

type NetworkInterface interface {
	Plug(iface *v1.Interface, network *v1.Network, domain *api.Domain) error
	Unplug()
}

func SetupNetworkInterfaces(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	// prepare networks map
	networks := map[string]*v1.Network{}
	for _, network := range vmi.Spec.Networks {
		networks[network.Name] = network.DeepCopy()
	}

	if len(networks) == 0 {
		return fmt.Errorf("no networks were specified on a vm spec")
	}

	for _, iface := range vmi.Spec.Domain.Devices.Interfaces {

		network, ok := networks[iface.Name]
		if !ok {
			return fmt.Errorf("failed to find a network %s", iface.Name)
		}
		vif := NetworkInterfaceFactory(network)
		if vif == nil {
			continue
		}

		err := vif.Plug(&iface, network, domain)
		if err != nil {
			return err
		}
	}
	return nil
}

// a factory to get suitable network interface
func getNetworkClass(network *v1.Network) NetworkInterface {
	if network.Pod != nil {
		return new(PodInterface)
	}
	return nil
}
