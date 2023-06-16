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

package driver

import (
	"fmt"
	"net"
	"os"
	"os/exec"

	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	"kubevirt.io/kubevirt/pkg/util/sysctl"

	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	dhcpserver "kubevirt.io/kubevirt/pkg/network/dhcp/server"
	dhcpserverv6 "kubevirt.io/kubevirt/pkg/network/dhcp/serverv6"
	"kubevirt.io/kubevirt/pkg/network/dns"
	"kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

const (
	randomMacGenerationAttempts = 10
	allowForwarding             = "1"
	LibvirtUserAndGroupId       = "0"
	allowRouteLocalNet          = "1"
)

type IPVersion int

const (
	IPv4 IPVersion = 4
	IPv6 IPVersion = 6
)

type NetworkHandler interface {
	LinkByName(name string) (netlink.Link, error)
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
	ReadIPAddressesFromLink(interfaceName string) (string, string, error)
	RouteList(link netlink.Link, family int) ([]netlink.Route, error)
	AddrDel(link netlink.Link, addr *netlink.Addr) error
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	AddrReplace(link netlink.Link, addr *netlink.Addr) error
	LinkSetDown(link netlink.Link) error
	LinkSetUp(link netlink.Link) error
	LinkSetName(link netlink.Link, name string) error
	LinkAdd(link netlink.Link) error
	LinkDel(link netlink.Link) error
	LinkSetLearningOff(link netlink.Link) error
	ParseAddr(s string) (*netlink.Addr, error)
	LinkSetHardwareAddr(link netlink.Link, hwaddr net.HardwareAddr) error
	LinkSetMaster(link netlink.Link, master *netlink.Bridge) error
	StartDHCP(nic *cache.DHCPConfig, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions) error
	HasIPv4GlobalUnicastAddress(interfaceName string) (bool, error)
	HasIPv6GlobalUnicastAddress(interfaceName string) (bool, error)
	IsIpv4Primary() (bool, error)
	ConfigureIpForwarding(ipVersion IPVersion) error
	ConfigureRouteLocalNet(string) error
	ConfigureIpv4ArpIgnore() error
	ConfigurePingGroupRange() error
	ConfigureUnprivilegedPortStart(string) error
	NftablesNewChain(ipVersion IPVersion, table, chain string) error
	NftablesNewTable(ipVersion IPVersion, name string) error
	NftablesAppendRule(ipVersion IPVersion, table, chain string, rulespec ...string) error
	CheckNftables() error
	GetNFTIPString(ipVersion IPVersion) string
	CreateTapDevice(tapName string, queueNumber uint32, launcherPID int, mtu int, tapOwner string) error
	BindTapDeviceToBridge(tapName string, bridgeName string) error
	DisableTXOffloadChecksum(ifaceName string) error
}

type NetworkUtilsHandler struct{}

func (h *NetworkUtilsHandler) LinkSetHardwareAddr(link netlink.Link, hwaddr net.HardwareAddr) error {
	return netlink.LinkSetHardwareAddr(link, hwaddr)
}

func (h *NetworkUtilsHandler) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}
func (h *NetworkUtilsHandler) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}
func (h *NetworkUtilsHandler) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	return netlink.RouteList(link, family)
}
func (h *NetworkUtilsHandler) AddrReplace(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrReplace(link, addr)
}
func (h *NetworkUtilsHandler) AddrDel(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrDel(link, addr)
}
func (h *NetworkUtilsHandler) LinkSetDown(link netlink.Link) error {
	return netlink.LinkSetDown(link)
}
func (h *NetworkUtilsHandler) LinkSetUp(link netlink.Link) error {
	return netlink.LinkSetUp(link)
}
func (h *NetworkUtilsHandler) LinkSetName(link netlink.Link, name string) error {
	return netlink.LinkSetName(link, name)
}
func (h *NetworkUtilsHandler) LinkAdd(link netlink.Link) error {
	return netlink.LinkAdd(link)
}
func (h *NetworkUtilsHandler) LinkDel(link netlink.Link) error {
	return netlink.LinkDel(link)
}
func (h *NetworkUtilsHandler) LinkSetLearningOff(link netlink.Link) error {
	return netlink.LinkSetLearning(link, false)
}
func (h *NetworkUtilsHandler) ParseAddr(s string) (*netlink.Addr, error) {
	return netlink.ParseAddr(s)
}
func (h *NetworkUtilsHandler) AddrAdd(link netlink.Link, addr *netlink.Addr) error {
	return netlink.AddrAdd(link, addr)
}
func (h *NetworkUtilsHandler) LinkSetMaster(link netlink.Link, master *netlink.Bridge) error {
	return netlink.LinkSetMaster(link, master)
}

