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
	"encoding/json"
	"fmt"
	"github.com/vishvananda/netlink"
	"io/ioutil"

	"k8s.io/apimachinery/pkg/types"
	netutils "k8s.io/utils/net"
	"os"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/client-go/precond"
	networkdriver "kubevirt.io/kubevirt/pkg/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var bridgeFakeIP = "169.254.75.1%d/32"

type BindMechanism interface {
	loadCachedInterface() (bool, error)

	// virt-handler that executes phase1 of network configuration needs to
	// pass details about discovered networking port into phase2 that is
	// executed by virt-launcher. Virt-launcher cannot discover some of
	// these details itself because at this point phase1 is complete and
	// ports are rewired, meaning, routes and IP addresses configured by
	// CNI plugin may be gone. For this matter, we use a cached VIF file to
	// pass discovered information between phases.
	loadCachedVIF() error

	// The following entry points require domain initialized for the
	// binding and can be used in phase2 only.
	decorateConfig() error
	startDHCP(vmi *v1.VirtualMachineInstance) error
}

type podNICImpl struct{}

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
	networkdriver.InitHandler()

	// There is nothing to plug for SR-IOV devices
	if iface.SRIOV != nil {
		return nil
	}

	bindMechanism, err := getPhase2Binding(vmi, iface, network, domain, podInterfaceName)
	if err != nil {
		return err
	}

	isExist, err := bindMechanism.loadCachedInterface()
	if err != nil {
		log.Log.Reason(err).Critical("failed to load cached interface configuration")
	}
	if !isExist {
		log.Log.Reason(err).Critical("cached interface configuration doesn't exist")
	}

	if err = bindMechanism.loadCachedVIF(); err != nil {
		log.Log.Reason(err).Critical("failed to load cached vif configuration")
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

func getPhase2Binding(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) (BindMechanism, error) {
	if iface.Bridge != nil {
		return generateBridgeBindingMech(vmi, iface, podInterfaceName, domain)
	}
	if iface.Masquerade != nil {
		return generateMasqueradeBindingMech(vmi, iface, network, domain, podInterfaceName)
	}
	if iface.Slirp != nil {
		return &SlirpBindMechanism{vmi: vmi, iface: iface, domain: domain}, nil
	}
	if iface.Macvtap != nil {
		return generateMacvtapBindingMech(vmi, iface, domain, podInterfaceName)
	}
	return nil, fmt.Errorf("Not implemented")
}

func generateMacvtapBindingMech(vmi *v1.VirtualMachineInstance, iface *v1.Interface, domain *api.Domain, podInterfaceName string) (BindMechanism, error) {
	mac, err := networkdriver.RetrieveMacAddress(iface)
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

func generateMasqueradeBindingMech(vmi *v1.VirtualMachineInstance, iface *v1.Interface, network *v1.Network, domain *api.Domain, podInterfaceName string) (BindMechanism, error) {
	mac, err := networkdriver.RetrieveMacAddress(iface)
	if err != nil {
		return nil, err
	}
	vif := &networkdriver.VIF{Name: podInterfaceName}
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

func generateBridgeBindingMech(vmi *v1.VirtualMachineInstance, iface *v1.Interface, podInterfaceName string, domain *api.Domain) (BindMechanism, error) {
	mac, err := networkdriver.RetrieveMacAddress(iface)
	if err != nil {
		return nil, err
	}
	vif := &networkdriver.VIF{Name: podInterfaceName}
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

type BridgeBindMechanism struct {
	vmi                 *v1.VirtualMachineInstance
	vif                 *networkdriver.VIF
	iface               *v1.Interface
	virtIface           *api.Interface
	podNicLink          netlink.Link
	domain              *api.Domain
	podInterfaceName    string
	bridgeInterfaceName string
	arpIgnore           bool
	launcherPID         int
	queueNumber         uint32
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

func (b *BridgeBindMechanism) startDHCP(vmi *v1.VirtualMachineInstance) error {
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
		return networkdriver.Handler.StartDHCP(b.vif, fakeServerAddr.IP, b.bridgeInterfaceName, b.iface.DHCPOptions, true)
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

func (b *BridgeBindMechanism) loadCachedInterface() (bool, error) {
	var ifaceConfig api.Interface

	err := networkdriver.ReadFromVirtLauncherCachedFile(&ifaceConfig, fmt.Sprintf("%d", b.launcherPID), b.iface.Name)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	b.virtIface = &ifaceConfig
	return true, nil
}

func (b *BridgeBindMechanism) loadCachedVIF() error {
	buf, err := ioutil.ReadFile(networkdriver.GetVifFilePath("self", b.iface.Name))
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

func (b *BridgeBindMechanism) setInterfaceRoutes() error {
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
		dhcpRoutes := networkdriver.FilterPodNetworkRoutes(routes, b.vif)
		b.vif.Routes = &dhcpRoutes
	}
	return nil
}

type MasqueradeBindMechanism struct {
	vmi                 *v1.VirtualMachineInstance
	vif                 *networkdriver.VIF
	iface               *v1.Interface
	virtIface           *api.Interface
	podNicLink          netlink.Link
	domain              *api.Domain
	podInterfaceName    string
	bridgeInterfaceName string
	vmNetworkCIDR       string
	vmIpv6NetworkCIDR   string
	gatewayAddr         *netlink.Addr
	gatewayIpv6Addr     *netlink.Addr
	launcherPID         int
	queueNumber         uint32
}

func (b *MasqueradeBindMechanism) startDHCP(vmi *v1.VirtualMachineInstance) error {
	return networkdriver.Handler.StartDHCP(b.vif, b.vif.Gateway, b.bridgeInterfaceName, b.iface.DHCPOptions, false)
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

func (b *MasqueradeBindMechanism) loadCachedInterface() (bool, error) {
	var ifaceConfig api.Interface

	err := networkdriver.ReadFromVirtLauncherCachedFile(&ifaceConfig, "self", b.iface.Name)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	b.virtIface = &ifaceConfig
	return true, nil
}

func (b *MasqueradeBindMechanism) loadCachedVIF() error {
	buf, err := ioutil.ReadFile(networkdriver.GetVifFilePath("self", b.iface.Name))
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

type SlirpBindMechanism struct {
	vmi         *v1.VirtualMachineInstance
	iface       *v1.Interface
	virtIface   *api.Interface
	domain      *api.Domain
	launcherPID int
}

func (s *SlirpBindMechanism) startDHCP(vmi *v1.VirtualMachineInstance) error {
	return nil
}

func (s *SlirpBindMechanism) decorateConfig() error {
	// remove slirp interface from domain spec devices interfaces
	var foundIfaceModelType string
	ifaces := s.domain.Spec.Devices.Interfaces
	for i, iface := range ifaces {
		if iface.Alias.GetName() == s.iface.Name {
			s.domain.Spec.Devices.Interfaces = append(ifaces[:i], ifaces[i+1:]...)
			foundIfaceModelType = iface.Model.Type
			break
		}
	}

	if foundIfaceModelType == "" {
		return fmt.Errorf("failed to find interface %s in vmi spec", s.iface.Name)
	}

	qemuArg := fmt.Sprintf("%s,netdev=%s,id=%s", foundIfaceModelType, s.iface.Name, s.iface.Name)
	if s.iface.MacAddress != "" {
		// We assume address was already validated in API layer so just pass it to libvirt as-is.
		qemuArg += fmt.Sprintf(",mac=%s", s.iface.MacAddress)
	}
	// Add interface configuration to qemuArgs
	s.domain.Spec.QEMUCmd.QEMUArg = append(s.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: "-device"})
	s.domain.Spec.QEMUCmd.QEMUArg = append(s.domain.Spec.QEMUCmd.QEMUArg, api.Arg{Value: qemuArg})

	return nil
}

func (s *SlirpBindMechanism) loadCachedInterface() (bool, error) {
	return true, nil
}

func (s *SlirpBindMechanism) loadCachedVIF() error {
	return nil
}

type MacvtapBindMechanism struct {
	vmi              *v1.VirtualMachineInstance
	iface            *v1.Interface
	virtIface        *api.Interface
	domain           *api.Domain
	podInterfaceName string
	podNicLink       netlink.Link
	launcherPID      int
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

func (b *MacvtapBindMechanism) loadCachedInterface() (bool, error) {
	var ifaceConfig api.Interface

	err := networkdriver.ReadFromVirtLauncherCachedFile(&ifaceConfig, fmt.Sprintf("%d", b.launcherPID), b.iface.Name)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	b.virtIface = &ifaceConfig
	return true, nil
}

func (b *MacvtapBindMechanism) loadCachedVIF() error {
	return nil
}

func (b *MacvtapBindMechanism) startDHCP(vmi *v1.VirtualMachineInstance) error {
	// macvtap will connect to the host's subnet
	return nil
}
