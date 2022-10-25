package infraconfigurators

import (
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

type BridgePodNetworkConfigurator struct {
	bridgeInterfaceName string
	vmiSpecIface        *v1.Interface
	ipamEnabled         bool
	handler             netdriver.NetworkHandler
	launcherPID         int
	vmMac               *net.HardwareAddr
	podIfaceIP          *netlink.Addr
	podIfaceIPv6        *netlink.Addr
	podNicLink          netlink.Link
	podIfaceRoutes      []netlink.Route
	tapDeviceName       string
	vmi                 *v1.VirtualMachineInstance
}

func NewBridgePodNetworkConfigurator(vmi *v1.VirtualMachineInstance, vmiSpecIface *v1.Interface, launcherPID int, handler netdriver.NetworkHandler) *BridgePodNetworkConfigurator {
	return &BridgePodNetworkConfigurator{
		vmi:          vmi,
		vmiSpecIface: vmiSpecIface,
		launcherPID:  launcherPID,
		handler:      handler,
	}
}

func (b *BridgePodNetworkConfigurator) DiscoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link

	b.ipamEnabled = false

	addrList, err := b.handler.AddrList(b.podNicLink, netlink.FAMILY_V4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ip address for %s", podIfaceName)
		return err
	}
	if len(addrList) > 0 {
		b.podIfaceIP = &addrList[0]
		b.ipamEnabled = true
		if err := b.learnInterfaceRoutes(); err != nil {
			return err
		}
	}

	addrV6List, err := b.handler.AddrList(b.podNicLink, netlink.FAMILY_V6)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get an ipv6 address for %s", podIfaceName)
		return err
	}
	for _, addr := range addrV6List {
		if addr.IP.IsGlobalUnicast() {
			b.podIfaceIPv6 = &addr
			b.ipamEnabled = true
			break
		}
	}

	b.bridgeInterfaceName = virtnetlink.GenerateBridgeName(podIfaceName)
	b.tapDeviceName = virtnetlink.GenerateTapDeviceName(podIfaceName)

	b.vmMac, err = virtnetlink.RetrieveMacAddressFromVMISpecIface(b.vmiSpecIface)
	if err != nil {
		return err
	}
	if b.vmMac == nil {
		b.vmMac = &b.podNicLink.Attrs().HardwareAddr
	}

	return nil
}

func (b *BridgePodNetworkConfigurator) GenerateNonRecoverableDHCPConfig() *cache.DHCPConfig {
	if !b.ipamEnabled {
		return &cache.DHCPConfig{IPAMDisabled: true}
	}

	dhcpConfig := &cache.DHCPConfig{
		MAC:          *b.vmMac,
		IPAMDisabled: false,
	}

	if b.podIfaceIP != nil {
		dhcpConfig.IP = *b.podIfaceIP
		if len(b.podIfaceRoutes) > 0 {
			log.Log.V(4).Infof("got to add %d routes to the DhcpConfig", len(b.podIfaceRoutes))
			b.decorateDhcpConfigRoutes(dhcpConfig)
		}
	}
	if b.podIfaceIPv6 != nil {
		dhcpConfig.IPv6 = *b.podIfaceIPv6
	}

	return dhcpConfig
}

func (b *BridgePodNetworkConfigurator) PreparePodNetworkInterface() error {
	// Ensure that any IPv6 address will be removed by setting the link down
	if err := b.handler.ConfigureIpv6FlushAddrOnDown(); err != nil {
		log.Log.Reason(err).Error("failed to set keep_addr_on_down=-1")
		return err
	}

	// Set interface link to down to change its MAC address
	if err := b.handler.LinkSetDown(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link down for interface: %s", b.podNicLink.Attrs().Name)
		return err
	}

	if b.ipamEnabled {
		// Remove IP from POD interface
		if b.podIfaceIP != nil {
			err := b.handler.AddrDel(b.podNicLink, b.podIfaceIP)
			if err != nil {
				log.Log.Reason(err).Errorf("failed to delete v4 address for interface: %s", b.podNicLink.Attrs().Name)
				return err
			}
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

	if err := b.createBridge(); err != nil {
		return err
	}

	tapOwner := netdriver.LibvirtUserAndGroupId
	if util.IsNonRootVMI(b.vmi) {
		tapOwner = strconv.Itoa(util.NonRootUID)
	}

	queues := converter.CalculateNetworkQueues(b.vmi, converter.GetInterfaceType(b.vmiSpecIface))
	err := createAndBindTapToBridge(b.handler, b.tapDeviceName, b.bridgeInterfaceName, b.launcherPID, b.podNicLink.Attrs().MTU, tapOwner, queues)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", b.tapDeviceName)
		return err
	}

	if err := b.handler.LinkSetUp(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.podNicLink.Attrs().Name)
		return err
	}

	if err := b.handler.LinkSetLearningOff(b.podNicLink); err != nil {
		log.Log.Reason(err).Errorf("failed to disable mac learning for interface: %s", b.podNicLink.Attrs().Name)
		return err
	}

	return nil
}

func (b *BridgePodNetworkConfigurator) GenerateNonRecoverableDomainIfaceSpec() *api.Interface {
	return &api.Interface{
		MAC: &api.MAC{MAC: b.vmMac.String()},
	}
}

func (b *BridgePodNetworkConfigurator) learnInterfaceRoutes() error {
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

func (b *BridgePodNetworkConfigurator) decorateDhcpConfigRoutes(dhcpConfig *cache.DHCPConfig) {
	log.Log.V(4).Infof("the default route is: %s", b.podIfaceRoutes[0].String())
	dhcpConfig.Gateway = b.podIfaceRoutes[0].Gw
	if len(b.podIfaceRoutes) > 1 {
		dhcpRoutes := virtnetlink.FilterPodNetworkRoutes(b.podIfaceRoutes, dhcpConfig)
		dhcpConfig.Routes = &dhcpRoutes
	}
}

func (b *BridgePodNetworkConfigurator) createBridge() error {
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

	brLink, err := b.handler.LinkByName(b.bridgeInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to fetch bridge %s link", bridge.Name)
		return err
	}

	if err := b.handler.LinkSetHardwareAddr(b.podNicLink, brLink.Attrs().HardwareAddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set on pod interface (%s) the mac (%s)", b.podNicLink.Attrs().Name, brLink.Attrs().HardwareAddr.String())
		return err
	}

	err = b.handler.LinkSetMaster(b.podNicLink, bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to connect interface %s to bridge %s", b.podNicLink.Attrs().Name, bridge.Name)
		return err
	}

	err = b.handler.LinkSetUp(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	// set fake ip on a bridge
	addr := virtnetlink.GetFakeBridgeIP(b.vmi.Spec.Domain.Devices.Interfaces, b.vmiSpecIface)
	fakeaddr, _ := b.handler.ParseAddr(addr)

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

func (b *BridgePodNetworkConfigurator) switchPodInterfaceWithDummy() error {
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
	if b.podIfaceIP != nil {
		err = b.handler.AddrReplace(dummy, b.podIfaceIP)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to replace original IP address to dummy interface: %s", originalPodInterfaceName)
			return err
		}
	}
	if b.podIfaceIPv6 != nil {
		err = b.handler.AddrReplace(dummy, b.podIfaceIPv6)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to replace original IPv6 address to dummy interface: %s", originalPodInterfaceName)
			return err
		}
	}

	return nil
}
