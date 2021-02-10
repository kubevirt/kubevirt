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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package network

import (
	"strconv"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	networkdriver "kubevirt.io/kubevirt/pkg/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type MacvtapNetworkingVMConfigurator struct {
	vmi              *v1.VirtualMachineInstance
	iface            *v1.Interface
	virtIface        *api.Interface
	domain           *api.Domain
	podInterfaceName string
	podNicLink       netlink.Link
	launcherPID      int
}

func (b *MacvtapNetworkingVMConfigurator) discoverPodNetworkInterface() error {
	link, err := networkdriver.Handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return err
	}
	b.podNicLink = link

	if b.virtIface.MAC == nil {
		// Get interface MAC address
		mac, err := networkdriver.Handler.GetMacDetails(b.podInterfaceName)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get MAC for %s", b.podInterfaceName)
			return err
		}
		b.virtIface.MAC = &api.MAC{MAC: mac.String()}
	}

	b.virtIface.MTU = &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)}
	b.virtIface.Target = &api.InterfaceTarget{
		Device:  b.podInterfaceName,
		Managed: "no",
	}

	return nil
}

func (b *MacvtapNetworkingVMConfigurator) prepareVMNetworkingInterfaces() error {
	return nil
}

func (b *MacvtapNetworkingVMConfigurator) loadCachedInterface() error {
	return nil
}

func (b *MacvtapNetworkingVMConfigurator) exportVIF() error {
	return nil
}

func (b *MacvtapNetworkingVMConfigurator) cacheInterface() error {
	return nil
}
