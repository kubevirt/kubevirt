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

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE

package network

import (
	"os"
	"strconv"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type MacvtapBindMechanism struct {
	vmi              *v1.VirtualMachineInstance
	iface            *v1.Interface
	virtIface        *api.Interface
	domain           *api.Domain
	podInterfaceName string
	podNicLink       netlink.Link
}

func (b *MacvtapBindMechanism) discoverPodNetworkInterface() error {
	link, err := Handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return err
	}
	b.podNicLink = link

	if b.virtIface.MAC == nil {
		// Get interface MAC address
		mac, err := Handler.GetMacDetails(b.podInterfaceName)
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

func (b *MacvtapBindMechanism) preparePodNetworkInterfaces(queueNumber uint32, launcherPID int) error {
	return nil
}

func (b *MacvtapBindMechanism) decorateConfig() error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.iface.Name {
			ifaces[i].MTU = b.virtIface.MTU
			ifaces[i].MAC = b.virtIface.MAC
			ifaces[i].Target = b.virtIface.Target
			break
		}
	}
	return nil
}

func (b *MacvtapBindMechanism) loadCachedInterface(pid, name string) (bool, error) {
	var ifaceConfig api.Interface

	err := readFromVirtLauncherCachedFile(&ifaceConfig, pid, name)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	b.virtIface = &ifaceConfig
	return true, nil
}

func (b *MacvtapBindMechanism) setCachedInterface(pid, name string) error {
	err := writeToVirtLauncherCachedFile(b.virtIface, pid, name)
	return err
}

func (b *MacvtapBindMechanism) loadCachedVIF(pid, name string) (bool, error) {
	return true, nil
}

func (b *MacvtapBindMechanism) setCachedVIF(pid, name string) error {
	return nil
}

func (b *MacvtapBindMechanism) startDHCP(vmi *v1.VirtualMachineInstance) error {
	// macvtap will connect to the host's subnet
	return nil
}

func createAndBindTapToBridge(deviceName string, bridgeIfaceName string, queueNumber uint32, launcherPID int, mtu int) error {
	err := Handler.CreateTapDevice(deviceName, queueNumber, launcherPID, mtu)
	if err != nil {
		return err
	}
	return Handler.BindTapDeviceToBridge(deviceName, bridgeIfaceName)
}

func generateTapDeviceName(podInterfaceName string) string {
	return "tap" + podInterfaceName[3:]
}
