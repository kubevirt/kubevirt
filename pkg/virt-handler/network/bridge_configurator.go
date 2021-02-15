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
	"fmt"
	"strconv"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	networkdriver "kubevirt.io/kubevirt/pkg/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const bridgeFakeIPTemplate = "169.254.75.1%d/32"

type BridgedNetworkingVMConfigurator struct {
	bridgeInterfaceName string
	iface               v1.Interface
	launcherPID         int
	queueNumber         uint32
	podInterfaceName    string
	podNicLink          netlink.Link
	vif                 networkdriver.VIF
	virtIface           *api.Interface
	vmi                 *v1.VirtualMachineInstance
}

func generateBridgeVMNetworkingConfigurator(vmi *v1.VirtualMachineInstance, iface *v1.Interface, podInterfaceName string, launcherPID int) (BridgedNetworkingVMConfigurator, error) {
	mac, err := networkdriver.RetrieveMacAddress(iface)
	if err != nil {
		return BridgedNetworkingVMConfigurator{}, err
	}
	vif := &networkdriver.VIF{Name: podInterfaceName}
	if mac != nil {
		vif.MAC = *mac
	}

	queueNumber := uint32(0)
	isMultiqueue := (vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue != nil) && (*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue)
	if isMultiqueue {
		queueNumber = converter.CalculateNetworkQueues(vmi)
	}
	return BridgedNetworkingVMConfigurator{
		iface:               *iface,
		virtIface:           &api.Interface{},
		vmi:                 vmi,
		vif:                 *vif,
		podInterfaceName:    podInterfaceName,
		bridgeInterfaceName: fmt.Sprintf("k6t-%s", podInterfaceName),
		launcherPID:         launcherPID,
		queueNumber:         queueNumber,
	}, nil
}

func (b *BridgedNetworkingVMConfigurator) discoverPodNetworkInterface() error {
	link, err := networkdriver.Handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return err
	}
	b.podNicLink = link

	// get IP address
	addrList, err := networkdriver.Handler.AddrList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", b.podInterfaceName)
		return err
	}
	if len(addrList) == 0 {
		b.vif.IPAMDisabled = true
	} else {
		b.vif.IP = addrList[0]
		b.vif.IPAMDisabled = false
	}

	if len(b.vif.MAC) == 0 {
		// Get interface MAC address
		mac, err := networkdriver.Handler.GetMacDetails(b.podInterfaceName)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to get MAC for %s", b.podInterfaceName)
			return err
		}
		b.vif.MAC = mac
	}

	if b.podNicLink.Attrs().MTU < 0 || b.podNicLink.Attrs().MTU > 65535 {
		return fmt.Errorf("MTU value out of range ")
	}

	// Get interface MTU
	b.vif.Mtu = uint16(b.podNicLink.Attrs().MTU)

	if !b.vif.IPAMDisabled {
		// Handle interface routes
		if err := b.setInterfaceRoutes(); err != nil {
			return err
		}
	}
	return nil
}

func (b *BridgedNetworkingVMConfigurator) setInterfaceRoutes() error {
	routes, err := networkdriver.Handler.RouteList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", b.podInterfaceName)
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("No gateway address found in routes for %s", b.podInterfaceName)
	}
	b.vif.Gateway = routes[0].Gw
	if len(routes) > 1 {
		dhcpRoutes := networkdriver.FilterPodNetworkRoutes(routes, &b.vif)
		b.vif.Routes = &dhcpRoutes
	}
	return nil
}

func (b *BridgedNetworkingVMConfigurator) prepareVMNetworkingInterfaces() error {
	// Set interface link to down to change its MAC address
	if err := networkdriver.Handler.LinkSetDown(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", b.podInterfaceName)
		return err
	}

	if _, err := networkdriver.Handler.SetRandomMac(b.podInterfaceName); err != nil {
		return err
	}

	if err := networkdriver.Handler.LinkSetUp(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.podInterfaceName)
		return err
	}

	if err := b.createBridge(); err != nil {
		return err
	}

	tapDeviceName := generateTapDeviceName(b.podInterfaceName)
	err := createAndBindTapToBridge(tapDeviceName, b.bridgeInterfaceName, b.queueNumber, b.launcherPID, int(b.vif.Mtu))
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", tapDeviceName)
		return err
	}

	if !b.vif.IPAMDisabled {
		// Remove IP from POD interface
		err := networkdriver.Handler.AddrDel(b.podNicLink, &b.vif.IP)

		if err != nil {
			log.Log.Reason(err).Errorf("failed to delete address for interface: %s", b.podInterfaceName)
			return err
		}
	}

	if err := networkdriver.Handler.LinkSetLearningOff(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to disable mac learning for interface: %s", b.podInterfaceName)
		return err
	}

	b.virtIface = &api.Interface{
		MAC: &api.MAC{MAC: b.vif.MAC.String()},
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  tapDeviceName,
			Managed: "no",
		},
	}

	return nil
}

func (b *BridgedNetworkingVMConfigurator) createBridge() error {
	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: b.bridgeInterfaceName,
		},
	}
	err := networkdriver.Handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}

	err = networkdriver.Handler.LinkSetMaster(b.podNicLink, bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to connect interface %s to bridge %s", b.podInterfaceName, bridge.Name)
		return err
	}

	err = networkdriver.Handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	// set fake ip on a bridge
	addr, err := b.getFakeBridgeIP()
	if err != nil {
		return err
	}
	fakeaddr, err := networkdriver.Handler.ParseAddr(addr)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	if err := networkdriver.Handler.AddrAdd(bridge, fakeaddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}

	if err = networkdriver.Handler.DisableTXOffloadChecksum(b.bridgeInterfaceName); err != nil {
		log.Log.Reason(err).Error("failed to disable TX offload checksum on bridge interface")
		return err
	}

	return nil
}

func (b *BridgedNetworkingVMConfigurator) getFakeBridgeIP() (string, error) {
	ifaces := b.vmi.Spec.Domain.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Name == b.iface.Name {
			return fmt.Sprintf(bridgeFakeIPTemplate, i), nil
		}
	}
	return "", fmt.Errorf("Failed to generate bridge fake address for interface %s", b.iface.Name)
}

func (b *BridgedNetworkingVMConfigurator) loadCachedInterface() error {
	cachedIface, err := loadCachedInterface(b.launcherPID, b.iface.Name)
	if cachedIface != nil {
		b.virtIface = cachedIface
	}
	return err
}

func (b *BridgedNetworkingVMConfigurator) cacheInterface() error {
	return networkdriver.WriteToVirtLauncherCachedFile(b.virtIface, fmt.Sprintf("%d", b.launcherPID), b.iface.Name)
}

func (b *BridgedNetworkingVMConfigurator) exportVIF() error {
	return setCachedVIF(b.vif, b.launcherPID, b.iface.Name)
}

func (b *BridgedNetworkingVMConfigurator) hasCachedInterface() bool {
	return b.virtIface != nil
}
