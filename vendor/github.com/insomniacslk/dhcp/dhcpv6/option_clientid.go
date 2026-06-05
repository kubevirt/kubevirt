package dhcpv6

import (
	"fmt"
)

// OptClientID represents a Client Identifier option as defined by RFC 3315
// Section 22.2.
func OptClientID(d DUID) Option {
	return &optClientID{d}
}

type optClientID struct {
	DUID
}

func (*optClientID) Code() OptionCode {
	return OptionClientID
}

func (op *optClientID) String() string {
	return fmt.Sprintf("%s: %s", op.Code(), op.DUID)
}

// FromBytes builds an optClientID structure from a sequence
// of bytes. The input data does not include option code and length
// bytes.
func (op *optClientID) FromBytes(data []byte) error {
	var err error
	op.DUID, err = DUIDFromBytes(data)
	return err
}
