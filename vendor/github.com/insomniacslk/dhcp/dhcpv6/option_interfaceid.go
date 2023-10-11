package dhcpv6

import (
	"fmt"
)

// OptInterfaceID returns an interface id option as defined by RFC 3315,
// Section 22.18.
func OptInterfaceID(id []byte) Option {
	return &optInterfaceID{ID: id}
}

type optInterfaceID struct {
	ID []byte
}

func (*optInterfaceID) Code() OptionCode {
	return OptionInterfaceID
}

func (op *optInterfaceID) ToBytes() []byte {
	return op.ID
}

func (op *optInterfaceID) String() string {
	return fmt.Sprintf("%s: %v", op.Code(), op.ID)
}

// FromBytes builds an optInterfaceID structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func (op *optInterfaceID) FromBytes(data []byte) error {
	op.ID = append([]byte(nil), data...)
	return nil
}
