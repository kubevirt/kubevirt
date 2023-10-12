// This module defines the optRelayPort structure.
// https://www.ietf.org/rfc/rfc8357.txt

package dhcpv6

import (
	"fmt"

	"github.com/u-root/uio/uio"
)

// OptRelayPort specifies an UDP port to use for the downstream relay
func OptRelayPort(port uint16) Option {
	return &optRelayPort{DownstreamSourcePort: port}
}

type optRelayPort struct {
	DownstreamSourcePort uint16
}

func (op *optRelayPort) Code() OptionCode {
	return OptionRelayPort
}

func (op *optRelayPort) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write16(op.DownstreamSourcePort)
	return buf.Data()
}

func (op *optRelayPort) String() string {
	return fmt.Sprintf("%s: %d", op.Code(), op.DownstreamSourcePort)
}

// FromBytes build an optRelayPort structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func (op *optRelayPort) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	op.DownstreamSourcePort = buf.Read16()
	return buf.FinError()
}
