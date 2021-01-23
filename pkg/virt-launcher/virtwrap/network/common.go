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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/coreos/go-iptables/iptables"
	"github.com/opencontainers/selinux/go-selinux"
	lmf "github.com/subgraph/libmacouflage"
	"github.com/vishvananda/netlink"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	"kubevirt.io/kubevirt/pkg/util/sysctl"

	netutils "k8s.io/utils/net"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	kvselinux "kubevirt.io/kubevirt/pkg/virt-handler/selinux"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/dhcp"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/dhcpv6"
)

const (
	randomMacGenerationAttempts = 10
	tapOwnerUID                 = "0"
	tapOwnerGID                 = "0"
)

type VIF struct {
	Name         string
	IP           netlink.Addr
	IPv6         netlink.Addr
	MAC          net.HardwareAddr
	Gateway      net.IP
	GatewayIpv6  net.IP
	Routes       *[]netlink.Route
	Mtu          uint16
	IPAMDisabled bool
}

type CriticalNetworkError struct {
	Msg string
}

func (e *CriticalNetworkError) Error() string { return e.Msg }

func (vif VIF) String() string {
	return fmt.Sprintf(
		"VIF: { Name: %s, IP: %s, Mask: %s, IPv6: %s, MAC: %s, Gateway: %s, MTU: %d, IPAMDisabled: %t}",
		vif.Name,
		vif.IP.IP,
		vif.IP.Mask,
		vif.IPv6,
		vif.MAC,
		vif.Gateway,
		vif.Mtu,
		vif.IPAMDisabled,
	)
}

type NetworkHandler interface {
	LinkByName(name string) (netlink.Link, error)
	AddrList(link netlink.Link, family int) ([]netlink.Addr, error)
	RouteList(link netlink.Link, family int) ([]netlink.Route, error)
	AddrDel(link netlink.Link, addr *netlink.Addr) error
	AddrAdd(link netlink.Link, addr *netlink.Addr) error
	LinkSetDown(link netlink.Link) error
	LinkSetUp(link netlink.Link) error
	LinkAdd(link netlink.Link) error
	LinkSetLearningOff(link netlink.Link) error
	ParseAddr(s string) (*netlink.Addr, error)
	GetHostAndGwAddressesFromCIDR(s string) (string, string, error)
	SetRandomMac(iface string) (net.HardwareAddr, error)
	GenerateRandomMac() (net.HardwareAddr, error)
	GetMacDetails(iface string) (net.HardwareAddr, error)
	LinkSetMaster(link netlink.Link, master *netlink.Bridge) error
	StartDHCP(nic *VIF, serverAddr net.IP, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions, filterByMAC bool) error
	HasNatIptables(proto iptables.Protocol) bool
	IsIpv6Enabled(interfaceName string) (bool, error)
	IsIpv4Primary() (bool, error)
	ConfigureIpv6Forwarding() error
	IptablesNewChain(proto iptables.Protocol, table, chain string) error
	IptablesAppendRule(proto iptables.Protocol, table, chain string, rulespec ...string) error
	NftablesNewChain(proto iptables.Protocol, table, chain string) error
	NftablesAppendRule(proto iptables.Protocol, table, chain string, rulespec ...string) error
	NftablesLoad(fnName string) error
	GetNFTIPString(proto iptables.Protocol) string
	CreateTapDevice(tapName string, queueNumber uint32, launcherPID int, mtu int) error
	BindTapDeviceToBridge(tapName string, bridgeName string) error
	DisableTXOffloadChecksum(ifaceName string) error
}

type NetworkUtilsHandler struct{}

type tapDeviceMaker struct {
	tapName         string
	queueNumber     uint32
	virtLauncherPID int
	tapMakingCmd    *exec.Cmd
}

var Handler NetworkHandler

