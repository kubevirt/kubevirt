package infraconfigurators

import (
	"net"
	"strconv"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type MacvtapPodNetworkConfigurator struct {
	podInterfaceName string
	podNicLink       netlink.Link
	vmiSpecIface     *v1.Interface
	vmMac            *net.HardwareAddr
	launcherPID      int
	handler          netdriver.NetworkHandler
}

func NewMacvtapPodNetworkConfigurator(podIfaceName string, vmiSpecIface *v1.Interface, handler netdriver.NetworkHandler) *MacvtapPodNetworkConfigurator {
	return &MacvtapPodNetworkConfigurator{
		podInterfaceName: podIfaceName,
		vmiSpecIface:     vmiSpecIface,
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
		MAC: &api.MAC{MAC: b.vmMac},
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

	b.vmMac, err = retrieveMacAddressFromVMISpecIface(b.vmiSpecIface)
	if err != nil {
		return err
	}
	if b.vmMac == nil {
		b.vmMac = &b.podNicLink.Attrs().HardwareAddr
	}

	return nil
}

func (b *MacvtapPodNetworkConfigurator) PreparePodNetworkInterface() error {
	return nil
}

func (b *MacvtapPodNetworkConfigurator) GenerateDomainIfaceSpec() api.Interface {
	return api.Interface{
		MAC: &api.MAC{MAC: b.vmMac.String()},
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
