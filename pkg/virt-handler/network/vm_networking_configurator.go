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
	"encoding/json"
	"fmt"
	"os"

	"github.com/vishvananda/netlink"
	"k8s.io/apimachinery/pkg/types"
	netutils "k8s.io/utils/net"

	"kubevirt.io/client-go/log"

	v1 "kubevirt.io/client-go/api/v1"
	networkdriver "kubevirt.io/kubevirt/pkg/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type VMInterfaceConfigurator interface {
	SetupPodInfrastructure() error
	SetupPodInfrastructureForIface(iface *v1.Interface, network *v1.Network, podInterfaceName string) error
	GenerateVMNetworkingConfiguratorForIface(iface *v1.Interface, network *v1.Network, podInterfaceName string) (VMNetworkingConfiguration, error)
}

type VMInterfaceConfiguratorImpl struct {
	vmi         *v1.VirtualMachineInstance
	launcherPID int
}

func NewVMNetworkConfigurator(vmi *v1.VirtualMachineInstance, pid int) VMInterfaceConfiguratorImpl {
	return VMInterfaceConfiguratorImpl{
		vmi:         vmi,
		launcherPID: pid,
	}
}

func (vic *VMInterfaceConfiguratorImpl) SetupPodInfrastructure() error {
	if err := networkdriver.CreateVirtHandlerCacheDir(vic.vmi.ObjectMeta.UID); err != nil {
		return err
	}
	networks, cniNetworks := networkdriver.GetNetworksAndCniNetworks(vic.vmi)
	for _, iface := range vic.vmi.Spec.Domain.Devices.Interfaces {
		network := networks[iface.Name]
		cniNetworkIndex := cniNetworks[iface.Name]
		podInterfaceName := networkdriver.GetPodInterfaceName(network, cniNetworkIndex)
		if err := vic.SetupPodInfrastructureForIface(&iface, network, podInterfaceName); err != nil {
			return err
		}
	}
	return nil
}

func (vic *VMInterfaceConfiguratorImpl) SetupPodInfrastructureForIface(iface *v1.Interface, network *v1.Network, podInterfaceName string) error {
	networkdriver.InitHandler()
	// There is nothing to plug for SR-IOV devices
	if iface.SRIOV != nil {
		return nil
	}

	vmNetworkingConfigurator, err := vic.GenerateVMNetworkingConfiguratorForIface(iface, network, podInterfaceName)
	if err != nil {
		return err
	}
	if err := vmNetworkingConfigurator.loadCachedInterface(); err != nil {
		return err
	}
	// ignore the vmNetworkingConfigurator.loadCachedInterface for slirp and set the Pod interface cache
	isIfaceCached := vmNetworkingConfigurator.hasCachedInterface()
	if !isIfaceCached || iface.Slirp != nil {
		if err := setPodInterfaceCache(iface, podInterfaceName, string(vic.vmi.UID)); err != nil {
			return err
		}
	}
	if !isIfaceCached {
		err = vmNetworkingConfigurator.discoverPodNetworkInterface()
		if err != nil {
			return err
		}

		if err := vmNetworkingConfigurator.prepareVMNetworkingInterfaces(); err != nil {
			log.Log.Reason(err).Error("failed to prepare pod networking")
			return createCriticalNetworkError(err)
		}

		if err := vmNetworkingConfigurator.CacheInterface(); err != nil {
			log.Log.Reason(err).Error("failed to save interface configuration")
			return createCriticalNetworkError(err)
		}

		if err = vmNetworkingConfigurator.ExportVIF(); err != nil {
			log.Log.Reason(err).Error("failed to save vif configuration")
			return createCriticalNetworkError(err)
		}
	}

	return nil
}

func createCriticalNetworkError(err error) *networkdriver.CriticalNetworkError {
	return &networkdriver.CriticalNetworkError{Msg: fmt.Sprintf("Critical network error: %v", err)}
}

func (vic *VMInterfaceConfiguratorImpl) GenerateVMNetworkingConfiguratorForIface(iface *v1.Interface, network *v1.Network, podInterfaceName string) (VMNetworkingConfiguration, error) {
	// nothing to do for SRIOV
	if iface.SRIOV != nil {
		return nil, nil
	}
	if iface.Bridge != nil {
		bridgeNetworkingConfigurator, err := generateBridgeVMNetworkingConfigurator(vic.vmi, iface, podInterfaceName, vic.launcherPID)
		return VMNetworkingConfiguration(&bridgeNetworkingConfigurator), err
	}
	if iface.Masquerade != nil {
		masqueradeNetworkingConfigurator, err := generateMasqueradeVMNetworkingConfigurator(vic.vmi, iface, network, podInterfaceName, vic.launcherPID)
		return VMNetworkingConfiguration(&masqueradeNetworkingConfigurator), err
	}
	if iface.Slirp != nil {
		return VMNetworkingConfiguration(&SlirpNetworkingVMConfigurator{vmi: vic.vmi, iface: iface, launcherPID: vic.launcherPID}), nil
	}
	if iface.Macvtap != nil {
		macvtapNetworkingConfigurator, err := generateMacvtapVMNetworkingConfigurator(vic.vmi, iface, podInterfaceName, vic.launcherPID)
		return VMNetworkingConfiguration(&macvtapNetworkingConfigurator), err
	}
	return nil, fmt.Errorf("Not implemented")
}

