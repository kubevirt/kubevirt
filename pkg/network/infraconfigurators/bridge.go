package infraconfigurators

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type BridgePodNetworkConfigurator struct {
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
		IPAMDisabled: !b.ipamEnabled,
		IP:           b.podIfaceIP,
	}

	if b.ipamEnabled && len(b.podIfaceRoutes) > 0 {
		log.Log.V(4).Infof("got to add %d routes to the DhcpConfig", len(b.podIfaceRoutes))
		b.decorateDhcpConfigRoutes(dhcpConfig)
	}
	return dhcpConfig
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
