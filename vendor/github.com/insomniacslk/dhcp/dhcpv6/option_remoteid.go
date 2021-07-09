package dhcpv6

import (
	"fmt"

	"github.com/u-root/u-root/pkg/uio"
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
	return fmt.Sprintf("RemoteID: EnterpriseNumber %d RemoteID %v",
		op.EnterpriseNumber, op.RemoteID,
	)
}

// ParseOptRemoteId builds an OptRemoteId structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func ParseOptRemoteID(data []byte) (*OptRemoteID, error) {
	var opt OptRemoteID
	buf := uio.NewBigEndianBuffer(data)
	opt.EnterpriseNumber = buf.Read32()
	opt.RemoteID = buf.ReadAll()
	return &opt, buf.FinError()
}
