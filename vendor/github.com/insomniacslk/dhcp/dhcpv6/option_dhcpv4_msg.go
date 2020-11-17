package dhcpv6

import (
	"fmt"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

// OptDHCPv4Msg represents a OptionDHCPv4Msg option
//
// This module defines the OptDHCPv4Msg structure.
// https://www.ietf.org/rfc/rfc7341.txt
type OptDHCPv4Msg struct {
	Msg *dhcpv4.DHCPv4
}

// Code returns the option code
func (op *OptDHCPv4Msg) Code() OptionCode {
	return OptionDHCPv4Msg
}

// ToBytes returns the option serialized to bytes.
func (op *OptDHCPv4Msg) ToBytes() []byte {
	return op.Msg.ToBytes()
}

func (op *OptDHCPv4Msg) String() string {
	return fmt.Sprintf("OptDHCPv4Msg{%v}", op.Msg)
}

// ParseOptDHCPv4Msg builds an OptDHCPv4Msg structure
// from a sequence of bytes. The input data does not include option code and length
// bytes.
func ParseOptDHCPv4Msg(data []byte) (*OptDHCPv4Msg, error) {
	var opt OptDHCPv4Msg
	var err error
	opt.Msg, err = dhcpv4.FromBytes(data)
	return &opt, err
}
