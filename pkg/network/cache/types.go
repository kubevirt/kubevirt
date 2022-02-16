package cache

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"

	v1 "kubevirt.io/api/core/v1"
)

type PodIfaceState int

const (
	PodIfaceNetworkPreparationPending PodIfaceState = iota
	PodIfaceNetworkPreparationStarted
	PodIfaceNetworkPreparationFinished
)

type PodCacheInterface struct {
	Iface  *v1.Interface `json:"iface,omitempty"`
	PodIP  string        `json:"podIP,omitempty"`
	PodIPs []string      `json:"podIPs,omitempty"`
	State  PodIfaceState `json:"networkState,omitempty"`
}

type DHCPConfig struct {
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
	Subdomain           string
}

func (d DHCPConfig) String() string {
	return fmt.Sprintf(
		"DHCPConfig: { Name: %s, IPv4: %s, IPv6: %s, MAC: %s, AdvertisingIPAddr: %s, MTU: %d, Gateway: %s, IPAMDisabled: %t}",
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
