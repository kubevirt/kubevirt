package infraconfigurators

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/network"
	"kubevirt.io/kubevirt/pkg/network/cache"
	"kubevirt.io/kubevirt/pkg/network/consts"
	netdriver "kubevirt.io/kubevirt/pkg/network/driver"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	LibvirtLocalConnectionPort         = 22222
	LibvirtDirectMigrationPort         = 49152
	LibvirtBlockMigrationPort          = 49153
	EnvoyAdminPort                     = 15000
	EnvoyOutboundPort                  = 15001
	EnvoyInboundPort                   = 15006
	EnvoyTunnelPort                    = 15008
	EnvoyMergedPrometheusTelemetryPort = 15020
	EnvoyHealthCheckPort               = 15021
	EnvoyPrometheusTelemetryPort       = 15090
)

type MasqueradePodNetworkConfigurator struct {
	vmi                 *v1.VirtualMachineInstance
	vmiSpecIface        *v1.Interface
	podNicLink          netlink.Link
	bridgeInterfaceName string
	vmNetworkCIDR       string
	vmIPv6NetworkCIDR   string
	vmGatewayAddr       *netlink.Addr
	vmGatewayIpv6Addr   *netlink.Addr
	launcherPID         int
	handler             netdriver.NetworkHandler
	vmIPv4Addr          netlink.Addr
	vmIPv6Addr          netlink.Addr
	vmMac               *net.HardwareAddr
}

func NewMasqueradePodNetworkConfigurator(vmi *v1.VirtualMachineInstance, vmiSpecIface *v1.Interface, bridgeIfaceName string, vmNetworkCIDR string, vmIPv6NetworkCIDR string, launcherPID int, handler netdriver.NetworkHandler) *MasqueradePodNetworkConfigurator {
	return &MasqueradePodNetworkConfigurator{
		vmi:                 vmi,
		vmiSpecIface:        vmiSpecIface,
		vmNetworkCIDR:       vmNetworkCIDR,
		vmIPv6NetworkCIDR:   vmIPv6NetworkCIDR,
		bridgeInterfaceName: bridgeIfaceName,
		launcherPID:         launcherPID,
		handler:             handler,
	}
}

func (b *MasqueradePodNetworkConfigurator) DiscoverPodNetworkInterface(podIfaceName string) error {
	link, err := b.handler.LinkByName(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get a link for interface: %s", podIfaceName)
		return err
	}
	b.podNicLink = link

	if err := validateMTU(b.podNicLink.Attrs().MTU); err != nil {
		return err
	}

	if err := b.configureIPv4Addresses(); err != nil {
		return err
	}

	ipv6Enabled, err := b.handler.IsIpv6Enabled(podIfaceName)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", podIfaceName)
		return err
	}
	if ipv6Enabled {
		if err := b.configureIPv6Addresses(); err != nil {
			return err
		}
	}

	b.vmMac, err = retrieveMacAddressFromVMISpecIface(b.vmiSpecIface)
	if err != nil {
		return err
	}

	return nil
}

func (b *MasqueradePodNetworkConfigurator) configureIPv4Addresses() error {
	b.setDefaultCidr(iptables.ProtocolIPv4)
	vmIPv4Addr, gatewayIPv4, err := b.generateGatewayAndVmIPAddrs(iptables.ProtocolIPv4)
	if err != nil {
		return err
	}
	b.vmIPv4Addr = *vmIPv4Addr
	b.vmGatewayAddr = gatewayIPv4
	return nil
}

func (b *MasqueradePodNetworkConfigurator) configureIPv6Addresses() error {
	b.setDefaultCidr(iptables.ProtocolIPv6)
	vmIPv6Addr, gatewayIPv6, err := b.generateGatewayAndVmIPAddrs(iptables.ProtocolIPv6)
	if err != nil {
		return err
	}
	b.vmIPv6Addr = *vmIPv6Addr
	b.vmGatewayIpv6Addr = gatewayIPv6
	return nil

}

func (b *MasqueradePodNetworkConfigurator) setDefaultCidr(protocol iptables.Protocol) {
	if protocol == iptables.ProtocolIPv4 {
		if b.vmNetworkCIDR == "" {
			b.vmNetworkCIDR = api.DefaultVMCIDR
		}
	} else {
		if b.vmIPv6NetworkCIDR == "" {
			b.vmIPv6NetworkCIDR = api.DefaultVMIpv6CIDR
		}
	}
}

