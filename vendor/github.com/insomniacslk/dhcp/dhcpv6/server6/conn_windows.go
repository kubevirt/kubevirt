// +build windows

package server6

import (
	"errors"
	"net"
)

// NewIPv6UDPConn fails on Windows. Use WithConn() to pass the connection.
func NewIPv6UDPConn(iface string, addr *net.UDPAddr) (*net.UDPConn, error) {
	return nil, errors.New("not implemented on Windows")
}
