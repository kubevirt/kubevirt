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

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"

	netutils "k8s.io/utils/net"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network/cache"
)

var bridgeFakeIP = "169.254.75.1%d/32"

type BindMechanism interface {
	discoverPodNetworkInterface() error
	preparePodNetworkInterfaces() error

	loadCachedInterface() error
	setCachedInterface() error
	wasCachedInterfaceLoaded() bool

	// virt-handler that executes phase1 of network configuration needs to
	// pass details about discovered networking port into phase2 that is
	// executed by virt-launcher. Virt-launcher cannot discover some of
	// these details itself because at this point phase1 is complete and
	// ports are rewired, meaning, routes and IP addresses configured by
	// CNI plugin may be gone. For this matter, we use a cached VIF file to
	// pass discovered information between phases.
	loadCachedVIF(pid string) error
	setCachedVIF(pid string) error

	// The following entry points require domain initialized for the
	// binding and can be used in phase2 only.
	decorateConfig() error
	startDHCP() error
}

type podNIC struct {
	vmi              *v1.VirtualMachineInstance
	podInterfaceName string
	launcherPID      *int
	iface            *v1.Interface
	network          *v1.Network
	handler          NetworkHandler
	cacheFactory     cache.InterfaceCacheFactory
}

func newPodNIC(vmi *v1.VirtualMachineInstance, network *v1.Network, handler NetworkHandler, cacheFactory cache.InterfaceCacheFactory, launcherPID *int) (*podNIC, error) {
	if network.Pod == nil && network.Multus == nil {
		return nil, fmt.Errorf("Network not implemented")
	}

	correspondingNetworkIface := findInterfaceByNetworkName(vmi, network)
	if correspondingNetworkIface == nil {
		return nil, fmt.Errorf("no iface matching with network %s", network.Name)
	}

	podInterfaceName, err := composePodInterfaceName(vmi, network)
	if err != nil {
		return nil, err
	}
	return &podNIC{
		cacheFactory:     cacheFactory,
		handler:          handler,
		vmi:              vmi,
		network:          network,
		podInterfaceName: podInterfaceName,
		iface:            correspondingNetworkIface,
		launcherPID:      launcherPID,
	}, nil
}

func composePodInterfaceName(vmi *v1.VirtualMachineInstance, network *v1.Network) (string, error) {
	if isSecondaryMultusNetwork(*network) {
		multusIndex := findMultusIndex(vmi, network)
		if multusIndex == -1 {
			return "", fmt.Errorf("Network name %s not found", network.Name)
		}
		return fmt.Sprintf("net%d", multusIndex), nil
	}
	return primaryPodInterfaceName, nil
}

func findInterfaceByNetworkName(vmi *v1.VirtualMachineInstance, network *v1.Network) *v1.Interface {
	for i, iface := range vmi.Spec.Domain.Devices.Interfaces {
		if iface.Name == network.Name {
			return &vmi.Spec.Domain.Devices.Interfaces[i]
		}
	}
	return nil
}

func findMultusIndex(vmi *v1.VirtualMachineInstance, networkToFind *v1.Network) int {
	idxMultus := 0
	for _, network := range vmi.Spec.Networks {
		if isSecondaryMultusNetwork(network) {
			// multus pod interfaces start from 1
			idxMultus++
			if network.Name == networkToFind.Name {
				return idxMultus
			}
		}
	}
	return -1
}

func isSecondaryMultusNetwork(net v1.Network) bool {
	return net.Multus != nil && !net.Multus.Default
}

var vifCacheFile = "/proc/%s/root/var/run/kubevirt-private/vif-cache-%s.json"

func setVifCacheFile(path string) {
	vifCacheFile = path
}

func getVifFilePath(pid, name string) string {
	return fmt.Sprintf(vifCacheFile, pid, name)
}

func writeVifFile(buf []byte, pid, name string) error {
	err := ioutil.WriteFile(getVifFilePath(pid, name), buf, 0600)
	if err != nil {
		return fmt.Errorf("error writing vif object: %v", err)
	}
	return nil
}

