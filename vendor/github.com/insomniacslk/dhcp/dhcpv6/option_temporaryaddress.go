package dhcpv6

import (
	"fmt"

	"github.com/u-root/u-root/pkg/uio"
)

// OptIATA implements the identity association for non-temporary addresses
// option.
//
// This module defines the OptIATA structure.
// https://www.ietf.org/rfc/rfc8415.txt
type OptIATA struct {
	IaId    [4]byte
	Options IdentityOptions
}

// Code returns the option code for an IA_TA
func (op *OptIATA) Code() OptionCode {
	return OptionIATA
}

// ToBytes serializes IATA to DHCPv6 bytes.
func (op *OptIATA) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.WriteBytes(op.IaId[:])
	buf.WriteBytes(op.Options.ToBytes())
	return buf.Data()
}

func (op *OptIATA) String() string {
	return fmt.Sprintf("IATA: {IAID=%v, options=%v}",
		op.IaId, op.Options)
}

// ParseOptIATA builds an OptIATA structure from a sequence of bytes.  The
// input data does not include option code and length bytes.
func ParseOptIATA(data []byte) (*OptIATA, error) {
	var opt OptIATA
	buf := uio.NewBigEndianBuffer(data)
	buf.ReadBytes(opt.IaId[:])

	if err := opt.Options.FromBytes(buf.ReadAll()); err != nil {
		return nil, err
	}
	return &opt, buf.FinError()
}