type VMNetworkingConfiguration interface {
	CacheInterface() error
	discoverPodNetworkInterface() error
	ExportVIF() error
	loadCachedInterface() error
	hasCachedInterface() bool
	prepareVMNetworkingInterfaces() error
}

func createAndBindTapToBridge(deviceName string, bridgeIfaceName string, queueNumber uint32, launcherPID int, mtu int) error {
	err := networkdriver.Handler.CreateTapDevice(deviceName, queueNumber, launcherPID, mtu)
	if err != nil {
		return err
	}
	return networkdriver.Handler.BindTapDeviceToBridge(deviceName, bridgeIfaceName)
}

func generateTapDeviceName(podInterfaceName string) string {
	return "tap" + podInterfaceName[3:]
}

func loadCachedInterface(launcherPID int, ifaceName string) (*api.Interface, error) {
	var ifaceConfig *api.Interface
	err := networkdriver.ReadFromVirtLauncherCachedFile(&ifaceConfig, fmt.Sprintf("%d", launcherPID), ifaceName)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return ifaceConfig, nil
}

func setCachedVIF(vif networkdriver.VIF, pid int, ifaceName string) error {
	buf, err := json.MarshalIndent(vif, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling vif object: %v", err)
	}

	launcherPID := "self"
	if pid != 0 {
		launcherPID = fmt.Sprintf("%d", pid)
	}
	return networkdriver.WriteVifFile(buf, launcherPID, ifaceName)
}

type PodCacheInterface struct {
	Iface  *v1.Interface `json:"iface,omitempty"`
	PodIP  string        `json:"podIP,omitempty"`
	PodIPs []string      `json:"podIPs,omitempty"`
}

func setPodInterfaceCache(iface *v1.Interface, podInterfaceName string, uid string) error {
	cache := PodCacheInterface{Iface: iface}

	ipv4, ipv6, err := readIPAddressesFromLink(podInterfaceName)
	if err != nil {
		return err
	}

	switch {
	case ipv4 != "" && ipv6 != "":
		cache.PodIPs, err = sortIPsBasedOnPrimaryIP(ipv4, ipv6)
		if err != nil {
			return err
		}
	case ipv4 != "":
		cache.PodIPs = []string{ipv4}
	case ipv6 != "":
		cache.PodIPs = []string{ipv6}
	default:
		return nil
	}

	cache.PodIP = cache.PodIPs[0]
	err = networkdriver.WriteToVirtHandlerCachedFile(cache, types.UID(uid), iface.Name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to write pod Interface to cache, %s", err.Error())
		return err
	}

	return nil
}

func readIPAddressesFromLink(podInterfaceName string) (string, string, error) {
	link, err := networkdriver.Handler.LinkByName(podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podInterfaceName)
		return "", "", err
	}

	// get IP address
	addrList, err := networkdriver.Handler.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a address for interface: %s", podInterfaceName)
		return "", "", err
	}

	// no ip assigned. ipam disabled
	if len(addrList) == 0 {
		return "", "", nil
	}

	var ipv4, ipv6 string
	for _, addr := range addrList {
		if addr.IP.IsGlobalUnicast() {
			if netutils.IsIPv6(addr.IP) && ipv6 == "" {
				ipv6 = addr.IP.String()
			} else if !netutils.IsIPv6(addr.IP) && ipv4 == "" {
				ipv4 = addr.IP.String()
			}
		}
	}

	return ipv4, ipv6, nil
}

// sortIPsBasedOnPrimaryIP returns a sorted slice of IP/s based on the detected cluster primary IP.
// The operation clones the Pod status IP list order logic.
func sortIPsBasedOnPrimaryIP(ipv4, ipv6 string) ([]string, error) {
	ipv4Primary, err := networkdriver.Handler.IsIpv4Primary()
	if err != nil {
		return nil, err
	}

	if ipv4Primary {
		return []string{ipv4, ipv6}, nil
	}

	return []string{ipv6, ipv4}, nil
}