func (l *podNIC) setPodInterfaceCache() error {
	ifCache := &cache.PodCacheInterface{Iface: l.iface}

	ipv4, ipv6, err := l.readIPAddressesFromLink()
	if err != nil {
		return err
	}

	switch {
	case ipv4 != "" && ipv6 != "":
		ifCache.PodIPs, err = l.sortIPsBasedOnPrimaryIP(ipv4, ipv6)
		if err != nil {
			return err
		}
	case ipv4 != "":
		ifCache.PodIPs = []string{ipv4}
	case ipv6 != "":
		ifCache.PodIPs = []string{ipv6}
	default:
		return nil
	}

	ifCache.PodIP = ifCache.PodIPs[0]
	err = l.cacheFactory.CacheForVMI(l.vmi).Write(l.iface.Name, ifCache)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to write pod Interface to ifCache, %s", err.Error())
		return err
	}

	return nil
}

func (l *podNIC) readIPAddressesFromLink() (string, string, error) {
	link, err := l.handler.LinkByName(l.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", l.podInterfaceName)
		return "", "", err
	}

	// get IP address
	addrList, err := l.handler.AddrList(link, netlink.FAMILY_ALL)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a address for interface: %s", l.podInterfaceName)
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
func (l *podNIC) sortIPsBasedOnPrimaryIP(ipv4, ipv6 string) ([]string, error) {
	ipv4Primary, err := l.handler.IsIpv4Primary()
	if err != nil {
		return nil, err
	}

	if ipv4Primary {
		return []string{ipv4, ipv6}, nil
	}

	return []string{ipv6, ipv4}, nil
}

func (l *podNIC) PlugPhase1() error {

	// There is nothing to plug for SR-IOV devices
	if l.iface.SRIOV != nil {
		return nil
	}

	bindMechanism, err := l.getPhase1Binding()
	if err != nil {
		return err
	}

	if err := bindMechanism.loadCachedInterface(); err != nil {
		return err
	}

	doesExist := bindMechanism.wasCachedInterfaceLoaded()
	// ignore the bindMechanism.loadCachedInterface for slirp and set the Pod interface cache
	if !doesExist || l.iface.Slirp != nil {
		err := l.setPodInterfaceCache()
		if err != nil {
			return err
		}
	}
	if !doesExist {
		err = bindMechanism.discoverPodNetworkInterface()
		if err != nil {
			return err
		}

		if err := bindMechanism.preparePodNetworkInterfaces(); err != nil {
			log.Log.Reason(err).Error("failed to prepare pod networking")
			return createCriticalNetworkError(err)
		}

		if err := bindMechanism.setCachedVIF(getPIDString(l.launcherPID)); err != nil {
			log.Log.Reason(err).Error("failed to save vif configuration")
			return createCriticalNetworkError(err)
		}

		if err := bindMechanism.setCachedInterface(); err != nil {
			log.Log.Reason(err).Error("failed to save interface configuration")
			return createCriticalNetworkError(err)
		}
	}

	return nil
}

func createCriticalNetworkError(err error) *CriticalNetworkError {
	return &CriticalNetworkError{fmt.Sprintf("Critical network error: %v", err)}
}

func ensureDHCP(bindMechanism BindMechanism, podInterfaceName string) error {
	dhcpStartedFile := fmt.Sprintf("/var/run/kubevirt-private/dhcp_started-%s", podInterfaceName)
	_, err := os.Stat(dhcpStartedFile)
	if os.IsNotExist(err) {
		if err := bindMechanism.startDHCP(); err != nil {
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

func (l *podNIC) PlugPhase2(domain *api.Domain) error {
	precond.MustNotBeNil(domain)

	// There is nothing to plug for SR-IOV devices
	if l.iface.SRIOV != nil {
		return nil
	}

	bindMechanism, err := l.getPhase2Binding(domain)
	if err != nil {
		return err
	}

	if err := bindMechanism.loadCachedInterface(); err != nil {
		log.Log.Reason(err).Critical("failed to load cached interface configuration")
	}
	if !bindMechanism.wasCachedInterfaceLoaded() {
		log.Log.Reason(err).Critical("cached interface configuration doesn't exist")
	}

	pid := "self"
	if err = bindMechanism.loadCachedVIF(pid); err != nil {
		log.Log.Reason(err).Critical("failed to load cached vif configuration")
	}

	err = bindMechanism.decorateConfig()
	if err != nil {
		log.Log.Reason(err).Critical("failed to create libvirt configuration")
	}

	if err := ensureDHCP(bindMechanism, l.podInterfaceName); err != nil {
		log.Log.Reason(err).Criticalf("failed to ensure dhcp service running for %s: %s", l.podInterfaceName, err)
		panic(err)
	}

	return nil
}

func (l *podNIC) getPhase1Binding() (BindMechanism, error) {
	return l.getPhase2Binding(nil)
}

func (l *podNIC) getPhase2Binding(domain *api.Domain) (BindMechanism, error) {
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

	if l.iface.Bridge != nil {
		mac, err := retrieveMacAddress(l.iface)
		if err != nil {
			return nil, err
		}
		vif := &VIF{Name: l.podInterfaceName}
		if mac != nil {
			vif.MAC = *mac
		}

		return &BridgeBindMechanism{iface: l.iface,
			vmi:                 l.vmi,
			vif:                 vif,
			domain:              domain,
			podInterfaceName:    l.podInterfaceName,
			bridgeInterfaceName: fmt.Sprintf("k6t-%s", l.podInterfaceName),
			cacheFactory:        l.cacheFactory,
			launcherPID:         l.launcherPID,
			queueCount:          calculateNetworkQueues(l.vmi),
			handler:             l.handler,
		}, nil
	}
	if l.iface.Masquerade != nil {
		mac, err := retrieveMacAddress(l.iface)
		if err != nil {
			return nil, err
		}
		vif := &VIF{Name: l.podInterfaceName}
		if mac != nil {
			vif.MAC = *mac
		}

		return &MasqueradeBindMechanism{iface: l.iface,
			vmi:                 l.vmi,
			vif:                 vif,
			domain:              domain,
			podInterfaceName:    l.podInterfaceName,
			vmNetworkCIDR:       l.network.Pod.VMNetworkCIDR,
			vmIPv6NetworkCIDR:   l.network.Pod.VMIPv6NetworkCIDR,
			bridgeInterfaceName: fmt.Sprintf("k6t-%s", l.podInterfaceName),
			cacheFactory:        l.cacheFactory,
			launcherPID:         l.launcherPID,
			queueCount:          calculateNetworkQueues(l.vmi),
			handler:             l.handler,
		}, nil
	}
	if l.iface.Slirp != nil {
		return &SlirpBindMechanism{iface: l.iface, domain: domain}, nil
	}
	if l.iface.Macvtap != nil {
		mac, err := retrieveMacAddress(l.iface)
		if err != nil {
			return nil, err
		}

		return &MacvtapBindMechanism{
			vmi:              l.vmi,
			iface:            l.iface,
			domain:           domain,
			podInterfaceName: l.podInterfaceName,
			mac:              mac,
			cacheFactory:     l.cacheFactory,
			launcherPID:      l.launcherPID,
			handler:          l.handler,
		}, nil
	}
	return nil, fmt.Errorf("Not implemented")
}

type BridgeBindMechanism struct {
	vmi                 *v1.VirtualMachineInstance
	vif                 *VIF
	iface               *v1.Interface
	virtIface           *api.Interface
	podNicLink          netlink.Link
	domain              *api.Domain
	podInterfaceName    string
	bridgeInterfaceName string
	arpIgnore           bool
	cacheFactory        cache.InterfaceCacheFactory
	launcherPID         *int
	queueCount          uint32
	handler             NetworkHandler
}

func (b *BridgeBindMechanism) discoverPodNetworkInterface() error {
	link, err := b.handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return err
	}
	b.podNicLink = link

	// get IP address
	addrList, err := b.handler.AddrList(b.podNicLink, netlink.FAMILY_V4)
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
		b.vif.MAC = b.podNicLink.Attrs().HardwareAddr
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

func (b *BridgeBindMechanism) getFakeBridgeIP() (string, error) {
	ifaces := b.vmi.Spec.Domain.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Name == b.iface.Name {
			return fmt.Sprintf(bridgeFakeIP, i), nil
		}
	}
	return "", fmt.Errorf("Failed to generate bridge fake address for interface %s", b.iface.Name)
}

func (b *BridgeBindMechanism) startDHCP() error {
	if !b.vif.IPAMDisabled {
		addr, err := b.getFakeBridgeIP()
		if err != nil {
			return err
		}
		fakeServerAddr, err := netlink.ParseAddr(addr)
		if err != nil {
			return fmt.Errorf("failed to parse address while starting DHCP server: %s", addr)
		}
		log.Log.Object(b.vmi).Infof("bridge pod interface: %+v %+v", b.vif, b)
		return b.handler.StartDHCP(b.vif, fakeServerAddr.IP, b.bridgeInterfaceName, b.iface.DHCPOptions, true)
	}
	return nil
}

func (b *BridgeBindMechanism) preparePodNetworkInterfaces() error {
	// Set interface link to down to change its MAC address
	if err := b.handler.LinkSetDown(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", b.podInterfaceName)
		return err
	}

	tapDeviceName := generateTapDeviceName(b.podInterfaceName)

	if !b.vif.IPAMDisabled {
		// Remove IP from POD interface
		err := b.handler.AddrDel(b.podNicLink, &b.vif.IP)

		if err != nil {
			log.Log.Reason(err).Errorf("failed to delete address for interface: %s", b.podInterfaceName)
			return err
		}

		if err := b.switchPodInterfaceWithDummy(); err != nil {
			log.Log.Reason(err).Error("failed to switch pod interface with a dummy")
			return err
		}
	}

	if _, err := b.handler.SetRandomMac(b.podInterfaceName); err != nil {
		return err
	}

	if err := b.createBridge(); err != nil {
		return err
	}

	err := createAndBindTapToBridge(b.handler, tapDeviceName, b.bridgeInterfaceName, b.queueCount, *b.launcherPID, int(b.vif.Mtu), libvirtUserAndGroupId)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", tapDeviceName)
		return err
	}

	if b.arpIgnore {
		if err := b.handler.ConfigureIpv4ArpIgnore(); err != nil {
			log.Log.Reason(err).Errorf("failed to set arp_ignore=1 on interface %s", b.bridgeInterfaceName)
			return err
		}
	}

	if err := b.handler.LinkSetUp(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.podInterfaceName)
		return err
	}

	if err := b.handler.LinkSetLearningOff(b.podNicLink); err != nil {
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

func (b *BridgeBindMechanism) decorateConfig() error {
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.iface.Name {
			ifaces[i].MTU = b.virtIface.MTU
			ifaces[i].MAC = &api.MAC{MAC: b.vif.MAC.String()}
			ifaces[i].Target = b.virtIface.Target
			break
		}
	}
	return nil
}

func getPIDString(pid *int) string {
	if pid != nil {
		return fmt.Sprintf("%d", *pid)
	}
	return "self"
}

func (b *BridgeBindMechanism) loadCachedInterface() error {
	ifaceConfig, err := b.cacheFactory.CacheForPID(getPIDString(b.launcherPID)).Read(b.iface.Name)

	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	b.virtIface = ifaceConfig
	return nil
}

func (b *BridgeBindMechanism) setCachedInterface() error {
	return b.cacheFactory.CacheForPID(getPIDString(b.launcherPID)).Write(b.iface.Name, b.virtIface)
}

func (b *BridgeBindMechanism) wasCachedInterfaceLoaded() bool {
	return b.virtIface != nil
}

func (b *BridgeBindMechanism) loadCachedVIF(pid string) error {
	buf, err := ioutil.ReadFile(getVifFilePath(pid, b.iface.Name))
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &b.vif)
	if err != nil {
		return err
	}
	b.vif.Gateway = b.vif.Gateway.To4()
	return nil
}

func (b *BridgeBindMechanism) setCachedVIF(pid string) error {
	return setCachedVIF(*b.vif, pid, b.iface.Name)
}

func (b *BridgeBindMechanism) setInterfaceRoutes() error {
	routes, err := b.handler.RouteList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", b.podInterfaceName)
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("No gateway address found in routes for %s", b.podInterfaceName)
	}
	b.vif.Gateway = routes[0].Gw
	if len(routes) > 1 {
		dhcpRoutes := filterPodNetworkRoutes(routes, b.vif)
		b.vif.Routes = &dhcpRoutes
	}
	return nil
}

