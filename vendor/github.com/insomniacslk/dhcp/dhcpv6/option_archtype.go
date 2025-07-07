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
	return fmt.Sprintf("%s: %s", op.Code(), op.Archs)
}

func (op *optClientArchType) FromBytes(p []byte) error {
	return op.Archs.FromBytes(p)
}
