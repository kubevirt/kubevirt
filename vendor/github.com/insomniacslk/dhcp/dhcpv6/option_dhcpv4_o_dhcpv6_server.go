package dhcpv6

import (
	"fmt"
	"net"

	"github.com/u-root/uio/uio"
)

// OptDHCP4oDHCP6Server represents a OptionDHCP4oDHCP6Server option
//
// This module defines the OptDHCP4oDHCP6Server structure.
// https://www.ietf.org/rfc/rfc7341.txt
type OptDHCP4oDHCP6Server struct {
	DHCP4oDHCP6Servers []net.IP
}

// Code returns the option code
func (op *OptDHCP4oDHCP6Server) Code() OptionCode {
	return OptionDHCP4oDHCP6Server
}

// ToBytes returns the option serialized to bytes.
func (op *OptDHCP4oDHCP6Server) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	for _, addr := range op.DHCP4oDHCP6Servers {
		buf.WriteBytes(addr.To16())
	}
	return buf.Data()
}

func (op *OptDHCP4oDHCP6Server) String() string {
	return fmt.Sprintf("%s: %v", op.Code(), op.DHCP4oDHCP6Servers)
}

// FromBytes builds an OptDHCP4oDHCP6Server structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func (op *OptDHCP4oDHCP6Server) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	for buf.Has(net.IPv6len) {
		op.DHCP4oDHCP6Servers = append(op.DHCP4oDHCP6Servers, buf.CopyN(net.IPv6len))
	}
	return buf.FinError()
}