func (b *BridgeBindMechanism) createBridge() error {
	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: b.bridgeInterfaceName,
		},
	}
	err := b.handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}

	err = b.handler.LinkSetMaster(b.podNicLink, bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to connect interface %s to bridge %s", b.podInterfaceName, bridge.Name)
		return err
	}

	err = b.handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	// set fake ip on a bridge
	addr, err := b.getFakeBridgeIP()
	if err != nil {
		return err
	}
	fakeaddr, err := b.handler.ParseAddr(addr)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	if err := b.handler.AddrAdd(bridge, fakeaddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}

	if err = b.handler.DisableTXOffloadChecksum(b.bridgeInterfaceName); err != nil {
		log.Log.Reason(err).Error("failed to disable TX offload checksum on bridge interface")
		return err
	}

	return nil
}

func (b *BridgeBindMechanism) switchPodInterfaceWithDummy() error {
	originalPodInterfaceName := b.podInterfaceName
	newPodInterfaceName := fmt.Sprintf("%s-nic", originalPodInterfaceName)
	dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: originalPodInterfaceName}}

	// Set arp_ignore=1 on the bridge interface to avoid
	// the interface being seen by Duplicate Address Detection (DAD).
	// Without this, some VMs will lose their ip address after a few
	// minutes.
	b.arpIgnore = true

	// Rename pod interface to free the original name for a new dummy interface
	err := b.handler.LinkSetName(b.podNicLink, newPodInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to rename interface : %s", b.podInterfaceName)
		return err
	}

	b.podInterfaceName = newPodInterfaceName
	b.podNicLink, err = b.handler.LinkByName(newPodInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return err
	}

	// Create a dummy interface named after the original interface
	err = b.handler.LinkAdd(dummy)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create dummy interface : %s", originalPodInterfaceName)
		return err
	}

	// Replace original pod interface IP address to the dummy
	// Since the dummy is not connected to anything, it should not affect networking
	// Replace will add if ip doesn't exist or modify the ip
	err = b.handler.AddrReplace(dummy, &b.vif.IP)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to replace original IP address to dummy interface: %s", originalPodInterfaceName)
		return err
	}

	return nil
}

