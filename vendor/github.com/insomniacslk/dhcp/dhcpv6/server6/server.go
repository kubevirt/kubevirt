// Package server6 is a basic, extensible DHCPv6 server.
//
// To use the DHCPv6 server code you have to call NewServer with three arguments:
// - an interface to listen on,
// - an address to listen on, and
// - a handler function, that will be called every time a valid DHCPv6 packet is
//   received.
//
// The address to listen on is used to know IP address, port and optionally the
// scope to create and UDP socket to listen on for DHCPv6 traffic.
//
// The handler is a function that takes as input a packet connection, that can be
// used to reply to the client; a peer address, that identifies the client sending
// the request, and the DHCPv6 packet itself. Just implement your custom logic in
// the handler.
//
// Optionally, NewServer can receive options that will modify the server object.
// Some options already exist, for example WithConn. If this option is passed with
// a valid connection, the listening address argument is ignored.
//
// Example program:
//
//	package main
//
//	import (
//		"log"
//		"net"
//
//		"github.com/insomniacslk/dhcp/dhcpv6"
//		"github.com/insomniacslk/dhcp/dhcpv6/server6"
//	)
//
//	func handler(conn net.PacketConn, peer net.Addr, m dhcpv6.DHCPv6) {
//		// this function will just print the received DHCPv6 message, without replying
//		log.Print(m.Summary())
//	}
//
//	func main() {
//		laddr := net.UDPAddr{
//			IP:   net.ParseIP("::1"),
//			Port: 547,
//		}
//		server, err := server6.NewServer("eth0", &laddr, handler)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		// This never returns. If you want to do other stuff, dump it into a
//		// goroutine.
//		server.Serve()
//	}
//
package server6

import (
	"log"
	"net"
	"os"

	"github.com/insomniacslk/dhcp/dhcpv6"
	"golang.org/x/net/ipv6"
)

// Handler is a type that defines the handler function to be called every time a
// valid DHCPv6 message is received
type Handler func(conn net.PacketConn, peer net.Addr, m dhcpv6.DHCPv6)

// Server represents a DHCPv6 server object
type Server struct {
	conn    net.PacketConn
	handler Handler
	logger  Logger
}

// Serve starts the DHCPv6 server. The listener will run in background, and can
// be interrupted with `Server.Close`.
func (s *Server) Serve() error {
	s.logger.Printf("Server listening on %s", s.conn.LocalAddr())
	s.logger.Printf("Ready to handle requests")

	defer s.Close()
	for {
		rbuf := make([]byte, 4096) // FIXME this is bad
		n, peer, err := s.conn.ReadFrom(rbuf)
		if err != nil {
			s.logger.Printf("Error reading from packet conn: %v", err)
			return err
		}
		s.logger.Printf("Handling request from %v", peer)

		d, err := dhcpv6.FromBytes(rbuf[:n])
		if err != nil {
			s.logger.Printf("Error parsing DHCPv6 request: %v", err)
			continue
		}

		go s.handler(s.conn, peer, d)
	}
}

// Close sends a termination request to the server, and closes the UDP listener
func (s *Server) Close() error {
	return s.conn.Close()
}

// A ServerOpt configures a Server.
type ServerOpt func(s *Server)

// WithConn configures a server with the given connection.
func WithConn(conn net.PacketConn) ServerOpt {
	return func(s *Server) {
		s.conn = conn
	}
}

// NewServer initializes and returns a new Server object, listening on `addr`.
// * If `addr` is a multicast group, the group will be additionally joined
// * If `addr` is the wildcard address on the DHCPv6 server port (`[::]:547), the
//   multicast groups All_DHCP_Relay_Agents_and_Servers(`[ff02::1:2]`) and
//   All_DHCP_Servers(`[ff05::1:3]:547`) will be joined.
// * If `addr` is nil, IPv6 unspec on the DHCP server port is used and the above
//   behaviour applies
// If `WithConn` is used with a non-nil address, `addr` and `ifname` have
// no effect. In such case, joining the multicast group is the caller's
// responsibility.
func NewServer(ifname string, addr *net.UDPAddr, handler Handler, opt ...ServerOpt) (*Server, error) {
	s := &Server{
		handler: handler,
		logger:  EmptyLogger{},
	}

	for _, o := range opt {
		o(s)
	}
	if s.conn != nil {
		return s, nil
	}

	if addr == nil {
		addr = &net.UDPAddr{
			IP:   net.IPv6unspecified,
			Port: dhcpv6.DefaultServerPort,
		}
	}

	var (
		err   error
		iface *net.Interface
	)
	if ifname == "" {
		iface = nil
	} else {
		iface, err = net.InterfaceByName(ifname)
		if err != nil {
			return nil, err
		}
	}
	// no connection provided by the user, create a new one
	s.conn, err = NewIPv6UDPConn(ifname, addr)
	if err != nil {
		return nil, err
	}

	p := ipv6.NewPacketConn(s.conn)
	if addr.IP.IsMulticast() {
		if err := p.JoinGroup(iface, addr); err != nil {
			return nil, err
		}
	} else if (addr.IP == nil || addr.IP.IsUnspecified()) && addr.Port == dhcpv6.DefaultServerPort {
		// For wildcard addresses on the correct port, listen on both multicast
		// addresses defined in the RFC as a "default" behaviour
		for _, g := range []net.IP{dhcpv6.AllDHCPRelayAgentsAndServers, dhcpv6.AllDHCPServers} {
			group := net.UDPAddr{
				IP:   g,
				Port: dhcpv6.DefaultServerPort,
			}
			if err := p.JoinGroup(iface, &group); err != nil {
				return nil, err
			}

		}
	}

	return s, nil
}

// WithSummaryLogger logs one-line DHCPv6 message summaries when sent & received.
func WithSummaryLogger() ServerOpt {
	return func(s *Server) {
		s.logger = ShortSummaryLogger{
			Printfer: log.New(os.Stderr, "[dhcpv6] ", log.LstdFlags),
		}
	}
}

// WithDebugLogger logs multi-line full DHCPv6 messages when sent & received.
func WithDebugLogger() ServerOpt {
	return func(s *Server) {
		s.logger = DebugLogger{
			Printfer: log.New(os.Stderr, "[dhcpv6] ", log.LstdFlags),
		}
	}
}

// WithLogger set the logger (see interface Logger).
func WithLogger(newLogger Logger) ServerOpt {
	return func(s *Server) {
		s.logger = newLogger
	}
}
