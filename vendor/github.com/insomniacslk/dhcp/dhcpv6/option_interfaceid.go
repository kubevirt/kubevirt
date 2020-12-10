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
	return fmt.Sprintf("InterfaceID: %v", op.ID)
}

// build an optInterfaceID structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func parseOptInterfaceID(data []byte) (*optInterfaceID, error) {
	var opt optInterfaceID
	opt.ID = append([]byte(nil), data...)
	return &opt, nil
}