type MasqueradeBindMechanism struct {
	vmi                 *v1.VirtualMachineInstance
	vif                 *VIF
	iface               *v1.Interface
	virtIface           *api.Interface
	podNicLink          netlink.Link
	domain              *api.Domain
	podInterfaceName    string
	bridgeInterfaceName string
	vmNetworkCIDR       string
	vmIPv6NetworkCIDR   string
	gatewayAddr         *netlink.Addr
	gatewayIpv6Addr     *netlink.Addr
	cacheFactory        cache.InterfaceCacheFactory
	launcherPID         *int
	queueCount          uint32
	handler             NetworkHandler
}

func (b *MasqueradeBindMechanism) discoverPodNetworkInterface() error {
	link, err := b.handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return err
	}
	b.podNicLink = link

	if b.podNicLink.Attrs().MTU < 0 || b.podNicLink.Attrs().MTU > 65535 {
		return fmt.Errorf("MTU value out of range ")
	}

	// Get interface MTU
	b.vif.Mtu = uint16(b.podNicLink.Attrs().MTU)

	err = configureVifV4Addresses(b, err)
	if err != nil {
		return err
	}

	ipv6Enabled, err := b.handler.IsIpv6Enabled(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", b.podInterfaceName)
		return err
	}
	if ipv6Enabled {
		err = configureVifV6Addresses(b, err)
		if err != nil {
			return err
		}
	}
	return nil
}