func (b *MasqueradePodNetworkConfigurator) generateGatewayAndVmIPAddrs(protocol iptables.Protocol) (*netlink.Addr, *netlink.Addr, error) {
	cidrToConfigure := b.vmNetworkCIDR
	if protocol == iptables.ProtocolIPv6 {
		cidrToConfigure = b.vmIPv6NetworkCIDR
	}

	vmIP, gatewayIP, err := b.handler.GetHostAndGwAddressesFromCIDR(cidrToConfigure)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to get gw and vm available addresses from CIDR %s", cidrToConfigure)
		return nil, nil, err
	}

	gatewayAddr, err := b.handler.ParseAddr(gatewayIP)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse gateway address %s err %v", gatewayAddr, err)
	}
	vmAddr, err := b.handler.ParseAddr(vmIP)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse vm address %s err %v", vmAddr, err)
	}
	return gatewayAddr, vmAddr, nil
}

func (b *MasqueradePodNetworkConfigurator) GenerateDHCPConfig() *cache.DHCPConfig {
	dhcpConfig := &cache.DHCPConfig{
		Name: b.podNicLink.Attrs().Name,
		IP:   b.vmIPv4Addr,
		IPv6: b.vmIPv6Addr,
	}
	if b.vmMac != nil {
		dhcpConfig.MAC = *b.vmMac
	}
	if b.podNicLink != nil {
		dhcpConfig.Mtu = uint16(b.podNicLink.Attrs().MTU)
	}
	if b.vmGatewayAddr != nil {
		dhcpConfig.AdvertisingIPAddr = b.vmGatewayAddr.IP.To4()
		dhcpConfig.Gateway = b.vmGatewayAddr.IP.To4()
	}
	if b.vmGatewayIpv6Addr != nil {
		dhcpConfig.AdvertisingIPv6Addr = b.vmGatewayIpv6Addr.IP.To16()
	}

	return dhcpConfig
}

func (b *MasqueradePodNetworkConfigurator) PreparePodNetworkInterface() error {
	if err := b.createBridge(); err != nil {
		return err
	}

	tapOwner := netdriver.LibvirtUserAndGroupId
	if util.IsNonRootVMI(b.vmi) {
		tapOwner = strconv.Itoa(util.NonRootUID)
	}
	tapDeviceName := generateTapDeviceName(b.podNicLink.Attrs().Name)
	err := createAndBindTapToBridge(b.handler, tapDeviceName, b.bridgeInterfaceName, b.launcherPID, b.podNicLink.Attrs().MTU, tapOwner, b.vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create tap device named %s", tapDeviceName)
		return err
	}

	err = b.createNatRules(iptables.ProtocolIPv4)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create ipv4 nat rules for vm error: %v", err)
		return err
	}

	ipv6Enabled, err := b.handler.IsIpv6Enabled(b.podNicLink.Attrs().Name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", b.podNicLink.Attrs().Name)
		return err
	}
	if ipv6Enabled {
		err = b.createNatRules(iptables.ProtocolIPv6)
		if err != nil {
			log.Log.Reason(err).Errorf("failed to create ipv6 nat rules for vm error: %v", err)
			return err
		}
	}

	return nil
}

func (b *MasqueradePodNetworkConfigurator) GenerateDomainIfaceSpec() api.Interface {
	domainIface := api.Interface{
		MTU: &api.MTU{Size: strconv.Itoa(b.podNicLink.Attrs().MTU)},
		Target: &api.InterfaceTarget{
			Device:  generateTapDeviceName(b.podNicLink.Attrs().Name),
			Managed: "no",
		},
	}
	if b.vmMac != nil {
		domainIface.MAC = &api.MAC{MAC: b.vmMac.String()}
	}
	return domainIface
}

func (b *MasqueradePodNetworkConfigurator) createBridge() error {
	mac, err := net.ParseMAC(network.StaticMasqueradeBridgeMAC)
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

	if err := b.handler.AddrAdd(bridge, b.vmGatewayAddr); err != nil {
		log.Log.Reason(err).Errorf("failed to set bridge IP")
		return err
	}
	ipv6Enabled, err := b.handler.IsIpv6Enabled(b.podNicLink.Attrs().Name)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to verify whether ipv6 is configured on %s", b.podNicLink.Attrs().Name)
		return err
	}
	if ipv6Enabled {
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

func (b *MasqueradePodNetworkConfigurator) skipForwardingForReservedPortsUsingNftables(proto iptables.Protocol) error {
	chainWhereDnatIsPerformed := "output"
	chainWhereSnatIsPerformed := "KUBEVIRT_POSTINBOUND"
	for _, chain := range []string{chainWhereDnatIsPerformed, chainWhereSnatIsPerformed} {
		err := b.handler.NftablesAppendRule(proto, "nat", chain,
			"tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(PortsUsedByLiveMigration(), ", ")),
			b.handler.GetNFTIPString(proto), "saddr", GetLoopbackAdrress(proto),
			"counter", "return")
		if err != nil {
			return err
		}
	}
	return nil
}

func hasIstioSidecarInjectionEnabled(vmi *v1.VirtualMachineInstance) bool {
	if val, ok := vmi.GetAnnotations()[consts.ISTIO_INJECT_ANNOTATION]; ok {
		return strings.ToLower(val) == "true"
	}
	return false
}

func GetEnvoyLoopbackAddress() string {
	return "127.0.0.6"
}

func GetLoopbackAdrress(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return "127.0.0.1"
	} else {
		return "::1"
	}
}

