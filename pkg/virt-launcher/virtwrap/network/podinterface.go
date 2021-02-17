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
	"fmt"
	"io/ioutil"
	"net"
	"os"

	"k8s.io/apimachinery/pkg/types"
	netutils "k8s.io/utils/net"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var bridgeFakeIP = "169.254.75.1%d/32"

type podNICImpl struct{}

func getVifFilePath(pid, name string) string {
	return fmt.Sprintf(vifCacheFile, pid, name)
}

func writeVifFile(buf []byte, pid, name string) error {
	err := ioutil.WriteFile(getVifFilePath(pid, name), buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing vif object: %v", err)
	}
	return nil
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
	err = WriteToVirtHandlerCachedFile(cache, types.UID(uid), iface.Name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to write pod Interface to cache, %s", err.Error())
		return err
	}

	return nil
}

func readIPAddressesFromLink(podInterfaceName string) (string, string, error) {
	link, err := Handler.LinkByName(podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podInterfaceName)
		return "", "", err
	}

	// get IP address
	addrList, err := Handler.AddrList(link, netlink.FAMILY_ALL)
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
	ipv4Primary, err := Handler.IsIpv4Primary()
	if err != nil {
		return nil, err
	}

	if ipv4Primary {
		return []string{ipv4, ipv6}, nil
	}

	return []string{ipv6, ipv4}, nil
}

func (l *podNICImpl) PlugPhase1(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, podInterfaceName string, pid int) error {
	initHandler()

	// There is nothing to plug for SR-IOV devices
	if iface.SRIOV != nil {
		return nil
	}

	bindMechanism, err := getPhase1Binding(vmi, iface, network, podInterfaceName)
	if err != nil {
		return err
	}

	pidStr := fmt.Sprintf("%d", pid)
	isExist, err := bindMechanism.loadCachedInterface(pidStr, iface.Name)
	if err != nil {
		return err
	}

	// ignore the bindMechanism.loadCachedInterface for slirp and set the Pod interface cache
	if !isExist || iface.Slirp != nil {
		err := setPodInterfaceCache(iface, podInterfaceName, string(vmi.ObjectMeta.UID))
		if err != nil {
			return err
		}
	}
	if !isExist {
		err = bindMechanism.discoverPodNetworkInterface()
		if err != nil {
			return err
		}

		queueNumber := uint32(0)
		isMultiqueue := (vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue != nil) && (*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue)
		if isMultiqueue {
			queueNumber = converter.CalculateNetworkQueues(vmi)
		}
		if err := bindMechanism.preparePodNetworkInterfaces(queueNumber, pid); err != nil {
			log.Log.Reason(err).Error("failed to prepare pod networking")
			return createCriticalNetworkError(err)
		}

		err = bindMechanism.setCachedInterface(pidStr, iface.Name)
		if err != nil {
			log.Log.Reason(err).Error("failed to save interface configuration")
			return createCriticalNetworkError(err)
		}

		err = bindMechanism.setCachedVIF(pidStr, iface.Name)
		if err != nil {
			log.Log.Reason(err).Error("failed to save vif configuration")
			return createCriticalNetworkError(err)
		}
	}

	return nil
}

func createCriticalNetworkError(err error) *CriticalNetworkError {
	return &CriticalNetworkError{fmt.Sprintf("Critical network error: %v", err)}
}

func ensureDHCP(vmi *v1.VirtualMachineInstance, bindMechanism BindMechanism, podInterfaceName string) error {
	dhcpStartedFile := fmt.Sprintf("/var/run/kubevirt-private/dhcp_started-%s", podInterfaceName)
	_, err := os.Stat(dhcpStartedFile)
	if os.IsNotExist(err) {
		if err := bindMechanism.startDHCP(vmi); err != nil {
			return fmt.Errorf("failed to start DHCP server for interface %s", podInterfaceName)
		}
		newFile, err := os.Create(dhcpStartedFile)
		if err != nil {
			return fmt.Errorf("failed to create dhcp started file %s: %s", dhcpStartedFile, err)
		}
		newFile.Close()
	}
	return nil
}