func configureVifV4Addresses(b *MasqueradeBindMechanism, err error) error {
	if b.vmNetworkCIDR == "" {
		b.vmNetworkCIDR = api.DefaultVMCIDR
	}

	defaultGateway, vm, err := b.handler.GetHostAndGwAddressesFromCIDR(b.vmNetworkCIDR)
	if err != nil {
		log.Log.Errorf("failed to get gw and vm available addresses from CIDR %s", b.vmNetworkCIDR)
		return err
	}

	gatewayAddr, err := b.handler.ParseAddr(defaultGateway)
	if err != nil {
		return fmt.Errorf("failed to parse gateway ip address %s", defaultGateway)
	}
	b.vif.Gateway = gatewayAddr.IP.To4()
	b.gatewayAddr = gatewayAddr

	vmAddr, err := b.handler.ParseAddr(vm)
	if err != nil {
		return fmt.Errorf("failed to parse vm ip address %s", vm)
	}
	b.vif.IP = *vmAddr
	return nil
}

func configureVifV6Addresses(b *MasqueradeBindMechanism, err error) error {
	if b.vmIPv6NetworkCIDR == "" {
		b.vmIPv6NetworkCIDR = api.DefaultVMIpv6CIDR
	}

	defaultGatewayIpv6, vmIpv6, err := b.handler.GetHostAndGwAddressesFromCIDR(b.vmIPv6NetworkCIDR)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get gw and vm available ipv6 addresses from CIDR %s", b.vmIPv6NetworkCIDR)
		return err
	}

	gatewayIpv6Addr, err := b.handler.ParseAddr(defaultGatewayIpv6)
	if err != nil {
		return fmt.Errorf("failed to parse gateway ipv6 address %s err %v", gatewayIpv6Addr, err)
	}
	b.vif.GatewayIpv6 = gatewayIpv6Addr.IP.To16()
	b.gatewayIpv6Addr = gatewayIpv6Addr

	vmAddr, err := b.handler.ParseAddr(vmIpv6)
	if err != nil {
		return fmt.Errorf("failed to parse vm ipv6 address %s err %v", vmIpv6, err)
	}
	b.vif.IPv6 = *vmAddr
	return nil
}

