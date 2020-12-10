package dhcpv6

import (
	"fmt"
	"net"
)

// InterfaceAddresses is used to fetch addresses of an interface with given name
var InterfaceAddresses func(string) ([]net.Addr, error) = interfaceAddresses

func interfaceAddresses(ifname string) ([]net.Addr, error) {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return nil, err
	}
	return iface.Addrs()
}

func getMatchingAddr(ifname string, matches func(net.IP) bool) (net.IP, error) {
	ifaddrs, err := InterfaceAddresses(ifname)
	if err != nil {
		return nil, err
	}
	for _, ifaddr := range ifaddrs {
		if ifaddr, ok := ifaddr.(*net.IPNet); ok && matches(ifaddr.IP) {
			return ifaddr.IP, nil
		}
	}
	return nil, fmt.Errorf("no matching address found for interface %s", ifname)
}

// GetLinkLocalAddr returns a link-local address for the interface
func GetLinkLocalAddr(ifname string) (net.IP, error) {
	return getMatchingAddr(ifname, func(ip net.IP) bool {
		return ip.To4() == nil && ip.IsLinkLocalUnicast()
	})
}

// GetGlobalAddr returns a global address for the interface
func GetGlobalAddr(ifname string) (net.IP, error) {
	return getMatchingAddr(ifname, func(ip net.IP) bool {
		return ip.To4() == nil && ip.IsGlobalUnicast()
	})
}

// GetMacAddressFromEUI64 will return a valid MAC address ONLY if it's a EUI-48
func GetMacAddressFromEUI64(ip net.IP) (net.HardwareAddr, error) {
	if ip.To16() == nil {
		return nil, fmt.Errorf("IP address shorter than 16 bytes")
	}

	if isEUI48 := ip[11] == 0xff && ip[12] == 0xfe; !isEUI48 {
		return nil, fmt.Errorf("IP address is not an EUI48 address")
	}

	mac := make(net.HardwareAddr, 6)
	copy(mac[0:3], ip[8:11])
	copy(mac[3:6], ip[13:16])
	mac[0] ^= 0x02

	return mac, nil
}

// ExtractMAC looks into the inner most PeerAddr field in the RelayInfo header
// which contains the EUI-64 address of the client making the request, populated
// by the dhcp relay, it is possible to extract the mac address from that IP.
// If that fails, it looks for the MAC addressed embededded in the DUID.
// Note that this only works with type DuidLL and DuidLLT.
// If a mac address cannot be found an error will be returned.
func ExtractMAC(packet DHCPv6) (net.HardwareAddr, error) {
	msg := packet
	if packet.IsRelay() {
		inner, err := DecapsulateRelayIndex(packet, -1)
		if err != nil {
			return nil, err
		}
		relay := inner.(*RelayMessage)
		if _, mac := relay.Options.ClientLinkLayerAddress(); mac != nil {
			return mac, nil
		}
		if mac, err := GetMacAddressFromEUI64(relay.PeerAddr); err == nil {
			return mac, nil
		}
		msg, err = msg.(*RelayMessage).GetInnerMessage()
		if err != nil {
			return nil, err
		}
	}
	duid := msg.(*Message).Options.ClientID()
	if duid == nil {
		return nil, fmt.Errorf("client ID not found in packet")
	}
	if duid.LinkLayerAddr == nil {
		return nil, fmt.Errorf("failed to extract MAC")
	}
	return duid.LinkLayerAddr, nil
}
