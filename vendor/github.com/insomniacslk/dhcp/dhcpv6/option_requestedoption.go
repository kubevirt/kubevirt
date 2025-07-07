package dhcpv6

import (
	"fmt"
	"strings"

	"github.com/u-root/uio/uio"
)

// OptionCodes are a collection of option codes.
type OptionCodes []OptionCode

// Add adds an option to the list, ignoring duplicates.
func (o *OptionCodes) Add(c OptionCode) {
	if !o.Contains(c) {
		*o = append(*o, c)
	}
}

// Contains returns whether the option codes contain c.
func (o OptionCodes) Contains(c OptionCode) bool {
	for _, oo := range o {
		if oo == c {
			return true
		}
	}
	return false
}

// ToBytes implements Option.ToBytes.
func (o OptionCodes) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	for _, ro := range o {
		buf.Write16(uint16(ro))
	}
	return buf.Data()
}

func (o OptionCodes) String() string {
	names := make([]string, 0, len(o))
	for _, code := range o {
		names = append(names, code.String())
	}
	return strings.Join(names, ", ")
}

// FromBytes populates o from binary-encoded data.
func (o *OptionCodes) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	for buf.Has(2) {
		o.Add(OptionCode(buf.Read16()))
	}
	return buf.FinError()
}

// OptRequestedOption implements the requested options option as defined by RFC
// 3315 Section 22.7.
func OptRequestedOption(o ...OptionCode) Option {
	return &optRequestedOption{
		OptionCodes: o,
	}
}

type optRequestedOption struct {
	OptionCodes
}

// Code implements Option.Code.
func (*optRequestedOption) Code() OptionCode {
	return OptionORO
}

func (op *optRequestedOption) String() string {
	return fmt.Sprintf("%s: %s", op.Code(), op.OptionCodes)
}