func (b *MasqueradeBindMechanism) startDHCP() error {
	return b.handler.StartDHCP(b.vif, b.vif.Gateway, b.bridgeInterfaceName, b.iface.DHCPOptions, false)
}

func (b *MasqueradeBindMechanism) preparePodNetworkInterfaces() error {
	// Create an master bridge interface
	bridgeNicName := fmt.Sprintf("%s-nic", b.bridgeInterfaceName)
	bridgeNic := &netlink.Dummy{
		LinkAttrs: netlink.LinkAttrs{
			Name: bridgeNicName,
			MTU:  int(b.vif.Mtu),
		},
	}
	err := b.handler.LinkAdd(bridgeNic)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create an interface: %s", bridgeNic.Name)
		return err
	}

	err = b.handler.LinkSetUp(bridgeNic)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", bridgeNic.Name)
		return err
	}

	if err := b.createBridge(); err != nil {
		return err
	}

	tapDeviceName := generateTapDeviceName(b.podInterfaceName)
	err = createAndBindTapToBridge(b.handler, tapDeviceName, b.bridgeInterfaceName, b.queueCount, *b.launcherPID, int(b.vif.Mtu), libvirtUserAndGroupId)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", tapDeviceName)
		return err
	}

	err = b.createNatRules(iptables.ProtocolIPv4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create ipv4 nat rules for vm error: %v", err)
		return err
	}

	ipv6Enabled, err := b.handler.IsIpv6Enabled(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", b.podInterfaceName)
		return err
	}
	if ipv6Enabled {
		err = b.createNatRules(iptables.ProtocolIPv6)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to create ipv6 nat rules for vm error: %v", err)
			return err
		}
	}

	b.virtIface = &api.Interface{
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  tapDeviceName,
			Managed: "no",
		},
	}
	if b.vif.MAC != nil {
		b.virtIface.MAC = &api.MAC{MAC: b.vif.MAC.String()}
	}

	return nil
}

func (b *MasqueradeBindMechanism) decorateConfig() error {
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

func (b *MasqueradeBindMechanism) loadCachedInterface() error {
	ifaceConfig, err := b.cacheFactory.CacheForPID(getPIDString(b.launcherPID)).Read(b.iface.Name)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	b.virtIface = ifaceConfig
	return nil
}

func (b *MasqueradeBindMechanism) setCachedInterface() error {
	return b.cacheFactory.CacheForPID(getPIDString(b.launcherPID)).Write(b.iface.Name, b.virtIface)
}

func (b *MasqueradeBindMechanism) wasCachedInterfaceLoaded() bool {
	return b.virtIface != nil
}

func (b *MasqueradeBindMechanism) loadCachedVIF(pid string) error {
	buf, err := ioutil.ReadFile(getVifFilePath(pid, b.iface.Name))
	if err != nil {
		return err
	}
	err = json.Unmarshal(buf, &b.vif)
	if err != nil {
		return err
	}
	b.vif.Gateway = b.vif.Gateway.To4()
	b.vif.GatewayIpv6 = b.vif.GatewayIpv6.To16()
	return nil
}

func (b *MasqueradeBindMechanism) setCachedVIF(pid string) error {
	return setCachedVIF(*b.vif, pid, b.iface.Name)
}

func (b *MasqueradeBindMechanism) createBridge() error {
	// Get dummy link
	bridgeNicName := fmt.Sprintf("%s-nic", b.bridgeInterfaceName)
	bridgeNicLink, err := b.handler.LinkByName(bridgeNicName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to find dummy interface for bridge")
		return err
	}

	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name: b.bridgeInterfaceName,
			MTU:  int(b.vif.Mtu),
		},
	}
	err = b.handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}

	err = b.handler.LinkSetMaster(bridgeNicLink, bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to connect %s interface to bridge %s", bridgeNicName, b.bridgeInterfaceName)
		return err
	}

	err = b.handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	if err := b.handler.AddrAdd(bridge, b.gatewayAddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}

	ipv6Enabled, err := b.handler.IsIpv6Enabled(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", b.podInterfaceName)
		return err
	}
	if ipv6Enabled {
		if err := b.handler.AddrAdd(bridge, b.gatewayIpv6Addr); err != nil {
			log.Log.Reason(err).Errorf("failed to set bridge IPv6")
			return err
		}
	}

	if err = b.handler.DisableTXOffloadChecksum(b.bridgeInterfaceName); err != nil {
		log.Log.Reason(err).Error("failed to disable TX offload checksum on bridge interface")
		return err
	}

	return nil
}

