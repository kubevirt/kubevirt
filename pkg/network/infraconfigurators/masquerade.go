package infraconfigurators

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/network/cache"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/network/istio"
	"kubevirt.io/kubevirt/pkg/network/link"
	virtnetlink "kubevirt.io/kubevirt/pkg/network/link"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
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

func (b *MasqueradePodNetworkConfigurator) PreparePodNetworkInterface() error {
	if err := b.createBridge(); err != nil {
		return err
	}

	tapOwner := netdriver.LibvirtUserAndGroupId
	if util.IsNonRootVMI(b.vmi) {
		tapOwner = strconv.Itoa(util.NonRootUID)
	}
	tapDeviceName := virtnetlink.GenerateTapDeviceName(b.podNicLink.Attrs().Name)

	queues := converter.CalculateNetworkQueues(b.vmi, converter.GetInterfaceType(b.vmiSpecIface))
	err := createAndBindTapToBridge(b.handler, tapDeviceName, b.bridgeInterfaceName, b.launcherPID, b.podNicLink.Attrs().MTU, tapOwner, queues)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", tapDeviceName)
		return err
	}

	ipv4Enabled, err := b.handler.HasIPv4GlobalUnicastAddress(b.podNicLink.Attrs().Name)
	if err != nil {
		log.Log.Reason(err).Errorf(ipVerifyFailFmt, "4", b.podNicLink.Attrs().Name)
		return err
	}
	if ipv4Enabled {
		err = b.handler.ConfigureRouteLocalNet(api.DefaultBridgeName)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to configure routing of local addresses for %s", api.DefaultBridgeName)
			return err
		}
		err = b.createNatRules(netdriver.IPv4)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to create ipv4 nat rules for vm error: %v", err)
			return err
		}
	}

	ipv6Enabled, err := b.handler.HasIPv6GlobalUnicastAddress(b.podNicLink.Attrs().Name)
	if err != nil {
		log.Log.Reason(err).Errorf(ipVerifyFailFmt, "6", b.podNicLink.Attrs().Name)
		return err
	}
	if ipv6Enabled {
		err = b.createNatRules(netdriver.IPv6)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to create ipv6 nat rules for vm error: %v", err)
			return err
		}
	}

	return nil
}

func (b *MasqueradePodNetworkConfigurator) GenerateNonRecoverableDomainIfaceSpec() *api.Interface {
	return nil
}

