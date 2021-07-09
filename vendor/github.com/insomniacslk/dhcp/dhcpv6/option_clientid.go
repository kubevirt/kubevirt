package dhcpv6

import (
	"fmt"
)

// OptClientID represents a Client Identifier option as defined by RFC 3315
// Section 22.2.
func OptClientID(d Duid) Option {
	return &optClientID{d}
}

type optClientID struct {
	Duid
}

func (*optClientID) Code() OptionCode {
	return OptionClientID
}

func (op *optClientID) String() string {
	return fmt.Sprintf("ClientID: %v", op.Duid.String())
}

// parseOptClientID builds an OptClientId structure from a sequence
// of bytes. The input data does not include option code and length
// bytes.
func parseOptClientID(data []byte) (*optClientID, error) {
	cid, err := DuidFromBytes(data)
	if err != nil {
		return nil, err
	}
	return &optClientID{*cid}, nil
}
