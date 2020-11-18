package dhcpv6

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv6"

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
	serverIface   *net.Interface
}

func SingleClientDHCPv6Server(
	clientIP net.IP,
	serverIfaceName string,
	dnsIPs [][]byte,
	routes *[]netlink.Route,
	searchDomains []string,
	mtu uint16,
	customDHCPOptions *v1.DHCPOptions) error {

	log.Log.Info("Starting SingleClientDHCPv6Server")

	iface, err := net.InterfaceByName(serverIfaceName)
	if err != nil {
		log.Log.Infof("DHCPv6 - couldn't get the server interface: %v", err)
	}

	handler := &DHCPv6Handler{
		clientIP:      clientIP,
		leaseDuration: infiniteLease,
		serverIface:   iface,
	}

	conn, err := handler.createConnection()
	if err != nil {
		return fmt.Errorf("couldn't create connection for dhcpv6 server: %v", err)
	}

	s, err := server6.NewServer("", nil, handler.ServeDHCPv6, server6.WithConn(conn))
	if err != nil {
		return fmt.Errorf("couldn't create dhcpv6 server: %v", err)
	}
	go func() {
		_ = s.Serve()
	}()

	return nil
}

func (h *DHCPv6Handler) createConnection() (*FilteredConn, error) {
	// no connection provided by the user, create a new one
	addr := &net.UDPAddr{
		IP:   net.IPv6unspecified,
		Port: dhcpv6.DefaultServerPort,
	}
	udpConn, err := server6.NewIPv6UDPConn("", addr)
	if err != nil {
		return nil, err
	}

	packetConn := ipv6.NewPacketConn(udpConn)
	if err := packetConn.SetControlMessage(ipv6.FlagInterface, true); err != nil {
		return nil, err
	}

	for _, groupAddrerss := range []net.IP{dhcpv6.AllDHCPRelayAgentsAndServers, dhcpv6.AllDHCPServers} {
		group := net.UDPAddr{
			IP:   groupAddrerss,
			Port: dhcpv6.DefaultServerPort,
		}
		if err := packetConn.JoinGroup(h.serverIface, &group); err != nil {
			return nil, err
		}
	}

	return &FilteredConn{packetConn: packetConn, ifIndex: h.serverIface.Index}, err
}

func (h *DHCPv6Handler) ServeDHCPv6(conn net.PacketConn, peer net.Addr, m dhcpv6.DHCPv6) {
	log.Log.V(4).Info("DHCPv6 serving a new request")

	// TODO how can we make sure the request is from the vm? Is filtering requests arrived to the bridge interface enough?

	msg := m.(*dhcpv6.Message)

	var response *dhcpv6.Message
	var err error

	optIAAddress := dhcpv6.OptIAAddress{IPv6Addr: h.clientIP, PreferredLifetime: h.leaseDuration, ValidLifetime: h.leaseDuration}

	duid := dhcpv6.Duid{Type: dhcpv6.DUID_LL, HwType: iana.HWTypeEthernet, LinkLayerAddr: h.serverIface.HardwareAddr}

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