func (h *NetworkUtilsHandler) ConfigureIpv4ArpIgnore() error {
	err := sysctl.New().SetSysctl(sysctl.Ipv4ArpIgnoreAll, "1")
	return err
}

func (h *NetworkUtilsHandler) ConfigureIpForwarding(ipVersion IPVersion) error {
	var forwarding string
	if ipVersion == IPv6 {
		forwarding = sysctl.NetIPv6Forwarding
	} else {
		forwarding = sysctl.NetIPv4Forwarding
	}

	err := sysctl.New().SetSysctl(forwarding, allowForwarding)
	return err
}

func (h *NetworkUtilsHandler) ConfigurePingGroupRange() error {
	err := sysctl.New().SetSysctl(sysctl.PingGroupRange, "107 107")
	return err
}

func (h *NetworkUtilsHandler) ConfigureRouteLocalNet(iface string) error {
	routeLocalNetForIface := fmt.Sprintf(sysctl.IPv4RouteLocalNet, iface)
	err := sysctl.New().SetSysctl(routeLocalNetForIface, allowRouteLocalNet)
	return err
}

func (h *NetworkUtilsHandler) ConfigureUnprivilegedPortStart(port string) error {
	err := sysctl.New().SetSysctl(sysctl.UnprivilegedPortStart, port)
	return err
}

func (h *NetworkUtilsHandler) HasIPv4GlobalUnicastAddress(interfaceName string) (bool, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		return false, err
	}
	addrList, err := h.AddrList(link, netlink.FAMILY_V4)
	if err != nil {
		return false, err
	}

	for _, addr := range addrList {
		if addr.IP.IsGlobalUnicast() {
			return true, nil
		}
	}
	return false, nil
}

func (h *NetworkUtilsHandler) HasIPv6GlobalUnicastAddress(interfaceName string) (bool, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		return false, err
	}
	addrList, err := h.AddrList(link, netlink.FAMILY_V6)
	if err != nil {
		return false, err
	}

	for _, addr := range addrList {
		if addr.IP.IsGlobalUnicast() {
			return true, nil
		}
	}
	return false, nil
}

func (h *NetworkUtilsHandler) IsIpv4Primary() (bool, error) {
	podIP, exist := os.LookupEnv("MY_POD_IP")
	if !exist {
		return false, fmt.Errorf("MY_POD_IP doesnt exists")
	}

	return !netutils.IsIPv6String(podIP), nil
}

func (h *NetworkUtilsHandler) NftablesNewTable(ipVersion IPVersion, name string) error {
	// #nosec g204 no risk to use GetNFTIPString as  argument as it returns either "ipv6" or "ip" strings
	output, err := exec.Command("nft", "add", "table", h.GetNFTIPString(ipVersion), name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s, error: %s", string(output), err.Error())
	}

	return nil
}

func (h *NetworkUtilsHandler) NftablesNewChain(ipVersion IPVersion, table, chain string) error {
	// #nosec g204 no risk to use GetNFTIPString as  argument as it returns either "ipv6" or "ip" strings
	output, err := exec.Command("nft", "add", "chain", h.GetNFTIPString(ipVersion), table, chain).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s, error: %s", string(output), err.Error())
	}

	return nil
}

func (h *NetworkUtilsHandler) NftablesAppendRule(ipVersion IPVersion, table, chain string, rulespec ...string) error {
	cmd := append([]string{"add", "rule", h.GetNFTIPString(ipVersion), table, chain}, rulespec...)
	// #nosec No risk for attacket injection. CMD variables are predefined strings
	output, err := exec.Command("nft", cmd...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apped new nfrule error %s", string(output))
	}

	return nil
}

func (h *NetworkUtilsHandler) GetNFTIPString(ipVersion IPVersion) string {
	if ipVersion == IPv6 {
		return "ip6"
	}
	return "ip"
}

func (h *NetworkUtilsHandler) CheckNftables() error {
	output, err := exec.Command("nft", "list", "ruleset").CombinedOutput()
	if err != nil {
		log.Log.V(5).Reason(err).Infof("failed to list nftable ruleset")
		return fmt.Errorf("failed to list nftable ruleset, error: %s", string(output))
	}

	return nil
}