func PortsUsedByLiveMigration() []string {
	return []string{
		fmt.Sprint(LibvirtLocalConnectionPort),
		fmt.Sprint(LibvirtDirectMigrationPort),
		fmt.Sprint(LibvirtBlockMigrationPort),
	}
}

func (b *MasqueradePodNetworkConfigurator) createNatRules(protocol iptables.Protocol) error {
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

func (b *MasqueradePodNetworkConfigurator) createNatRulesUsingIptables(protocol iptables.Protocol) error {
	err := b.handler.IptablesNewChain(protocol, "nat", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesNewChain(protocol, "nat", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "POSTROUTING", "-s", b.geVmIfaceIpByProtocol(protocol), "-j", "MASQUERADE")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "PREROUTING", "-i", b.podNicLink.Attrs().Name, "-j", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.IptablesAppendRule(protocol, "nat", "POSTROUTING", "-o", b.bridgeInterfaceName, "-j", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.skipForwardingForPortsUsingIptables(protocol, portsUsedByLiveMigration())
	if err != nil {
		return err
	}

	if len(b.vmiSpecIface.Ports) == 0 {
		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_PREINBOUND",
			"-j",
			"DNAT",
			"--to-destination", b.geVmIfaceIpByProtocol(protocol))
		if err != nil {
			return err
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "KUBEVIRT_POSTINBOUND",
			"--source", getLoopbackAdrress(protocol),
			"-j",
			"SNAT",
			"--to-source", b.getGatewayByProtocol(protocol))
		if err != nil {
			return err
		}

		err = b.handler.IptablesAppendRule(protocol, "nat", "OUTPUT",
			"--destination", getLoopbackAdrress(protocol),
			"-j",
			"DNAT",
			"--to-destination", b.geVmIfaceIpByProtocol(protocol))
		if err != nil {
			return err
		}

		return nil
	}

	for _, port := range b.vmiSpecIface.Ports {
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
			"--to-destination", b.geVmIfaceIpByProtocol(protocol))
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
			"--to-destination", b.geVmIfaceIpByProtocol(protocol))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *MasqueradePodNetworkConfigurator) skipForwardingForPortsUsingIptables(protocol iptables.Protocol, ports []string) error {
	chainWhereDnatIsPerformed := "OUTPUT"
	chainWhereSnatIsPerformed := "KUBEVIRT_POSTINBOUND"
	for _, chain := range []string{chainWhereDnatIsPerformed, chainWhereSnatIsPerformed} {
		err := b.handler.IptablesAppendRule(protocol, "nat", chain,
			"-p", "tcp", "--match", "multiport",
			"--dports", fmt.Sprintf("%s", strings.Join(ports, ",")),
			"--source", getLoopbackAdrress(protocol),
			"-j", "RETURN")
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *MasqueradePodNetworkConfigurator) createNatRulesUsingNftables(proto iptables.Protocol) error {
	err := b.handler.NftablesNewChain(proto, "nat", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesNewChain(proto, "nat", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "postrouting", b.handler.GetNFTIPString(proto), "saddr", b.geVmIfaceIpByProtocol(proto), "counter", "masquerade")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "prerouting", "iifname", b.podNicLink.Attrs().Name, "counter", "jump", "KUBEVIRT_PREINBOUND")
	if err != nil {
		return err
	}

	err = b.handler.NftablesAppendRule(proto, "nat", "postrouting", "oifname", b.bridgeInterfaceName, "counter", "jump", "KUBEVIRT_POSTINBOUND")
	if err != nil {
		return err
	}

	err = b.skipForwardingForPortsUsingNftables(proto, portsUsedByLiveMigration())
	if err != nil {
		return err
	}

	addressesToDnat, err := b.getDstAddressesToDnat(proto)
	if err != nil {
		return err
	}

	if len(b.vmiSpecIface.Ports) == 0 {
		if hasIstioSidecarInjectionEnabled(b.vmi) {
			err = b.skipForwardingForPortsUsingNftables(proto, PortsUsedByIstio())
			if err != nil {
				return err
			}
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND",
			b.handler.GetNFTIPString(proto), "saddr", b.getSrcAddressesToSnat(proto),
			"counter", "snat", "to", b.getGatewayByProtocol(proto))
		if err != nil {
			return err
		}

		if !hasIstioSidecarInjectionEnabled(b.vmi) {
			err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND",
				"counter", "dnat", "to", b.geVmIfaceIpByProtocol(proto))
			if err != nil {
				return err
			}
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "output",
			b.handler.GetNFTIPString(proto), "daddr", addressesToDnat,
			"counter", "dnat", "to", b.geVmIfaceIpByProtocol(proto))
		if err != nil {
			return err
		}

		return nil
	}

	for _, port := range b.vmiSpecIface.Ports {
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_POSTINBOUND",
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			b.handler.GetNFTIPString(proto), "saddr", b.getSrcAddressesToSnat(proto),
			"counter", "snat", "to", b.getGatewayByProtocol(proto))
		if err != nil {
			return err
		}

		if !hasIstioSidecarInjectionEnabled(b.vmi) {
			err = b.handler.NftablesAppendRule(proto, "nat", "KUBEVIRT_PREINBOUND",
				strings.ToLower(port.Protocol),
				"dport",
				strconv.Itoa(int(port.Port)),
				"counter", "dnat", "to", b.geVmIfaceIpByProtocol(proto))
			if err != nil {
				return err
			}
		}

		err = b.handler.NftablesAppendRule(proto, "nat", "output",
			b.handler.GetNFTIPString(proto), "daddr", addressesToDnat,
			strings.ToLower(port.Protocol),
			"dport",
			strconv.Itoa(int(port.Port)),
			"counter", "dnat", "to", b.geVmIfaceIpByProtocol(proto))
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *MasqueradePodNetworkConfigurator) skipForwardingForPortsUsingNftables(proto iptables.Protocol, ports []string) error {
	chainWhereDnatIsPerformed := "output"
	chainWhereSnatIsPerformed := "KUBEVIRT_POSTINBOUND"
	for _, chain := range []string{chainWhereDnatIsPerformed, chainWhereSnatIsPerformed} {
		err := b.handler.NftablesAppendRule(proto, "nat", chain,
			"tcp", "dport", fmt.Sprintf("{ %s }", strings.Join(ports, ", ")),
			b.handler.GetNFTIPString(proto), "saddr", getLoopbackAdrress(proto),
			"counter", "return")
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *MasqueradePodNetworkConfigurator) getGatewayByProtocol(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return b.vmGatewayAddr.IP.String()
	} else {
		return b.vmGatewayIpv6Addr.IP.String()
	}
}

func (b *MasqueradePodNetworkConfigurator) geVmIfaceIpByProtocol(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return b.vmIPv4Addr.IP.String()
	} else {
		return b.vmIPv6Addr.IP.String()
	}
}

