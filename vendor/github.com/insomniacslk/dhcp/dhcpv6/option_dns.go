package dhcpv6

import (
	"fmt"
	"net"

	"github.com/u-root/u-root/pkg/uio"
)

// OptDNS returns a DNS Recursive Name Server option as defined by RFC 3646.
func OptDNS(ip ...net.IP) Option {
	return &optDNS{NameServers: ip}
}

type optDNS struct {
	NameServers []net.IP
}

// Code returns the option code
func (op *optDNS) Code() OptionCode {
	return OptionDNSRecursiveNameServer
}

// ToBytes returns the option serialized to bytes.
func (op *optDNS) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	for _, ns := range op.NameServers {
		buf.WriteBytes(ns.To16())
	}
	return buf.Data()
}

func (op *optDNS) String() string {
	return fmt.Sprintf("DNS: %v", op.NameServers)
}

// parseOptDNS builds an optDNS structure
// from a sequence of bytes. The input data does not include option code and length
// bytes.
func parseOptDNS(data []byte) (*optDNS, error) {
	var opt optDNS
	buf := uio.NewBigEndianBuffer(data)
	for buf.Has(net.IPv6len) {
		opt.NameServers = append(opt.NameServers, buf.CopyN(net.IPv6len))
	}
	return &opt, buf.FinError()
}
