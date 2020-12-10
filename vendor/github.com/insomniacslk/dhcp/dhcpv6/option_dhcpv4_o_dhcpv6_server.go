package dhcpv6

import (
	"fmt"
	"net"

	"github.com/u-root/u-root/pkg/uio"
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
	return fmt.Sprintf("OptDHCP4oDHCP6Server{4o6-servers=%v}", op.DHCP4oDHCP6Servers)
}

// ParseOptDHCP4oDHCP6Server builds an OptDHCP4oDHCP6Server structure
// from a sequence of bytes. The input data does not include option code and length
// bytes.
func ParseOptDHCP4oDHCP6Server(data []byte) (*OptDHCP4oDHCP6Server, error) {
	var opt OptDHCP4oDHCP6Server
	buf := uio.NewBigEndianBuffer(data)
	for buf.Has(net.IPv6len) {
		opt.DHCP4oDHCP6Servers = append(opt.DHCP4oDHCP6Servers, buf.CopyN(net.IPv6len))
	}
	return &opt, buf.FinError()
}
