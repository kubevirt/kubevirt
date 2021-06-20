package infraconfigurators

import (
	"net"
	"strconv"

	"github.com/vishvananda/netlink"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type MacvtapPodNetworkConfigurator struct {
	podInterfaceName string
	podNicLink       netlink.Link
	vmMac            *net.HardwareAddr
	launcherPID      int
	handler          netdriver.NetworkHandler
}

func NewMacvtapPodNetworkConfigurator(podIfaceName string, vmMac *net.HardwareAddr, handler netdriver.NetworkHandler) *MacvtapPodNetworkConfigurator {
	return &MacvtapPodNetworkConfigurator{
		podInterfaceName: podIfaceName,
		vmMac:            vmMac,
		handler:          handler,
	}
}

func (b *MacvtapPodNetworkConfigurator) discoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link

	return nil
}

func (b *MacvtapPodNetworkConfigurator) preparePodNetworkInterface() error {
	return nil
}

func (b *MacvtapPodNetworkConfigurator) generateDomainIfaceSpec() api.Interface {
	return api.Interface{
		MAC: &api.MAC{MAC: b.podIfaceMAC()},
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  b.podNicLink.Attrs().Name,
			Managed: "no",
		},
	}
}
func (b *MacvtapPodNetworkConfigurator) DiscoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(b.podInterfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link

	return nil
}

func (b *MacvtapPodNetworkConfigurator) podIfaceMAC() string {
	if b.vmMac != nil {
		return b.vmMac.String()
	} else {
		return b.podNicLink.Attrs().HardwareAddr.String()
	}
}

func (b *MacvtapPodNetworkConfigurator) PreparePodNetworkInterface() error {
	return nil
}

func (b *MacvtapPodNetworkConfigurator) GenerateDomainIfaceSpec() api.Interface {
	return api.Interface{
		MAC: &api.MAC{MAC: b.podIfaceMAC()},
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  b.podNicLink.Attrs().Name,
			Managed: "no",
		},
	}
}

func (b *MacvtapPodNetworkConfigurator) GenerateDHCPConfig() *cache.DHCPConfig {
	return nil
}
