package dhcpv6

import "net"

// Default ports
const (
	DefaultClientPort = 546
	DefaultServerPort = 547
)

// Default multicast groups
var (
	AllDHCPRelayAgentsAndServers = net.ParseIP("ff02::1:2")
	AllDHCPServers               = net.ParseIP("ff05::1:3")
)