func (h *NetworkUtilsHandler) LinkByName(name string) (netlink.Link, error) {
	return netlink.LinkByName(name)
}
func (h *NetworkUtilsHandler) AddrList(link netlink.Link, family int) ([]netlink.Addr, error) {
	return netlink.AddrList(link, family)
}
func (h *NetworkUtilsHandler) RouteList(link netlink.Link, family int) ([]netlink.Route, error) {
	return netlink.RouteList(link, family)
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
func (h *NetworkUtilsHandler) LinkAdd(link netlink.Link) error {
	return netlink.LinkAdd(link)
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
func (h *NetworkUtilsHandler) HasNatIptables(proto iptables.Protocol) bool {
	iptablesObject, err := iptables.NewWithProtocol(proto)
	if err != nil {
		log.Log.V(5).Reason(err).Infof("No iptables")
		return false
	}

	_, err = iptablesObject.List("nat", "OUTPUT")
	if err != nil {
		log.Log.V(5).Reason(err).Infof("No nat iptables")
		return false
	}

	return true
}

func (h *NetworkUtilsHandler) ConfigureIpv6Forwarding() error {
	err := sysctl.New().SetSysctl(sysctl.NetIPv6Forwarding, 1)
	return err
}

func (h *NetworkUtilsHandler) IsIpv6Enabled(interfaceName string) (bool, error) {
	link, err := Handler.LinkByName(interfaceName)
	if err != nil {
		return false, err
	}
	addrList, err := Handler.AddrList(link, netlink.FAMILY_V6)
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

func (h *NetworkUtilsHandler) IptablesNewChain(proto iptables.Protocol, table, chain string) error {
	iptablesObject, err := iptables.NewWithProtocol(proto)
	if err != nil {
		return err
	}

	return iptablesObject.NewChain(table, chain)
}

func (h *NetworkUtilsHandler) IptablesAppendRule(proto iptables.Protocol, table, chain string, rulespec ...string) error {
	iptablesObject, err := iptables.NewWithProtocol(proto)
	if err != nil {
		return err
	}

	return iptablesObject.Append(table, chain, rulespec...)
}

func (h *NetworkUtilsHandler) NftablesNewChain(proto iptables.Protocol, table, chain string) error {
	// #nosec g204 no risk to use GetNFTIPString as  argument as it returns either "ipv6" or "ip" strings
	output, err := exec.Command("nft", "add", "chain", Handler.GetNFTIPString(proto), table, chain).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	return nil
}

func (h *NetworkUtilsHandler) NftablesAppendRule(proto iptables.Protocol, table, chain string, rulespec ...string) error {
	cmd := append([]string{"add", "rule", Handler.GetNFTIPString(proto), table, chain}, rulespec...)
	// #nosec No risk for attacket injection. CMD variables are predefined strings
	output, err := exec.Command("nft", cmd...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apped new nfrule error %s", string(output))
	}

	return nil
}

func (h *NetworkUtilsHandler) GetNFTIPString(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv6 {
		return "ip6"
	}
	return "ip"
}

func (h *NetworkUtilsHandler) NftablesLoad(fnName string) error {
	// #nosec g204 no risk to use Sprintf as  argument as it uses two static strings (fname limited to ipv4-nat or ipv6-nat)
	output, err := exec.Command("nft", "-f", fmt.Sprintf("/etc/nftables/%s.nft", fnName)).CombinedOutput()
	if err != nil {
		log.Log.V(5).Reason(err).Infof("failed to load nftable %s", fnName)
		return fmt.Errorf("failed to load nftable %s error %s", fnName, string(output))
	}

	return nil
}
func (h *NetworkUtilsHandler) GetHostAndGwAddressesFromCIDR(s string) (string, string, error) {
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return "", "", err
	}

	subnet, _ := ipnet.Mask.Size()
	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, fmt.Sprintf("%s/%d", ip.String(), subnet))

		if len(ips) == 4 {
			// remove network address and broadcast address
			return ips[1], ips[2], nil
		}
	}

	return "", "", fmt.Errorf("less than 4 addresses on network")
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// GetMacDetails from an interface
func (h *NetworkUtilsHandler) GetMacDetails(iface string) (net.HardwareAddr, error) {
	currentMac, err := lmf.GetCurrentMac(iface)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get mac information for interface: %s", iface)
		return nil, err
	}
	return currentMac, nil
}

