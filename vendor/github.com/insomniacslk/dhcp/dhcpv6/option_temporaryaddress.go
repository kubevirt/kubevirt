package dhcpv6

import (
	"fmt"

	"github.com/u-root/uio/uio"
)

// OptIATA implements the identity association for non-temporary addresses
// option.
//
// This module defines the OptIATA structure, as defined by RFC 8415 Section
// 21.5.
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
	return fmt.Sprintf("%s: {IAID=%#x, Options=%v}", op.Code(), op.IaId, op.Options)
}

// LongString returns a multi-line string representation of IATA data.
func (op *OptIATA) LongString(indentSpace int) string {
	return fmt.Sprintf("%s: IAID=%#x Options=%v", op.Code(), op.IaId, op.Options.LongString(indentSpace))
}

// FromBytes builds an OptIATA structure from a sequence of bytes.  The input
// data does not include option code and length bytes.
func (op *OptIATA) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	buf.ReadBytes(op.IaId[:])

	if err := op.Options.FromBytes(buf.ReadAll()); err != nil {
		return err
	}
	return buf.FinError()
}
