package infraconfigurators

import (
	"crypto/rand"
	"fmt"
	"net"
	"strconv"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
)

type MacvtapPodNetworkConfigurator struct {
	bridgeInterfaceName string
	vmiSpecIface        *v1.Interface
	ipamEnabled         bool
	handler             netdriver.NetworkHandler
	launcherPID         int
	vmMac               *net.HardwareAddr
	podIfaceIP          netlink.Addr
	podNicLink          netlink.Link
	podIfaceRoutes      []netlink.Route
	tapDeviceName       string
	vmi                 *v1.VirtualMachineInstance
}

func NewMacvtapPodNetworkConfigurator(vmi *v1.VirtualMachineInstance, vmiSpecIface *v1.Interface, launcherPID int, handler netdriver.NetworkHandler) *MacvtapPodNetworkConfigurator {
	return &MacvtapPodNetworkConfigurator{
		vmi:          vmi,
		vmiSpecIface: vmiSpecIface,
		launcherPID:  launcherPID,
		handler:      handler,
	}
}

func (b *MacvtapPodNetworkConfigurator) DiscoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link
	b.bridgeInterfaceName = virtnetlink.GenerateBridgeName(link.Attrs().Name)

	addrList, err := b.handler.AddrList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", podIfaceName)
		return err
	}
	if len(addrList) == 0 {
		b.ipamEnabled = false
	} else {
		b.podIfaceIP = addrList[0]
		b.ipamEnabled = true
		if err := b.learnInterfaceRoutes(); err != nil {
			return err
		}
	}

	b.tapDeviceName = virtnetlink.GenerateTapDeviceName(podIfaceName)

	b.vmMac, err = virtnetlink.RetrieveMacAddressFromVMISpecIface(b.vmiSpecIface)
	if err != nil {
		return err
	}
	if b.vmMac == nil || b.podNicLink.Type() != "macvtap" {
		b.vmMac = &b.podNicLink.Attrs().HardwareAddr
	}

	return nil
}

func (b *MacvtapPodNetworkConfigurator) GenerateNonRecoverableDHCPConfig() *cache.DHCPConfig {
	if !b.ipamEnabled {
		return &cache.DHCPConfig{IPAMDisabled: true}
	}

	dhcpConfig := &cache.DHCPConfig{
		MAC:          *b.vmMac,
		IPAMDisabled: !b.ipamEnabled,
		IP:           b.podIfaceIP,
	}

	if b.ipamEnabled && len(b.podIfaceRoutes) > 0 {
		log.Log.V(4).Infof("got to add %d routes to the DhcpConfig", len(b.podIfaceRoutes))
		b.decorateDhcpConfigRoutes(dhcpConfig)
	}
	return dhcpConfig
}

func (b *MacvtapPodNetworkConfigurator) PreparePodNetworkInterface() error {

	if b.ipamEnabled {
		// In case if podNicLink is configured via 'macvtap-cni', we have no access to the lower device.
		// But we can use existing macvtap on this purpose. This way macvlan will inherit the same parent.
		if err := b.createMacvlan(); err != nil {
			log.Log.Reason(err).Errorf("failed to create macvlan device named %s", b.bridgeInterfaceName)
			return err
		}
	}

	if b.podNicLink.Type() == "macvtap" {
		return nil
	}

	// Set interface link to down to change its MAC address
	if err := b.handler.LinkSetDown(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", b.podNicLink.Attrs().Name)
		return err
	}

	if b.ipamEnabled {
		// Remove IP from POD interface
		err := b.handler.AddrDel(b.podNicLink, &b.podIfaceIP)

		if err != nil {
			log.Log.Reason(err).Errorf("failed to delete address for interface: %s", b.podNicLink.Attrs().Name)
			return err
		}

		if err := b.switchPodInterfaceWithDummy(); err != nil {
			log.Log.Reason(err).Error("failed to switch pod interface with a dummy")
			return err
		}

		// Set arp_ignore=1 to avoid
		// the dummy interface being seen by Duplicate Address Detection (DAD).
		// Without this, some VMs will lose their ip address after a few
		// minutes.
		if err := b.handler.ConfigureIpv4ArpIgnore(); err != nil {
			log.Log.Reason(err).Errorf("failed to set arp_ignore=1")
			return err
		}
	}

	tapOwner := netdriver.LibvirtUserAndGroupId
	if util.IsNonRootVMI(b.vmi) {
		tapOwner = strconv.Itoa(util.NonRootUID)
	}

	queues := converter.CalculateNetworkQueues(b.vmi, converter.GetInterfaceType(b.vmiSpecIface))
	err := createMacvtap(b.handler, b.tapDeviceName, b.podNicLink.Attrs().Name, b.launcherPID, b.podNicLink.Attrs().MTU, tapOwner, queues, b.vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", b.tapDeviceName)
		return err
	}

	tapDevice, err := b.handler.LinkByName(b.tapDeviceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get tap interface: %s", b.tapDeviceName)
		return err
	}

	// Swap MAC addresses with PodNic
	tapMac := b.podNicLink.Attrs().HardwareAddr
	podMac := GenerateMac()

	if err := netlink.LinkSetHardwareAddr(b.podNicLink, podMac); err != nil {
		log.Log.Reason(err).Errorf("failed to set pod interface mac address %s %s, error: %v", b.podNicLink.Attrs().Name, podMac, err)
		return err
	}
	if err := netlink.LinkSetHardwareAddr(tapDevice, tapMac); err != nil {
		log.Log.Reason(err).Errorf("failed to set tap interface mac address %s %s, error: %v", b.tapDeviceName, tapMac, err)
		return err
	}

	if err := b.handler.LinkSetUp(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.podNicLink.Attrs().Name)
		return err
	}

	return nil
}