// SetRandomMac changes the MAC address for a given interface to a randomly generated, preserving the vendor prefix
func (h *NetworkUtilsHandler) SetRandomMac(iface string) (net.HardwareAddr, error) {
	var mac net.HardwareAddr

	currentMac, err := Handler.GetMacDetails(iface)
	if err != nil {
		return nil, err
	}

	changed := false

	for i := 0; i < randomMacGenerationAttempts; i++ {
		changed, err = lmf.SpoofMacSameVendor(iface, false)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to spoof MAC for an interface: %s", iface)
			return nil, err
		}

		if changed {
			mac, err = Handler.GetMacDetails(iface)
			if err != nil {
				return nil, err
			}
			log.Log.Infof("updated MAC for %s interface: old: %s -> new: %s", iface, currentMac, mac)
			break
		}
	}
	if !changed {
		err := fmt.Errorf("failed to spoof MAC for an interface %s after %d attempts", iface, randomMacGenerationAttempts)
		log.Log.Reason(err)
		return nil, err
	}
	return currentMac, nil
}

func (h *NetworkUtilsHandler) StartDHCP(nic *VIF, serverAddr net.IP, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions, filterByMAC bool) error {
	log.Log.V(4).Infof("StartDHCP network Nic: %+v", nic)
	nameservers, searchDomains, err := converter.GetResolvConfDetailsFromPod()
	if err != nil {
		return fmt.Errorf("Failed to get DNS servers from resolv.conf: %v", err)
	}

	// panic in case the DHCP server failed during the vm creation
	// but ignore dhcp errors when the vm is destroyed or shutting down
	go func() {
		if err = DHCPServer(
			nic.MAC,
			filterByMAC,
			nic.IP.IP,
			nic.IP.Mask,
			bridgeInterfaceName,
			serverAddr,
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

// Generate a random mac for interface
// Avoid MAC address starting with reserved value 0xFE (https://github.com/kubevirt/kubevirt/issues/1494)
func (h *NetworkUtilsHandler) GenerateRandomMac() (net.HardwareAddr, error) {
	prefix := []byte{0x02, 0x00, 0x00} // local unicast prefix
	suffix := make([]byte, 3)
	_, err := rand.Read(suffix)
	if err != nil {
		return nil, err
	}
	return net.HardwareAddr(append(prefix, suffix...)), nil
}

func (h *NetworkUtilsHandler) CreateTapDevice(tapName string, queueNumber uint32, launcherPID int, mtu int) error {
	tapDeviceMaker, err := buildTapDeviceMaker(tapName, queueNumber, launcherPID, mtu)
	if err != nil {
		return fmt.Errorf("error creating tap device named %s; %v", tapName, err)
	}

	if err := tapDeviceMaker.makeTapDevice(); err != nil {
		return fmt.Errorf("error creating tap device named %s; %v", tapName, err)
	}

	log.Log.Infof("Created tap device: %s in PID: %d", tapName, launcherPID)
	return nil
}

func buildTapDeviceMaker(tapName string, queueNumber uint32, virtLauncherPID int, mtu int) (*tapDeviceMaker, error) {
	createTapDeviceArgs := []string{
		"create-tap",
		"--tap-name", tapName,
		"--uid", tapOwnerUID,
		"--gid", tapOwnerGID,
		"--queue-number", fmt.Sprintf("%d", queueNumber),
		"--mtu", fmt.Sprintf("%d", mtu),
	}
	// #nosec No risk for attacket injection. createTapDeviceArgs includes predefined strings
	cmd := exec.Command("virt-chroot", createTapDeviceArgs...)

	return &tapDeviceMaker{
		tapMakingCmd:    cmd,
		tapName:         tapName,
		queueNumber:     queueNumber,
		virtLauncherPID: virtLauncherPID,
	}, nil
}

func (tmc *tapDeviceMaker) makeTapDevice() error {
	if isSELinuxEnabled() {
		defer func() {
			_ = resetVirtHandlerSELinuxContext()
		}()

		if err := setVirtLauncherSELinuxContext(tmc.virtLauncherPID); err != nil {
			return err
		}
	}

	preventFDLeakOntoChild()
	if err := tmc.tapMakingCmd.Run(); err != nil {
		return fmt.Errorf("failed to create tap device %s: %v", tmc.tapName, err)
	}
	return nil
}

func isSELinuxEnabled() bool {
	_, selinuxEnabled, err := kvselinux.NewSELinux()
	return err == nil && selinuxEnabled
}

func setVirtLauncherSELinuxContext(virtLauncherPID int) error {
	virtLauncherSELinuxLabel, err := getProcessCurrentSELinuxLabel(virtLauncherPID)
	if err != nil {
		return fmt.Errorf("error reading virt-launcher %d selinux label. Reason: %v", virtLauncherPID, err)
	}

	runtime.LockOSThread()
	if err := selinux.SetExecLabel(virtLauncherSELinuxLabel); err != nil {
		return fmt.Errorf("failed to switch selinux context to %s. Reason: %v", virtLauncherSELinuxLabel, err)
	}
	return nil
}

func resetVirtHandlerSELinuxContext() error {
	virtHandlerSELinuxLabel := "system_u:system_r:spc_t:s0"
	err := selinux.SetExecLabel(virtHandlerSELinuxLabel)
	runtime.UnlockOSThread()
	return err
}

func getProcessCurrentSELinuxLabel(pid int) (string, error) {
	launcherSELinuxLabel, err := selinux.FileLabel(fmt.Sprintf("/proc/%d/attr/current", pid))
	if err != nil {
		return "", fmt.Errorf("could not retrieve pid %d selinux label: %v", pid, err)
	}
	return launcherSELinuxLabel, nil
}

func preventFDLeakOntoChild() {
	// we want to share the parent process std{in|out|err}
	for fd := 3; fd < 256; fd++ {
		syscall.CloseOnExec(fd)
	}
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
	if err := dhcp.EthtoolTXOff(ifaceName); err != nil {
		log.Log.Reason(err).Errorf("Failed to set tx offload for interface %s off", ifaceName)
		return err
	}

	return nil
}

// Allow mocking for tests
var DHCPServer = dhcp.SingleClientDHCPServer
var DHCPv6Server = dhcpv6.SingleClientDHCPv6Server

func initHandler() {
	if Handler == nil {
		Handler = &NetworkUtilsHandler{}
	}
}

func writeToCachedFile(inter interface{}, fileName, pid, name string) error {
	buf, err := json.MarshalIndent(&inter, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cached object: %v", err)
	}

	fileName = getInterfaceCacheFile(fileName, pid, name)
	err = ioutil.WriteFile(fileName, buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing cached object: %v", err)
	}
	return nil
}

func readFromCachedFile(pid, name, fileName string, inter interface{}) (bool, error) {
	buf, err := ioutil.ReadFile(getInterfaceCacheFile(fileName, pid, name))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	err = json.Unmarshal(buf, &inter)
	if err != nil {
		return false, fmt.Errorf("error unmarshaling cached object: %v", err)
	}
	return true, nil
}

func getInterfaceCacheFile(filePath, pid, name string) string {
	return fmt.Sprintf(filePath, pid, name)
}

// filter out irrelevant routes
func filterPodNetworkRoutes(routes []netlink.Route, nic *VIF) (filteredRoutes []netlink.Route) {
	for _, route := range routes {
		// don't create empty static routes
		if route.Dst == nil && route.Src.Equal(nil) && route.Gw.Equal(nil) {
			continue
		}

		// don't create static route for src == nic
		if route.Src != nil && route.Src.Equal(nic.IP.IP) {
			continue
		}

		filteredRoutes = append(filteredRoutes, route)
	}
	return
}

// only used by unit test suite
func setInterfaceCacheFile(path string) {
	interfaceCacheFile = path
}

func setVifCacheFile(path string) {
	vifCacheFile = path
}