func (l *podNICImpl) PlugPhase2(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) error {
	precond.MustNotBeNil(domain)
	initHandler()

	// There is nothing to plug for SR-IOV devices
	if iface.SRIOV != nil {
		return nil
	}

	bindMechanism, err := getPhase2Binding(vmi, iface, network, domain, podInterfaceName)
	if err != nil {
		return err
	}

	pid := "self"

	isExist, err := bindMechanism.loadCachedInterface(pid, iface.Name)
	if err != nil {
		log.Log.Reason(err).Critical("failed to load cached interface configuration")
	}
	if !isExist {
		log.Log.Reason(err).Critical("cached interface configuration doesn't exist")
	}

	isExist, err = bindMechanism.loadCachedVIF(pid, iface.Name)
	if err != nil {
		log.Log.Reason(err).Critical("failed to load cached vif configuration")
	}
	if !isExist {
		log.Log.Reason(err).Critical("cached vif configuration doesn't exist")
	}

	err = bindMechanism.decorateConfig()
	if err != nil {
		log.Log.Reason(err).Critical("failed to create libvirt configuration")
	}

	err = ensureDHCP(vmi, bindMechanism, podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Criticalf("failed to ensure dhcp service running for %s: %s", podInterfaceName, err)
		panic(err)
	}

	return nil
}

// The only difference between bindings for two phases is that the first phase
// should not require access to domain definition, hence we pass nil instead of
// it. This means that any functions called under phase1 code path should not
// use the domain set on the binding.
func getPhase1Binding(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, podInterfaceName string) (BindMechanism, error) {
	return getPhase2Binding(vmi, iface, network, nil, podInterfaceName)
}

func getPhase2Binding(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) (BindMechanism, error) {
	retrieveMacAddress := func(iface *v1.Interface) (*net.HardwareAddr, error) {
		if iface.MacAddress != "" {
			macAddress, err := net.ParseMAC(iface.MacAddress)
			if err != nil {
				return nil, err
			}
			return &macAddress, nil
		}
		return nil, nil
	}

	if iface.Bridge != nil {
		mac, err := retrieveMacAddress(iface)
		if err != nil {
			return nil, err
		}
		vif := &VIF{Name: podInterfaceName}
		if mac != nil {
			vif.MAC = *mac
		}
		return &BridgeBindMechanism{iface: iface,
			virtIface:           &api.Interface{},
			vmi:                 vmi,
			vif:                 vif,
			domain:              domain,
			podInterfaceName:    podInterfaceName,
			bridgeInterfaceName: fmt.Sprintf("k6t-%s", podInterfaceName)}, nil
	}
	if iface.Masquerade != nil {
		mac, err := retrieveMacAddress(iface)
		if err != nil {
			return nil, err
		}
		vif := &VIF{Name: podInterfaceName}
		if mac != nil {
			vif.MAC = *mac
		}
		return &MasqueradeBindMechanism{iface: iface,
			virtIface:           &api.Interface{},
			vmi:                 vmi,
			vif:                 vif,
			domain:              domain,
			podInterfaceName:    podInterfaceName,
			vmNetworkCIDR:       network.Pod.VMNetworkCIDR,
			vmIpv6NetworkCIDR:   "", // TODO add ipv6 cidr to PodNetwork schema
			bridgeInterfaceName: fmt.Sprintf("k6t-%s", podInterfaceName)}, nil
	}
	if iface.Slirp != nil {
		return &SlirpBindMechanism{vmi: vmi, iface: iface, domain: domain}, nil
	}
	if iface.Macvtap != nil {
		mac, err := retrieveMacAddress(iface)
		if err != nil {
			return nil, err
		}
		virtIface := &api.Interface{}
		if mac != nil {
			virtIface.MAC = &api.MAC{MAC: mac.String()}
		}
		return &MacvtapBindMechanism{
			vmi:              vmi,
			iface:            iface,
			virtIface:        virtIface,
			domain:           domain,
			podInterfaceName: podInterfaceName,
		}, nil
	}
	return nil, fmt.Errorf("Not implemented")
}
