package cache

import (
	"fmt"
	"net"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vishvananda/netlink"
)

var _ = Describe("DHCPConfig", func() {
	const ipv4Cidr = "10.0.0.200/24"
	const ipv6Cidr = "fd10:0:2::2/120"
	const mac = "de:ad:00:00:be:ef"
	const ipv4Gateway = "10.0.0.1"
	const mtu = 1450
	const vifName = "test-vif"

	routes := []netlink.Route{{
		Dst: &net.IPNet{
			IP:   net.ParseIP("0.0.0.0"),
			Mask: net.CIDRMask(32, 32),
		},
		Gw: net.ParseIP(ipv4Gateway),
	}}

	Context("String", func() {
		It("returns correct string representation", func() {
			dhcpConfig := createDummyDHCPConfig(vifName, ipv4Cidr, ipv4Gateway, "", mac, mtu, routes)
			Expect(dhcpConfig.String()).To(Equal(fmt.Sprintf("DHCPConfig: { Name: %s, IPv4: %s, IPv6: <nil>, MAC: %s, AdvertisingIPAddr: %s, MTU: %d, Gateway: %s, IPAMDisabled: false, Routes: %v}", vifName, ipv4Cidr, mac, ipv4Gateway, mtu, ipv4Gateway, &routes)))
		})
		It("returns correct string representation with ipv6", func() {
			dhcpConfig := createDummyDHCPConfig(vifName, ipv4Cidr, ipv4Gateway, ipv6Cidr, mac, mtu, routes)
			expRoutes := fmt.Sprintf("Routes: %v", &routes)
			Expect(dhcpConfig.String()).To(Equal(fmt.Sprintf("DHCPConfig: { Name: %s, IPv4: %s, IPv6: %s, MAC: %s, AdvertisingIPAddr: %s, MTU: %d, Gateway: %s, IPAMDisabled: false, %s}", vifName, ipv4Cidr, ipv6Cidr, mac, ipv4Gateway, mtu, ipv4Gateway, expRoutes)))
		})
		It("returns correct string representation when an IP is not defined", func() {
			gw := net.ParseIP(ipv4Gateway)
			macAddr, _ := net.ParseMAC(mac)
			dhcpConfig := DHCPConfig{
				Name:              vifName,
				MAC:               macAddr,
				AdvertisingIPAddr: gw,
				Mtu:               mtu,
				Gateway:           gw,
			}
			Expect(dhcpConfig.String()).To(Equal(fmt.Sprintf("DHCPConfig: { Name: %s, IPv4: <nil>, IPv6: <nil>, MAC: %s, AdvertisingIPAddr: %s, MTU: %d, Gateway: %s, IPAMDisabled: false, Routes: <nil>}", vifName, mac, ipv4Gateway, mtu, ipv4Gateway)))
		})
	})
})

func createDummyDHCPConfig(vifName, ipv4cidr, ipv4gateway, ipv6cidr, macStr string, mtu uint16, routes []netlink.Route) *DHCPConfig {
	mac, _ := net.ParseMAC(macStr)
	gw := net.ParseIP(ipv4gateway)
	dhcpConfig := &DHCPConfig{
		Name:              vifName,
		MAC:               mac,
		AdvertisingIPAddr: gw,
		Mtu:               mtu,
		Gateway:           gw,
		Routes:            &routes,
	}
	if ipv4cidr != "" {
		ipv4Addr, _ := netlink.ParseAddr(ipv4cidr)
		dhcpConfig.IP = *ipv4Addr
	}
	if ipv6cidr != "" {
		ipv6Addr, _ := netlink.ParseAddr(ipv6cidr)
		dhcpConfig.IPv6 = *ipv6Addr
	}

	return dhcpConfig
}