func (b *MasqueradeBindMechanism) createNatRules(protocol iptables.Protocol) error {
	err := b.handler.ConfigureIpForwarding(protocol)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to configure ip forwarding")
		return err
	}

	if b.handler.NftablesLoad(protocol) == nil {
		return b.createNatRulesUsingNftables(protocol)
	} else if b.handler.HasNatIptables(protocol) {
		return b.createNatRulesUsingIptables(protocol)
	}
	return fmt.Errorf("Couldn't configure ip nat rules")
}

func (b *MasqueradeBindMechanism) createNatRulesUsingIptables(protocol iptables.Protocol) error {
	err := b.handler.IptablesNewChain(protocol, "nat", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesNewChain(protocol, "nat", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "POSTROUTING", "-s", b.getVifIpByProtocol(protocol), "-j", "MASQUERADE")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "PREROUTING", "-i", b.podInterfaceName, "-j", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "POSTROUTING", "-o", b.bridgeInterfaceName, "-j", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	if len(b.iface.Ports) == 0 {
		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_PREINBOUND",
			"-j",
			"DNAT",
			"--to-destination", b.getVifIpByProtocol(protocol))

		return err
	}

	for _, port := range b.iface.Ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_POSTINBOUND",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"--source", getLoopbackAdrress(protocol),
			"-j",
			"SNAT",
			"--to-source", b.getGatewayByProtocol(protocol))
		if err != nil {
			return err
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_PREINBOUND",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"-j",
			"DNAT",
			"--to-destination", b.getVifIpByProtocol(protocol))
		if err != nil {
			return err
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "OUTPUT",
			"-p",
			strings.ToLower(port.Protocol),
			"--dport",
			strconv.Itoa(int(port.Port)),
			"--destination", getLoopbackAdrress(protocol),
			"-j",
			"DNAT",
			"--to-destination", b.getVifIpByProtocol(protocol))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *MasqueradeBindMechanism) getGatewayByProtocol(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return b.gatewayAddr.IP.String()
	} else {
		return b.gatewayIpv6Addr.IP.String()
	}
}

func (b *MasqueradeBindMechanism) getVifIpByProtocol(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return b.vif.IP.IP.String()
	} else {
		return b.vif.IPv6.IP.String()
	}
}

func getLoopbackAdrress(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return "127.0.0.1"
	} else {
		return "::1"
	}
}

func (b *MasqueradeBindMechanism) createNatRulesUsingNftables(proto iptables.Protocol) error {
	err := b.handler.NftablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "postrouting", b.handler.GetNFTIPString(proto), "saddr", b.getVifIpByProtocol(proto), "counter", "masquerade")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "prerouting", "iifname", b.podInterfaceName, "counter", "jump", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "postrouting", "oifname", b.bridgeInterfaceName, "counter", "jump", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	if len(b.iface.Ports) == 0 {
		err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND",
			"counter", "dnat", "to", b.getVifIpByProtocol(proto))

		return err
	}

	for _, port := range b.iface.Ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND",
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			b.handler.GetNFTIPString(proto), "saddr", getLoopbackAdrress(proto),
			"counter", "snat", "to", b.getGatewayByProtocol(proto))
		if err != nil {
			return err
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND",
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			"counter", "dnat", "to", b.getVifIpByProtocol(proto))
		if err != nil {
			return err
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "output",
			b.handler.GetNFTIPString(proto), "daddr", getLoopbackAdrress(proto),
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			"counter", "dnat", "to", b.getVifIpByProtocol(proto))
		if err != nil {
			return err
		}
	}

	return nil
}

type SlirpBindMechanism struct {
	iface  *v1.Interface
	domain *api.Domain
}

func (b *SlirpBindMechanism) discoverPodNetworkInterface() error {
	return nil
}

func (b *SlirpBindMechanism) preparePodNetworkInterfaces() error {
	return nil
}

func (b *SlirpBindMechanism) startDHCP() error {
	return nil
}

