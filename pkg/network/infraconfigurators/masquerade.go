package infraconfigurators

import (
	"fmt"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	ipVerifyFailFmt            = "failed to verify whether ipv%s is configured on %s"
	strFmt                     = "{ %s }"
	LibvirtDirectMigrationPort = 49152
	LibvirtBlockMigrationPort  = 49153
)

type MasqueradePodNetworkConfigurator struct {
	vmi                 *v1.VirtualMachineInstance
	vmiSpecIface        *v1.Interface
	vmiSpecNetwork      *v1.Network
	podNicLink          netlink.Link
	bridgeInterfaceName string
	vmGatewayAddr       *netlink.Addr
	vmGatewayIpv6Addr   *netlink.Addr
	launcherPID         int
	handler             netdriver.NetworkHandler
	vmIPv4Addr          netlink.Addr
	vmIPv6Addr          netlink.Addr
}

func NewMasqueradePodNetworkConfigurator(vmi *v1.VirtualMachineInstance, vmiSpecIface *v1.Interface, vmiSpecNetwork *v1.Network, launcherPID int, handler netdriver.NetworkHandler) *MasqueradePodNetworkConfigurator {
	return &MasqueradePodNetworkConfigurator{
		vmi:            vmi,
		vmiSpecIface:   vmiSpecIface,
		vmiSpecNetwork: vmiSpecNetwork,
		launcherPID:    launcherPID,
		handler:        handler,
	}
}

func (b *MasqueradePodNetworkConfigurator) DiscoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link
	b.bridgeInterfaceName = virtnetlink.GenerateBridgeName(link.Attrs().Name)

	ipv4Enabled, err := b.handler.HasIPv4GlobalUnicastAddress(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf(ipVerifyFailFmt, "4", podIfaceName)
		return err
	}
	if ipv4Enabled {
		if err := b.computeIPv4GatewayAndVmIp(); err != nil {
			return err
		}
	}

	ipv6Enabled, err := b.handler.HasIPv6GlobalUnicastAddress(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf(ipVerifyFailFmt, "6", podIfaceName)
		return err
	}
	if ipv6Enabled {
		if err := b.computeIPv6GatewayAndVmIp(); err != nil {
			return err
		}
	}

	return nil
}

func (b *MasqueradePodNetworkConfigurator) computeIPv4GatewayAndVmIp() error {
	ipv4Gateway, ipv4, err := virtnetlink.GenerateMasqueradeGatewayAndVmIPAddrs(b.vmiSpecNetwork, netdriver.IPv4)
	if err != nil {
		return err
	}

	b.vmGatewayAddr = ipv4Gateway
	b.vmIPv4Addr = *ipv4
	return nil
}

func (b *MasqueradePodNetworkConfigurator) computeIPv6GatewayAndVmIp() error {
	ipv6Gateway, ipv6, err := virtnetlink.GenerateMasqueradeGatewayAndVmIPAddrs(b.vmiSpecNetwork, netdriver.IPv6)
	if err != nil {
		return err
	}
	b.vmGatewayIpv6Addr = ipv6Gateway
	b.vmIPv6Addr = *ipv6
	return nil
}

func (b *MasqueradePodNetworkConfigurator) GenerateNonRecoverableDHCPConfig() *cache.DHCPConfig {
	return nil
}

func (b *MasqueradePodNetworkConfigurator) GenerateNonRecoverableDomainIfaceSpec() *api.Interface {
	return nil
}

func (b *MasqueradePodNetworkConfigurator) portsUsedByLiveMigration() []string {
	if b.vmi.Status.MigrationTransport == v1.MigrationTransportUnix {
		return nil
	}
	return []string{
		fmt.Sprint(LibvirtDirectMigrationPort),
		fmt.Sprint(LibvirtBlockMigrationPort),
	}
}
