package dhcpv6

import (
	"fmt"
	"net"

	"github.com/u-root/uio/uio"
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
	return fmt.Sprintf("%s: %v", op.Code(), op.NameServers)
}

// FromBytes builds an optDNS structure from a sequence of bytes. The input
// data does not include option code and length bytes.
func (op *optDNS) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	for buf.Has(net.IPv6len) {
		op.NameServers = append(op.NameServers, buf.CopyN(net.IPv6len))
	}
	return buf.FinError()
}