func (b *SlirpBindMechanism) decorateConfig() error {
	// remove slirp interface from domain spec devices interfaces
	var foundIfaceModelType string
	ifaces := b.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == b.iface.Name {
			b.domain.Spec.Devices.Interfaces = append(ifaces[:i], ifaces[i+1:]...)
			foundIfaceModelType = iface.Model.Type
			break
		}
	}

	if foundIfaceModelType == "" {
		return fmt.Errorf("failed to find interface %s in vmi spec", b.iface.Name)
	}

	qemuArg := fmt.Sprintf("%s,netdev=%s,id=%s", foundIfaceModelType, b.iface.Name, b.iface.Name)
	if b.iface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		qemuArg += fmt.Sprintf(",mac=%s", b.iface.MacAddress)
	}
	// Add interface configuration to qemuArgs
	b.domain.Spec.QEMUCmd.QEMUArg = append(b.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-device"})
	b.domain.Spec.QEMUCmd.QEMUArg = append(b.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: qemuArg})

	return nil
}

func (b *SlirpBindMechanism) loadCachedInterface() error {
	return nil
}

func (b *SlirpBindMechanism) loadCachedVIF(_ string) error {
	return nil
}

func (b *SlirpBindMechanism) setCachedVIF(_ string) error {
	return nil
}

func (b *SlirpBindMechanism) setCachedInterface() error {
	return nil
}

func (b *SlirpBindMechanism) wasCachedInterfaceLoaded() bool {
	return true
}

type MacvtapBindMechanism struct {
	vmi              *v1.VirtualMachineInstance
	iface            *v1.Interface
	virtIface        *api.Interface
	domain           *api.Domain
	podInterfaceName string
	podNicLink       netlink.Link
	mac              *net.HardwareAddr
	cacheFactory     cache.InterfaceCacheFactory
	launcherPID      *int
	handler          NetworkHandler
}

func (b *MacvtapBindMechanism) discoverPodNetworkInterface() error {
	link, err := b.handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", b.podInterfaceName)
		return err
	}
	b.podNicLink = link
	b.virtIface = &api.Interface{
		MAC: &api.MAC{MAC: b.podIfaceMAC()},
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  b.podInterfaceName,
			Managed: "no",
		},
	}

	return nil
}

func (b *MacvtapBindMechanism) podIfaceMAC() string {
	if b.mac != nil {
		return b.mac.String()
	} else {
		return b.podNicLink.Attrs().HardwareAddr.String()
	}
}

func (b *MacvtapBindMechanism) preparePodNetworkInterfaces() error {
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

func (b *MacvtapBindMechanism) loadCachedInterface() error {
	ifaceConfig, err := b.cacheFactory.CacheForPID(getPIDString(b.launcherPID)).Read(b.iface.Name)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return err
	}

	b.virtIface = ifaceConfig
	return nil
}

func (b *MacvtapBindMechanism) setCachedInterface() error {
	return b.cacheFactory.CacheForPID(getPIDString(b.launcherPID)).Write(b.iface.Name, b.virtIface)
}

func (b *MacvtapBindMechanism) wasCachedInterfaceLoaded() bool {
	return b.virtIface != nil
}

func (b *MacvtapBindMechanism) loadCachedVIF(_ string) error {
	return nil
}

func (b *MacvtapBindMechanism) setCachedVIF(_ string) error {
	return nil
}

func (b *MacvtapBindMechanism) startDHCP() error {
	// macvtap will connect to the host's subnet
	return nil
}

func createAndBindTapToBridge(handler NetworkHandler, deviceName string, bridgeIfaceName string, queueNumber uint32, launcherPID int, mtu int, tapOwner string) error {
	err := handler.CreateTapDevice(deviceName, queueNumber, launcherPID, mtu, tapOwner)
	if err != nil {
		return err
	}
	return handler.BindTapDeviceToBridge(deviceName, bridgeIfaceName)
}

func generateTapDeviceName(podInterfaceName string) string {
	return "tap" + podInterfaceName[3:]
}

func calculateNetworkQueues(vmi *v1.VirtualMachineInstance) uint32 {
	if isMultiqueue(vmi) {
		return converter.CalculateNetworkQueues(vmi)
	}
	return 0
}

func isMultiqueue(vmi *v1.VirtualMachineInstance) bool {
	return (vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue != nil) &&
		(*vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue)
}

func setCachedVIF(vif VIF, launcherPID string, ifaceName string) error {
	buf, err := json.MarshalIndent(vif, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling vif object: %v", err)
	}

	return writeVifFile(buf, launcherPID, ifaceName)
}