func (b *MasqueradePodNetworkConfigurator) getSrcAddressesToSnat(proto iptables.Protocol) string {
	addresses := []string{getLoopbackAdrress(proto)}
	if hasIstioSidecarInjectionEnabled(b.vmi) && proto == iptables.ProtocolIPv4 {
		addresses = append(addresses, getEnvoyLoopbackAddress())
	}
	return fmt.Sprintf("{ %s }", strings.Join(addresses, ", "))
}

func (b *MasqueradePodNetworkConfigurator) getDstAddressesToDnat(proto iptables.Protocol) (string, error) {
	addresses := []string{getLoopbackAdrress(proto)}
	if hasIstioSidecarInjectionEnabled(b.vmi) && proto == iptables.ProtocolIPv4 {
		ipv4, _, err := b.handler.ReadIPAddressesFromLink(b.podNicLink.Attrs().Name)
		if err != nil {
			return "", err
		}
		addresses = append(addresses, ipv4)
	}
	return fmt.Sprintf("{ %s }", strings.Join(addresses, ", ")), nil
}

func getEnvoyLoopbackAddress() string {
	return "127.0.0.6"
}

func getLoopbackAdrress(proto iptables.Protocol) string {
	if proto == iptables.ProtocolIPv4 {
		return "127.0.0.1"
	} else {
		return "::1"
	}
}

func portsUsedByLiveMigration() []string {
	return []string{
		fmt.Sprint(LibvirtLocalConnectionPort),
		fmt.Sprint(LibvirtDirectMigrationPort),
		fmt.Sprint(LibvirtBlockMigrationPort),
	}
}

func PortsUsedByIstio() []string {
	return []string{
		fmt.Sprint(EnvoyAdminPort),
		fmt.Sprint(EnvoyOutboundPort),
		fmt.Sprint(EnvoyInboundPort),
		fmt.Sprint(EnvoyTunnelPort),
		fmt.Sprint(EnvoyMergedPrometheusTelemetryPort),
		fmt.Sprint(EnvoyHealthCheckPort),
		fmt.Sprint(EnvoyPrometheusTelemetryPort),
	}
}
