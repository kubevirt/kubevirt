package dhcpv6

import (
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/rfc1035label"
	"github.com/u-root/uio/uio"
)

// NTPSuboptionSrvAddr is NTP_SUBOPTION_SRV_ADDR according to RFC 5908.
type NTPSuboptionSrvAddr net.IP

// Code returns the suboption code.
func (n *NTPSuboptionSrvAddr) Code() OptionCode {
	return NTPSuboptionSrvAddrCode
}

// ToBytes returns the byte serialization of the suboption.
func (n *NTPSuboptionSrvAddr) ToBytes() []byte {
	return net.IP(*n).To16()
}

func (n *NTPSuboptionSrvAddr) String() string {
	return fmt.Sprintf("Server Address: %s", net.IP(*n).String())
}

// FromBytes parses NTP server address from a byte slice p.
func (n *NTPSuboptionSrvAddr) FromBytes(p []byte) error {
	buf := uio.NewBigEndianBuffer(p)
	*n = NTPSuboptionSrvAddr(buf.CopyN(net.IPv6len))
	return buf.FinError()
}

// NTPSuboptionMCAddr is NTP_SUBOPTION_MC_ADDR according to RFC 5908.
type NTPSuboptionMCAddr net.IP

// Code returns the suboption code.
func (n *NTPSuboptionMCAddr) Code() OptionCode {
	return NTPSuboptionMCAddrCode
}

// ToBytes returns the byte serialization of the suboption.
func (n *NTPSuboptionMCAddr) ToBytes() []byte {
	return net.IP(*n).To16()
}

func (n *NTPSuboptionMCAddr) String() string {
	return fmt.Sprintf("Multicast Address: %s", net.IP(*n).String())
}

// FromBytes parses NTP multicast address from a byte slice p.
func (n *NTPSuboptionMCAddr) FromBytes(p []byte) error {
	buf := uio.NewBigEndianBuffer(p)
	*n = NTPSuboptionMCAddr(buf.CopyN(net.IPv6len))
	return buf.FinError()
}

// NTPSuboptionSrvFQDN is NTP_SUBOPTION_SRV_FQDN according to RFC 5908.
type NTPSuboptionSrvFQDN struct {
	rfc1035label.Labels
}

// Code returns the suboption code.
func (n *NTPSuboptionSrvFQDN) Code() OptionCode {
	return NTPSuboptionSrvFQDNCode
}

// ToBytes returns the byte serialization of the suboption.
func (n *NTPSuboptionSrvFQDN) ToBytes() []byte {
	return n.Labels.ToBytes()
}

func (n *NTPSuboptionSrvFQDN) String() string {
	return fmt.Sprintf("Server FQDN: %s", n.Labels.String())
}

// FromBytes parses an NTP server FQDN from a byte slice p.
func (n *NTPSuboptionSrvFQDN) FromBytes(p []byte) error {
	return n.Labels.FromBytes(p)
}

// NTPSuboptionSrvAddr is the value of NTP_SUBOPTION_SRV_ADDR according to RFC 5908.
const (
	NTPSuboptionSrvAddrCode = OptionCode(1)
	NTPSuboptionMCAddrCode  = OptionCode(2)
	NTPSuboptionSrvFQDNCode = OptionCode(3)
)

// parseNTPSuboption implements the OptionParser interface.
func parseNTPSuboption(code OptionCode, data []byte) (Option, error) {
	var o Option
	switch code {
	case NTPSuboptionSrvAddrCode:
		o = &NTPSuboptionSrvAddr{}
	case NTPSuboptionMCAddrCode:
		o = &NTPSuboptionMCAddr{}
	case NTPSuboptionSrvFQDNCode:
		o = &NTPSuboptionSrvFQDN{}
	default:
		o = &OptionGeneric{OptionCode: code}
	}
	return o, o.FromBytes(data)
}

// OptNTPServer is an option NTP server as defined by RFC 5908.
type OptNTPServer struct {
	Suboptions Options
}

// Code returns the option code
func (op *OptNTPServer) Code() OptionCode {
	return OptionNTPServer
}

// FromBytes parses a sequence of bytes into an OptNTPServer object.
func (op *OptNTPServer) FromBytes(data []byte) error {
	return op.Suboptions.FromBytesWithParser(data, parseNTPSuboption)
}

// ToBytes returns the option serialized to bytes.
func (op *OptNTPServer) ToBytes() []byte {
	return op.Suboptions.ToBytes()
}

func (op *OptNTPServer) String() string {
	return fmt.Sprintf("NTP: %v", op.Suboptions)
}