func (b *MasqueradePodNetworkConfigurator) createBridge() error {
	mac, err := net.ParseMAC(link.StaticMasqueradeBridgeMAC)
	if err != nil {
		return err
	}
	// Create a bridge
	bridge := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:         b.bridgeInterfaceName,
			MTU:          b.podNicLink.Attrs().MTU,
			HardwareAddr: mac,
		},
	}
	err = b.handler.LinkAdd(bridge)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create a bridge")
		return err
	}

	if err := b.handler.LinkSetUp(bridge); err != nil {
		log.Log.Reason(err).Errorf("failed to bring link up for interface: %s", b.bridgeInterfaceName)
		return err
	}

	if b.vmGatewayAddr != nil {
		if err := b.handler.AddrAdd(bridge, b.vmGatewayAddr); err != nil {
			log.Log.Reason(err).Errorf("failed to set bridge IP")
			return err
		}
	}

	if b.vmGatewayIpv6Addr != nil {
		if err := b.handler.AddrAdd(bridge, b.vmGatewayIpv6Addr); err != nil {
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

func GetLoopbackAdrress(ipVersion netdriver.IPVersion) string {
	if ipVersion == netdriver.IPv4 {
		return "127.0.0.1"
	} else {
		return "::1"
	}
}

func (b *MasqueradePodNetworkConfigurator) createNatRules(ipVersion netdriver.IPVersion) error {
	err := b.handler.ConfigureIpForwarding(ipVersion)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to configure ip forwarding")
		return err
	}

	if b.handler.CheckNftables() == nil {
		return b.createNatRulesUsingNftables(ipVersion)
	}
	return fmt.Errorf("Couldn't configure ip nat rules")
}

func (b *MasqueradePodNetworkConfigurator) createNatRulesUsingNftables(ipVersion netdriver.IPVersion) error {
	err := b.handler.NftablesNewTable(ipVersion, "nat")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(ipVersion, "nat", "prerouting { type nat hook prerouting priority -100; }")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(ipVersion, "nat", "input { type nat hook input priority 100; }")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(ipVersion, "nat", "output { type nat hook output priority -100; }")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(ipVersion, "nat", "postrouting { type nat hook postrouting priority 100; }")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(ipVersion, "nat", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(ipVersion, "nat", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(ipVersion, "nat", "postrouting", b.handler.GetNFTIPString(ipVersion), "saddr", b.geVmIfaceIpByProtocol(ipVersion), "counter", "masquerade")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(ipVersion, "nat", "prerouting", "iifname", b.podNicLink.Attrs().Name, "counter", "jump", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(ipVersion, "nat", "postrouting", "oifname", b.bridgeInterfaceName, "counter", "jump", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.skipForwardingForPortsUsingNftables(ipVersion, b.portsUsedByLiveMigration())
	if err != nil {
		return err
	}

	addressesToDnat, err := b.getDstAddressesToDnat(ipVersion)
	if err != nil {
		return err
	}

	if len(b.vmiSpecIface.Ports) == 0 {
		if istio.ProxyInjectionEnabled(b.vmi) {
			err = b.skipForwardingForPortsUsingNftables(ipVersion, istio.ReservedPorts())
			if err != nil {
				return err
			}
			for _, nonProxiedPort := range istio.NonProxiedPorts() {
				err = b.forwardPortUsingNftables(ipVersion, nonProxiedPort)
				if err != nil {
					return err
				}
			}
		}

		err = b.handler.NftablesAppendRule(ipVersion, "nat", "KUBEVIRT_POSTINBOUND",
			b.handler.GetNFTIPString(ipVersion), "saddr", b.getSrcAddressesToSnat(ipVersion),
			"counter", "snat", "to", b.getGatewayByProtocol(ipVersion))
		if err != nil {
			return err
		}

		if !istio.ProxyInjectionEnabled(b.vmi) {
			err = b.handler.NftablesAppendRule(ipVersion, "nat", "KUBEVIRT_PREINBOUND",
				"counter", "dnat", "to", b.geVmIfaceIpByProtocol(ipVersion))
			if err != nil {
				return err
			}
		}

		err = b.handler.NftablesAppendRule(ipVersion, "nat", "output",
			b.handler.GetNFTIPString(ipVersion), "daddr", addressesToDnat,
			"counter", "dnat", "to", b.geVmIfaceIpByProtocol(ipVersion))
		if err != nil {
			return err
		}

		return nil
	}

	for _, port := range b.vmiSpecIface.Ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}

		err = b.handler.NftablesAppendRule(ipVersion, "nat", "KUBEVIRT_POSTINBOUND",
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			b.handler.GetNFTIPString(ipVersion), "saddr", b.getSrcAddressesToSnat(ipVersion),
			"counter", "snat", "to", b.getGatewayByProtocol(ipVersion))
		if err != nil {
			return err
		}

		if !istio.ProxyInjectionEnabled(b.vmi) {
			err = b.handler.NftablesAppendRule(ipVersion, "nat", "KUBEVIRT_PREINBOUND",
				strings.ToLower(port.Protocol),
				"dport",
				strconv.Itoa(int(port.Port)),
				"counter", "dnat", "to", b.geVmIfaceIpByProtocol(ipVersion))
			if err != nil {
				return err
			}
		} else {
			for _, nonProxiedPort := range istio.NonProxiedPorts() {
				if int(port.Port) == nonProxiedPort {
					err = b.forwardPortUsingNftables(ipVersion, nonProxiedPort)
					if err != nil {
						return err
					}
				}
			}
		}

		err = b.handler.NftablesAppendRule(ipVersion, "nat", "output",
			b.handler.GetNFTIPString(ipVersion), "daddr", addressesToDnat,
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			"counter", "dnat", "to", b.geVmIfaceIpByProtocol(ipVersion))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *MasqueradePodNetworkConfigurator) skipForwardingForPortsUsingNftables(ipVersion netdriver.IPVersion, ports []string) error {
	if len(ports) == 0 {
		return nil
	}
	chainWhereDnatIsPerformed := "output"
	chainWhereSnatIsPerformed := "KUBEVIRT_POSTINBOUND"
	for _, chain := range []string{chainWhereDnatIsPerformed, chainWhereSnatIsPerformed} {
		err := b.handler.NftablesAppendRule(ipVersion, "nat", chain,
			"tcp", "dport", fmt.Sprintf(strFmt, strings.Join(ports, ", ")),
			b.handler.GetNFTIPString(ipVersion), "saddr", getLoopbackAdrress(ipVersion),
			"counter", "return")
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *MasqueradePodNetworkConfigurator) forwardPortUsingNftables(ipVersion netdriver.IPVersion, port int) error {
	return b.handler.NftablesAppendRule(ipVersion, "nat", "KUBEVIRT_PREINBOUND",
		"tcp", "dport", fmt.Sprintf("%d", port), "counter",
		"dnat", "to", b.geVmIfaceIpByProtocol(ipVersion))
}

func (b *MasqueradePodNetworkConfigurator) getGatewayByProtocol(ipVersion netdriver.IPVersion) string {
	if ipVersion == netdriver.IPv4 {
		return b.vmGatewayAddr.IP.String()
	} else {
		return b.vmGatewayIpv6Addr.IP.String()
	}
}

func (b *MasqueradePodNetworkConfigurator) geVmIfaceIpByProtocol(ipVersion netdriver.IPVersion) string {
	if ipVersion == netdriver.IPv4 {
		return b.vmIPv4Addr.IP.String()
	} else {
		return b.vmIPv6Addr.IP.String()
	}
}

func (b *MasqueradePodNetworkConfigurator) getSrcAddressesToSnat(ipVersion netdriver.IPVersion) string {
	addresses := []string{getLoopbackAdrress(ipVersion)}
	if istio.ProxyInjectionEnabled(b.vmi) && ipVersion == netdriver.IPv4 {
		addresses = append(addresses, istio.GetLoopbackAddress())
	}
	return fmt.Sprintf(strFmt, strings.Join(addresses, ", "))
}

func (b *MasqueradePodNetworkConfigurator) getDstAddressesToDnat(ipVersion netdriver.IPVersion) (string, error) {
	addresses := []string{getLoopbackAdrress(ipVersion)}
	if istio.ProxyInjectionEnabled(b.vmi) && ipVersion == netdriver.IPv4 {
		ipv4, _, err := b.handler.ReadIPAddressesFromLink(b.podNicLink.Attrs().Name)
		if err != nil {
			return "", err
		}
		addresses = append(addresses, ipv4)
	}
	return fmt.Sprintf(strFmt, strings.Join(addresses, ", ")), nil
}

func getLoopbackAdrress(ipVersion netdriver.IPVersion) string {
	if ipVersion == netdriver.IPv4 {
		return "127.0.0.1"
	} else {
		return "::1"
	}
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
