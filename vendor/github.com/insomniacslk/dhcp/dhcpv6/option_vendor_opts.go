package dhcpv6

import (
	"fmt"

	"github.com/u-root/u-root/pkg/uio"
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
	return fmt.Sprintf("OptVendorOpts{enterprisenum=%v, vendorOpts=%v}",
		op.EnterpriseNumber, op.VendorOpts,
	)
}

// ParseOptVendorOpts builds an OptVendorOpts structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func ParseOptVendorOpts(data []byte) (*OptVendorOpts, error) {
	var opt OptVendorOpts
	buf := uio.NewBigEndianBuffer(data)
	opt.EnterpriseNumber = buf.Read32()
	if err := opt.VendorOpts.FromBytesWithParser(buf.ReadAll(), vendParseOption); err != nil {
		return nil, err
	}
	return &opt, buf.FinError()
}

// vendParseOption builds a GenericOption from a slice of bytes
// We cannot use the existing ParseOption function in options.go because the
// sub-options include codes specific to each vendor. There are overlaps in these
// codes with RFC standard codes.
func vendParseOption(code OptionCode, data []byte) (Option, error) {
	return &OptionGeneric{OptionCode: code, OptionData: data}, nil
}