func (h *NetworkUtilsHandler) ReadIPAddressesFromLink(interfaceName string) (string, string, error) {
	link, err := h.LinkByName(interfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", interfaceName)
		return "", "", err
	}

	// get IP address
	addrList, err := h.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a address for interface: %s", interfaceName)
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

func (h *NetworkUtilsHandler) StartDHCP(nic *cache.DHCPConfig, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions) error {
	log.Log.V(4).Infof("StartDHCP network Nic: %+v", nic)
	nameservers, searchDomains, err := converter.GetResolvConfDetailsFromPod()
	if err != nil {
		return fmt.Errorf("Failed to get DNS servers from resolv.conf: %v", err)
	}

	domain := dns.DomainNameWithSubdomain(searchDomains, nic.Subdomain)
	if domain != "" {
		searchDomains = append([]string{domain}, searchDomains...)
	}

	if nic.IP.IPNet != nil {
		// panic in case the DHCP server failed during the vm creation
		// but ignore dhcp errors when the vm is destroyed or shutting down
		go func() {
			if err = DHCPServer(
				nic.MAC,
				nic.IP.IP,
				nic.IP.Mask,
				bridgeInterfaceName,
				nic.AdvertisingIPAddr,
				nic.Gateway,
				nameservers,
				nic.Routes,
				searchDomains,
				nic.Mtu,
				dhcpOptions,
			); err != nil {
				log.Log.Errorf("failed to run DHCP: %v", err)
				panic(err)
			}
		}()
	}

	if nic.IPv6.IPNet != nil {
		go func() {
			if err = DHCPv6Server(
				nic.IPv6.IP,
				bridgeInterfaceName,
			); err != nil {
				log.Log.Reason(err).Error("failed to run DHCPv6")
				panic(err)
			}
		}()
	}

	return nil
}

func (h *NetworkUtilsHandler) CreateTapDevice(tapName string, queueNumber uint32, launcherPID int, mtu int, tapOwner string) error {
	tapDeviceSELinuxCmdExecutor, err := buildTapDeviceMaker(tapName, queueNumber, launcherPID, mtu, tapOwner)
	if err != nil {
		return err
	}
	if err := tapDeviceSELinuxCmdExecutor.Execute(); err != nil {
		return fmt.Errorf("error creating tap device named %s; %v", tapName, err)
	}

	log.Log.Infof("Created tap device: %s in PID: %d", tapName, launcherPID)
	return nil
}

func buildTapDeviceMaker(tapName string, queueNumber uint32, virtLauncherPID int, mtu int, tapOwner string) (*selinux.ContextExecutor, error) {
	createTapDeviceArgs := []string{
		"create-tap",
		"--tap-name", tapName,
		"--uid", tapOwner,
		"--gid", tapOwner,
		"--queue-number", fmt.Sprintf("%d", queueNumber),
		"--mtu", fmt.Sprintf("%d", mtu),
	}
	// #nosec No risk for attacket injection. createTapDeviceArgs includes predefined strings
	cmd := exec.Command("virt-chroot", createTapDeviceArgs...)
	return selinux.NewContextExecutor(virtLauncherPID, cmd)
}

func (h *NetworkUtilsHandler) BindTapDeviceToBridge(tapName string, bridgeName string) error {
	tap, err := netlink.LinkByName(tapName)
	log.Log.V(4).Infof("Looking for tap device: %s", tapName)
	if err != nil {
		return fmt.Errorf("could not find tap device %s; %v", tapName, err)
	}

	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeName,
		},
	}
	if err := netlink.LinkSetMaster(tap, bridge); err != nil {
		return fmt.Errorf("failed to bind tap device %s to bridge %s; %v", tapName, bridgeName, err)
	}

	err = netlink.LinkSetUp(tap)
	if err != nil {
		return fmt.Errorf("failed to set tap device %s up; %v", tapName, err)
	}

	log.Log.Infof("Successfully configured tap device: %s", tapName)
	return nil
}

func (h *NetworkUtilsHandler) DisableTXOffloadChecksum(ifaceName string) error {
	if err := EthtoolTXOff(ifaceName); err != nil {
		log.Log.Reason(err).Errorf("Failed to set tx offload for interface %s off", ifaceName)
		return err
	}

	return nil
}

// Allow mocking for tests
var DHCPServer = dhcpserver.SingleClientDHCPServer
var DHCPv6Server = dhcpserverv6.SingleClientDHCPv6Server
