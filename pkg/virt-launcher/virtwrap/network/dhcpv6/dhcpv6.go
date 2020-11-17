package dhcpv6

import (
	"fmt"
	"net"
	"time"

	"github.com/vishvananda/netlink"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"
	"github.com/insomniacslk/dhcp/iana"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
)

const (
	infiniteLease = 999 * 24 * time.Hour
)

type DHCPv6Handler struct {
	clientIP      net.IP
	leaseDuration time.Duration
	serverIface   string
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
		leaseDuration: infiniteLease,
		serverIface:   serverIface,
	}

	s, err := server6.NewServer(serverIface, nil, handler.ServeDHCPv6)
	if err != nil {
		return fmt.Errorf("couldn't create dhcpv6 server: %v", err)
	}
	go func() {
		_ = s.Serve()
	}()

	return nil
}

func (h *DHCPv6Handler) ServeDHCPv6(conn net.PacketConn, peer net.Addr, m dhcpv6.DHCPv6) {
	log.Log.V(4).Info("DHCPv6 serving a new request")

	// TODO how can we make sure the request is from the vm? Is filtering requests arrived to the bridge interface enough?

	msg := m.(*dhcpv6.Message)

	var response *dhcpv6.Message

	optIAAddress := dhcpv6.OptIAAddress{IPv6Addr: h.clientIP, PreferredLifetime: h.leaseDuration, ValidLifetime: h.leaseDuration}

	iface, err := net.InterfaceByName(h.serverIface)
	if err != nil {
		log.Log.V(4).Info("DHCPv6 - couldn't get the server interface")
		return
	}
	duid := dhcpv6.Duid{Type: dhcpv6.DUID_LL, HwType: iana.HWTypeEthernet, LinkLayerAddr: iface.HardwareAddr}

	switch msg.Type() {
	case dhcpv6.MessageTypeSolicit:
		log.Log.V(4).Info("DHCPv6 - the request has message type Solicit")
		response, err = dhcpv6.NewAdvertiseFromSolicit(msg, dhcpv6.WithIANA(optIAAddress), dhcpv6.WithServerID(duid))
	default:
		log.Log.V(4).Info("DHCPv6 - non Solicit request recieved")
		response, err = dhcpv6.NewReplyFromMessage(msg, dhcpv6.WithIANA(optIAAddress), dhcpv6.WithServerID(duid))
	}

	if err != nil {
		log.Log.V(4).Errorf("DHCPv6 failed sending a response to the client: %v", err)
		return
	}

	if _, err := conn.WriteTo(response.ToBytes(), peer); err != nil {
		log.Log.V(4).Errorf("DHCPv6 cannot reply to client: %v", err)
	}
}
