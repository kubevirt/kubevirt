// +build !windows

package server6

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/insomniacslk/dhcp/interfaces"
	"golang.org/x/sys/unix"
)

// NewIPv6UDPConn returns a UDPv6-only connection bound to both the interface and port
// given based on a IPv6 DGRAM socket.
// As a bonus, you can actually listen on a multicast address instead of being punted to the wildcard
//
// The interface must already be configured.
func NewIPv6UDPConn(iface string, addr *net.UDPAddr) (*net.UDPConn, error) {
	fd, err := unix.Socket(unix.AF_INET6, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	if err != nil {
		return nil, fmt.Errorf("cannot get a UDP socket: %v", err)
	}
	f := os.NewFile(uintptr(fd), "")
	// net.FilePacketConn dups the FD, so we have to close this in any case.
	defer f.Close()

	// Allow broadcasting.
	if err := unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_V6ONLY, 1); err != nil {
		if errno, ok := err.(unix.Errno); !ok {
			return nil, fmt.Errorf("unexpected socket error: %v", err)
		} else if errno != unix.ENOPROTOOPT { // Unsupported on some OSes (but in that case v6only is default), so we ignore ENOPROTOOPT
			return nil, fmt.Errorf("cannot bind socket v6only %v", err)
		}
	}
	// Allow reusing the addr to aid debugging.
	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1); err != nil {
		return nil, fmt.Errorf("cannot set reuseaddr on socket: %v", err)
	}
	if len(iface) != 0 {
		// Bind directly to the interface.
		if err := interfaces.BindToInterface(fd, iface); err != nil {
			if errno, ok := err.(unix.Errno); ok && errno == unix.EACCES {
				// Return a more helpful error message in this (fairly common) case
				return nil, errors.New("Cannot bind to interface without CAP_NET_RAW or root permissions. " +
					"Restart with elevated privilege, or run without specifying an interface to bind to all available interfaces.")
			}
			return nil, fmt.Errorf("cannot bind to interface %s: %v", iface, err)
		}
	}

	if addr == nil {
		return nil, errors.New("An address to listen on needs to be specified")
	}
	// Bind to the port.
	saddr := unix.SockaddrInet6{Port: addr.Port}
	copy(saddr.Addr[:], addr.IP)
	if err := unix.Bind(fd, &saddr); err != nil {
		return nil, fmt.Errorf("cannot bind to address %v: %v", addr, err)
	}

	conn, err := net.FilePacketConn(f)
	if err != nil {
		return nil, err
	}
	udpconn, ok := conn.(*net.UDPConn)
	if !ok {
		return nil, errors.New("BUG(dhcp6): incorrect socket type, expected UDP")
	}
	return udpconn, nil
}
