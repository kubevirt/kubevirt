package dhcpv6

import (
	"fmt"

	"github.com/u-root/uio/uio"
)

// OptVendorOpts represents a DHCPv6 Status Code option
//
// This module defines the OptVendorOpts structure.
// https://tools.ietf.org/html/rfc3315#section-22.17
type OptVendorOpts struct {
	EnterpriseNumber uint32
	VendorOpts       Options
}

// Code returns the option code
func (op *OptVendorOpts) Code() OptionCode {
	return OptionVendorOpts
}

// ToBytes serializes the option and returns it as a sequence of bytes
func (op *OptVendorOpts) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write32(op.EnterpriseNumber)
	buf.WriteData(op.VendorOpts.ToBytes())
	return buf.Data()
}

// String returns a string representation of the VendorOpts data
func (op *OptVendorOpts) String() string {
	return fmt.Sprintf("%s: {EnterpriseNumber=%v VendorOptions=%v}", op.Code(), op.EnterpriseNumber, op.VendorOpts)
}

// LongString returns a string representation of the VendorOpts data
func (op *OptVendorOpts) LongString(indent int) string {
	return fmt.Sprintf("%s: EnterpriseNumber=%v VendorOptions=%s", op.Code(), op.EnterpriseNumber, op.VendorOpts.LongString(indent))
}

// FromBytes builds an OptVendorOpts structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func (op *OptVendorOpts) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	op.EnterpriseNumber = buf.Read32()
	if err := op.VendorOpts.FromBytesWithParser(buf.ReadAll(), vendParseOption); err != nil {
		return err
	}
	return buf.FinError()
}

// vendParseOption builds a GenericOption from a slice of bytes
// We cannot use the existing ParseOption function in options.go because the
// sub-options include codes specific to each vendor. There are overlaps in these
// codes with RFC standard codes.
func vendParseOption(code OptionCode, data []byte) (Option, error) {
	return &OptionGeneric{OptionCode: code, OptionData: data}, nil
}
