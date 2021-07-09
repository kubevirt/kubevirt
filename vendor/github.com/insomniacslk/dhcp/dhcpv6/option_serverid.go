package dhcpv6

import (
	"fmt"
)

// OptServerID represents a Server Identifier option as defined by RFC 3315
// Section 22.1.
func OptServerID(d Duid) Option {
	return &optServerID{d}
}

type optServerID struct {
	Duid
}

func (*optServerID) Code() OptionCode {
	return OptionServerID
}

func (op *optServerID) String() string {
	return fmt.Sprintf("ServerID: %v", op.Duid.String())
}

// parseOptServerID builds an optServerID structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func parseOptServerID(data []byte) (*optServerID, error) {
	sid, err := DuidFromBytes(data)
	if err != nil {
		return nil, err
	}
	return &optServerID{*sid}, nil
}
