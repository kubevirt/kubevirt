package dhcpv6

import (
	"fmt"

	"github.com/insomniacslk/dhcp/iana"
)

// OptClientArchType represents an option CLIENT_ARCH_TYPE.
//
// This module defines the OptClientArchType structure.
// https://www.ietf.org/rfc/rfc5970.txt
func OptClientArchType(a ...iana.Arch) Option {
	return &optClientArchType{Archs: a}
}

type optClientArchType struct {
	iana.Archs
}

func (op *optClientArchType) Code() OptionCode {
	return OptionClientArchType
}

func (op optClientArchType) String() string {
	return fmt.Sprintf("ClientArchType: %s", op.Archs.String())
}

// parseOptClientArchType builds an OptClientArchType structure from
// a sequence of bytes The input data does not include option code and
// length bytes.
func parseOptClientArchType(data []byte) (*optClientArchType, error) {
	var opt optClientArchType
	return &opt, opt.FromBytes(data)
}