func (b *MacvtapPodNetworkConfigurator) GenerateNonRecoverableDomainIfaceSpec() *api.Interface {
	return &api.Interface{
		MAC: &api.MAC{MAC: b.podNicLink.Attrs().HardwareAddr.String()},
	}
}

func (b *MacvtapPodNetworkConfigurator) learnInterfaceRoutes() error {
	routes, err := b.handler.RouteList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get routes for %s", b.podNicLink.Attrs().Name)
		return err
	}
	if len(routes) == 0 {
		return fmt.Errorf("no gateway address found in routes for %s", b.podNicLink.Attrs().Name)
	}
	b.podIfaceRoutes = routes
	return nil
}

func (b *MacvtapPodNetworkConfigurator) decorateDhcpConfigRoutes(dhcpConfig *cache.DHCPConfig) {
	log.Log.V(4).Infof("the default route is: %s", b.podIfaceRoutes[0].String())
	dhcpConfig.Gateway = b.podIfaceRoutes[0].Gw
	if len(b.podIfaceRoutes) > 1 {
		dhcpRoutes := virtnetlink.FilterPodNetworkRoutes(b.podIfaceRoutes, dhcpConfig)
		dhcpConfig.Routes = &dhcpRoutes
	}
}

func (b *MacvtapPodNetworkConfigurator) createMacvlan() error {
	m, err := netlink.LinkByName(b.podNicLink.Attrs().Name)
	if err != nil {
		return fmt.Errorf("failed to lookup lowerDevice %q: %v", b.podNicLink.Attrs().Name, err)
	}

	// Create a macvlan
	macvlanDevice := &netlink.Macvlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        b.bridgeInterfaceName,
			ParentIndex: m.Attrs().Index,
			// we had crashes if we did not set txqlen to some value
			TxQLen: m.Attrs().TxQLen,
		},
		Mode: netlink.MACVLAN_MODE_BRIDGE,
	}

	err = b.handler.LinkAdd(macvlanDevice)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a macvlan")
		return err
	}

	err = b.handler.LinkSetUp(macvlanDevice)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	// set fake ip on a macvlan
	addr := virtnetlink.GetFakeBridgeIP(b.vmi.Spec.Domain.Devices.Interfaces, b.vmiSpecIface)
	fakeaddr, _ := b.handler.ParseAddr(addr)

	if err := b.handler.AddrAdd(macvlanDevice, fakeaddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set macvlan IP")
		return err
	}

	return nil
}

func (b *MacvtapPodNetworkConfigurator) switchPodInterfaceWithDummy() error {
	originalPodInterfaceName := b.podNicLink.Attrs().Name
	newPodInterfaceName := virtnetlink.GenerateNewBridgedVmiInterfaceName(originalPodInterfaceName)
	dummy := &netlink.Dummy{LinkAttrs: netlink.LinkAttrs{Name: originalPodInterfaceName}}

	// Rename pod interface to free the original name for a new dummy interface
	err := b.handler.LinkSetName(b.podNicLink, newPodInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to rename interface : %s", b.podNicLink.Attrs().Name)
		return err
	}

	b.podNicLink, err = b.handler.LinkByName(newPodInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", newPodInterfaceName)
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
	err = b.handler.AddrReplace(dummy, &b.podIfaceIP)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to replace original IP address to dummy interface: %s", originalPodInterfaceName)
		return err
	}

	return nil
}

func GenerateMac() net.HardwareAddr {
	buf := make([]byte, 6)
	var mac net.HardwareAddr

	_, err := rand.Read(buf)
	if err != nil {
	}

	// Set local bit, ensure unicast address
	buf[0] = (buf[0] | 2) & 0xfe

	mac = append(mac, buf[0], buf[1], buf[2], buf[3], buf[4], buf[5])

	return mac
}
