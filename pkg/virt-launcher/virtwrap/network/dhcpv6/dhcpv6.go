package dhcpv6

import (
	"fmt"
	"net"
	"time"

	"github.com/vishvananda/netlink"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

const (
	infiniteLease = 999 * 24 * time.Hour
)

type DHCPv6Handler struct {
	serverIP      net.IP
	clientIP      net.IP
	clientMAC     net.HardwareAddr
	leaseDuration time.Duration
}

func SingleClientDHCPv6Server(
	clientIP net.IP,
	serverIface string,
	dnsIPs [][]byte,
	routes *[]netlink.Route,
	searchDomains []string,
	mtu uint16,
	customDHCPOptions *v1.DHCPOptions) error {

	log.Log.Info("Starting SingleClientDHCPv6Server")

	handler := &DHCPv6Handler{
		clientIP:      clientIP,
		clientMAC:     clientMAC,
		serverIP:      serverIP.To4(),
		leaseDuration: infiniteLease,
	}

	s, err := server6.NewServer("", nil, handler.ServeDHCPv6)
	if err != nil {
		return fmt.Errorf("couldn't create dhcpv6 server: %v", err)
	}
	go func() {
		_ = s.Serve()
	}()

	return nil
}

func (h *DHCPv6Handler) ServeDHCPv6(conn net.PacketConn, peer net.Addr, m dhcpv6.DHCPv6) {
	log.Log.V(4).Info("Serving a new request")

	// TODO how can we make sure the request is from the vm? Is filtering requests arrived to the bridge interface enough?

	msg := m.(*dhcpv6.Message)
	adv, err := dhcpv6.NewAdvertiseFromSolicit(msg)
	if err != nil {
		log.Log.V(4).Errorf("NewAdvertiseFromSolicit failed: %v", err)
		return
	}
	if _, err := conn.WriteTo(adv.ToBytes(), peer); err != nil {
		log.Log.V(4).Errorf("Cannot reply to client: %v", err)
	}
}
