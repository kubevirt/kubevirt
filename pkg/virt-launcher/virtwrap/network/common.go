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
	"os/exec"

	"io/ioutil"
	"net"
	"os"

	"github.com/coreos/go-iptables/iptables"

	lmf "github.com/subgraph/libmacouflage"
	"github.com/vishvananda/netlink"

	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/dhcp"
)

const randomMacGenerationAttempts = 10

type VIF struct {
	Name         string
	IP           netlink.Addr
	MAC          net.HardwareAddr
	Gateway      net.IP
	Routes       *[]netlink.Route
	Mtu          uint16
	IPAMDisabled bool
}

func (vif VIF) String() string {
	return fmt.Sprintf(
		"VIF: { Name: %s, IP: %s, Mask: %s, MAC: %s, Gateway: %s, MTU: %d, IPAMDisabled: %t}",
		vif.Name,
		vif.IP.IP,
		vif.IP.Mask,
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
	StartDHCP(nic *VIF, serverAddr *netlink.Addr, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions) error
	UseIptables() bool
	IptablesNewChain(table, chain string) error
	IptablesAppendRule(table, chain string, rulespec ...string) error
	NftablesNewChain(table, chain string) error
	NftablesAppendRule(table, chain string, rulespec ...string) error
	NftablesNewTable(table string) error
	NftablesLoad(fnName string) error
}

type NetworkUtilsHandler struct{}

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
func (h *NetworkUtilsHandler) UseIptables() bool {
	iptablesObject, err := iptables.New()
	if err != nil {
		return false
	}

	_, err = iptablesObject.List("nat", "OUTPUT")
	if err != nil {
		return false
	}

	return true
}
func (h *NetworkUtilsHandler) IptablesNewChain(table, chain string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}

	return iptablesObject.NewChain(table, chain)
}
func (h *NetworkUtilsHandler) IptablesAppendRule(table, chain string, rulespec ...string) error {
	iptablesObject, err := iptables.New()
	if err != nil {
		return err
	}

	return iptablesObject.Append(table, chain, rulespec...)
}
func (h *NetworkUtilsHandler) NftablesNewChain(table, chain string) error {
	output, err := exec.Command("nft", "add", "chain", "ip", table, chain).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", string(output))
	}

	return nil
}
func (h *NetworkUtilsHandler) NftablesAppendRule(table, chain string, rulespec ...string) error {
	cmd := append([]string{"add", "rule", "ip", table, chain}, rulespec...)
	output, err := exec.Command("nft", cmd...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apped new nfrule error %s", string(output))
	}

	return nil
}
func (h *NetworkUtilsHandler) NftablesNewTable(table string) error {
	output, err := exec.Command("nft", "add", "table", table).CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create new nftable error %s", string(output))
	}

	return nil
}
func (h *NetworkUtilsHandler) NftablesLoad(fnName string) error {
	output, err := exec.Command("nft", "-f", fmt.Sprintf("/etc/nftables/%s.nft", fnName)).CombinedOutput()
	if err != nil {
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

func (h *NetworkUtilsHandler) StartDHCP(nic *VIF, serverAddr *netlink.Addr, bridgeInterfaceName string, dhcpOptions *v1.DHCPOptions) error {
	log.Log.V(4).Infof("StartDHCP network Nic: %+v", nic)
	nameservers, searchDomains, err := api.GetResolvConfDetailsFromPod()
	if err != nil {
		return fmt.Errorf("Failed to get DNS servers from resolv.conf: %v", err)
	}

	// panic in case the DHCP server failed during the vm creation
	// but ignore dhcp errors when the vm is destroyed or shutting down
	go func() {
		if err = DHCPServer(
			nic.MAC,
			nic.IP.IP,
			nic.IP.Mask,
			bridgeInterfaceName,
			serverAddr.IP,
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

// Allow mocking for tests
var SetupPodNetworkPhase1 = SetupNetworkInterfacesPhase1
var SetupPodNetworkPhase2 = SetupNetworkInterfacesPhase2
var DHCPServer = dhcp.SingleClientDHCPServer

func initHandler() {
	if Handler == nil {
		Handler = &NetworkUtilsHandler{}
	}
}

func writeToCachedFile(inter interface{}, fileName string, uid types.UID, name string) error {
	buf, err := json.MarshalIndent(&inter, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling cached object: %v", err)
	}
	err = ioutil.WriteFile(getInterfaceCacheFile(fileName, uid, name), buf, 0644)
	if err != nil {
		return fmt.Errorf("error writing cached object: %v", err)
	}
	return nil
}

func readFromCachedFile(uid types.UID, name, fileName string, inter interface{}) (bool, error) {
	buf, err := ioutil.ReadFile(getInterfaceCacheFile(fileName, uid, name))
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

func getInterfaceCacheFile(filePath string, uid types.UID, name string) string {
	return fmt.Sprintf(filePath, uid, name)
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
