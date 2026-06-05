package dhcpv6

import (
	"fmt"
	"strings"

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
	return fmt.Sprintf("%s: %v", op.Code(), op.Msg)
}

// LongString returns a multi-line string representation of DHCPv4 data.
func (op *OptDHCPv4Msg) LongString(indent int) string {
	summary := op.Msg.Summary()
	ind := strings.Repeat(" ", indent+2)
	if strings.Contains(summary, "\n") {
		summary = strings.Replace(summary, "\n  ", "\n"+ind, -1)
	}
	ind = strings.Repeat(" ", indent)
	return fmt.Sprintf("%s: {%v%s}", op.Code(), summary, ind)
}

// FromBytes builds an OptDHCPv4Msg structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func (op *OptDHCPv4Msg) FromBytes(data []byte) error {
	var err error
	op.Msg, err = dhcpv4.FromBytes(data)
	return err
}
