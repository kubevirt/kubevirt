package dhcpv6

import (
	"fmt"

	"github.com/u-root/uio/uio"
)

// OptRemoteID implemens the Remote ID option as defined by RFC 4649.
type OptRemoteID struct {
	EnterpriseNumber uint32
	RemoteID         []byte
}

// Code implements Option.Code.
func (*OptRemoteID) Code() OptionCode {
	return OptionRemoteID
}

// ToBytes serializes this option to a byte stream.
func (op *OptRemoteID) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write32(op.EnterpriseNumber)
	buf.WriteBytes(op.RemoteID)
	return buf.Data()
}

func (op *OptRemoteID) String() string {
	return fmt.Sprintf("%s: {EnterpriseNumber=%d RemoteID=%#x}",
		op.Code(), op.EnterpriseNumber, op.RemoteID,
	)
}

// FromBytes builds an OptRemoteID structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func (op *OptRemoteID) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	op.EnterpriseNumber = buf.Read32()
	op.RemoteID = buf.ReadAll()
	return buf.FinError()
}
