package dhcpv6

import (
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/iana"
	"github.com/u-root/u-root/pkg/uio"
)

// OptClientLinkLayerAddress implements OptionClientLinkLayerAddr option.
// https://tools.ietf.org/html/rfc6939
func OptClientLinkLayerAddress(ht iana.HWType, lla net.HardwareAddr) *optClientLinkLayerAddress {
	return &optClientLinkLayerAddress{LinkLayerType: ht, LinkLayerAddress: lla}
}

type optClientLinkLayerAddress struct {
	LinkLayerType    iana.HWType
	LinkLayerAddress net.HardwareAddr
}

// Code returns the option code.
func (op *optClientLinkLayerAddress) Code() OptionCode {
	return OptionClientLinkLayerAddr
}

// ToBytes serializes the option and returns it as a sequence of bytes
func (op *optClientLinkLayerAddress) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write16(uint16(op.LinkLayerType))
	buf.WriteBytes(op.LinkLayerAddress)
	return buf.Data()
}

func (op *optClientLinkLayerAddress) String() string {
	return fmt.Sprintf("ClientLinkLayerAddress: Type=%s LinkLayerAddress=%s", op.LinkLayerType, op.LinkLayerAddress)
}

// parseOptClientLinkLayerAddress deserializes from bytes
// to build an optClientLinkLayerAddress structure.
func parseOptClientLinkLayerAddress(data []byte) (*optClientLinkLayerAddress, error) {
	var opt optClientLinkLayerAddress
	buf := uio.NewBigEndianBuffer(data)
	opt.LinkLayerType = iana.HWType(buf.Read16())
	opt.LinkLayerAddress = buf.ReadAll()
	return &opt, buf.FinError()
}
