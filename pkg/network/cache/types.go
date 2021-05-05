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

type VIF struct {
	Name         string
	IP           netlink.Addr
	IPv6         netlink.Addr
	MAC          net.HardwareAddr
	Gateway      net.IP
	GatewayIpv6  net.IP
	Routes       *[]netlink.Route
	Mtu          uint16
	IPAMDisabled bool
}

func (vif VIF) String() string {
	return fmt.Sprintf(
		"VIF: { Name: %s, IP: %s, Mask: %s, IPv6: %s, MAC: %s, Gateway: %s, MTU: %d, IPAMDisabled: %t}",
		vif.Name,
		vif.IP.IP,
		vif.IP.Mask,
		vif.IPv6,
		vif.MAC,
		vif.Gateway,
		vif.Mtu,
		vif.IPAMDisabled,
	)
}
