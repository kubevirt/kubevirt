package dhcpv6

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/net/ipv6"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/server6"
	"github.com/insomniacslk/dhcp/iana"

	"kubevirt.io/client-go/log"
)

const (
	infiniteLease = 999 * 24 * time.Hour
)

type DHCPv6Handler struct {
	clientIP    net.IP
	serverIface *net.Interface
	modifiers   []dhcpv6.Modifier
}

func SingleClientDHCPv6Server(
	clientIP net.IP,
	serverIfaceName string) error {

	log.Log.Info("Starting SingleClientDHCPv6Server")

	iface, err := net.InterfaceByName(serverIfaceName)
	if err != nil {
		log.Log.Infof("DHCPv6 - couldn't get the server interface: %v", err)
	}

	modifiers, err := prepareDHCPv6Modifiers(clientIP, iface.HardwareAddr)

	if err != nil {
		log.Log.Infof("DHCPv6 - couldn't prepare modifiers: %v", err)
	}
	handler := &DHCPv6Handler{
		clientIP:    clientIP,
		serverIface: iface,
		modifiers:   modifiers,
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

	// TODO if we extend the server to support bridge binding, we need to filter out non-vm requests

	response, err := h.buildResponse(m)
	if err != nil {
		log.Log.V(4).Errorf("DHCPv6 failed building a response to the client: %v", err)
	}

	if _, err := conn.WriteTo(response.ToBytes(), peer); err != nil {
		log.Log.V(4).Errorf("DHCPv6 failed sending a response to the client: %v", err)
	}
}

func (h *DHCPv6Handler) buildResponse(msg dhcpv6.DHCPv6) (*dhcpv6.Message, error) {
	var response *dhcpv6.Message
	var err error

	dhcpv6Msg := msg.(*dhcpv6.Message)
	switch dhcpv6Msg.Type() {
	case dhcpv6.MessageTypeSolicit:
		log.Log.V(4).Info("DHCPv6 - the request has message type Solicit")
		if dhcpv6Msg.GetOneOption(dhcpv6.OptionRapidCommit) == nil {
			response, err = dhcpv6.NewAdvertiseFromSolicit(dhcpv6Msg, h.modifiers...)
		} else {
			log.Log.V(4).Info("DHCPv6 - replying with rapid commit")
			response, err = dhcpv6.NewReplyFromMessage(dhcpv6Msg, h.modifiers...)
		}
	default:
		log.Log.V(4).Info("DHCPv6 - non Solicit request received")
		response, err = dhcpv6.NewReplyFromMessage(dhcpv6Msg, h.modifiers...)
	}

	if err != nil {
		return nil, err
	}

	ianaRequest := dhcpv6Msg.Options.OneIANA()
	ianaResponse := response.Options.OneIANA()
	ianaResponse.IaId = ianaRequest.IaId
	response.UpdateOption(ianaResponse)
	return response, nil
}

func prepareDHCPv6Modifiers(
	clientIP net.IP,
	serverInterfaceMac net.HardwareAddr) ([]dhcpv6.Modifier, error) {

	var modifiers []dhcpv6.Modifier

	optIAAddress := dhcpv6.OptIAAddress{IPv6Addr: clientIP, PreferredLifetime: infiniteLease, ValidLifetime: infiniteLease}
	modifiers = append(modifiers, dhcpv6.WithIANA(optIAAddress))

	duid := dhcpv6.Duid{Type: dhcpv6.DUID_LL, HwType: iana.HWTypeEthernet, LinkLayerAddr: serverInterfaceMac}
	modifiers = append(modifiers, dhcpv6.WithServerID(duid))

	return modifiers, nil
}
