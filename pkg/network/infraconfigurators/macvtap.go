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
	vmi              *v1.VirtualMachineInstance
	iface            *v1.Interface
	virtIface        *api.Interface
	podInterfaceName string
	podNicLink       netlink.Link
	mac              *net.HardwareAddr
	storeFactory     cache.InterfaceCacheFactory
	launcherPID      int
	handler          netdriver.NetworkHandler
}

func NewMacvtapPodNetworkConfigurator(vmi *v1.VirtualMachineInstance, iface *v1.Interface, podIfaceName string, mac *net.HardwareAddr, cacheFactory cache.InterfaceCacheFactory, launcherPID *int, handler netdriver.NetworkHandler) *MacvtapPodNetworkConfigurator {
	return &MacvtapPodNetworkConfigurator{
		vmi:              vmi,
		iface:            iface,
		podInterfaceName: podIfaceName,
		mac:              mac,
		storeFactory:     cacheFactory,
		launcherPID:      *launcherPID,
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
	if b.mac != nil {
		return b.mac.String()
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
