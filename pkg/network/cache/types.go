package cache

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/client-go/api/v1"
)

type PodCacheInterface struct {
	Iface  *v1.Interface `json:"iface,omitempty"`
	PodIP  string        `json:"podIP,omitempty"`
	PodIPs []string      `json:"podIPs,omitempty"`
}

type DhcpConfig struct {
	Name                string
	IP                  netlink.Addr
	IPv6                netlink.Addr
	MAC                 net.HardwareAddr
	AdvertisingIPAddr   net.IP
	AdvertisingIPv6Addr net.IP
	Routes              *[]netlink.Route
	Mtu                 uint16
	IPAMDisabled        bool
	Gateway             net.IP
}

func (d DhcpConfig) String() string {
	return fmt.Sprintf(
		"DhcpConfig: { Name: %s, IPv4: %s, IPv6: %s, MAC: %s, AdvertisingIPAddr: %s, MTU: %d, Gateway: %s, IPAMDisabled: %t}",
		d.Name,
		d.IP,
		d.IPv6,
		d.MAC,
		d.AdvertisingIPAddr,
		d.Mtu,
		d.Gateway,
		d.IPAMDisabled,
	)
}
