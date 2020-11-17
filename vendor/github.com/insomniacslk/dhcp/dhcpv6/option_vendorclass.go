package dhcpv6

import (
	"errors"
	"fmt"
	"strings"

	"github.com/u-root/u-root/pkg/uio"
)

// OptVendorClass represents a DHCPv6 Vendor Class option
type OptVendorClass struct {
	EnterpriseNumber uint32
	Data             [][]byte
}

// Code returns the option code
func (op *OptVendorClass) Code() OptionCode {
	return OptionVendorClass
}

// ToBytes serializes the option and returns it as a sequence of bytes
func (op *OptVendorClass) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write32(op.EnterpriseNumber)
	for _, data := range op.Data {
		buf.Write16(uint16(len(data)))
		buf.WriteBytes(data)
	}
	return buf.Data()
}

// String returns a string representation of the VendorClass data
func (op *OptVendorClass) String() string {
	vcStrings := make([]string, 0)
	for _, data := range op.Data {
		vcStrings = append(vcStrings, string(data))
	}
	return fmt.Sprintf("OptVendorClass{enterprisenum=%d, data=[%s]}", op.EnterpriseNumber, strings.Join(vcStrings, ", "))
}

// ParseOptVendorClass builds an OptVendorClass structure from a sequence of
// bytes. The input data does not include option code and length bytes.
func ParseOptVendorClass(data []byte) (*OptVendorClass, error) {
	var opt OptVendorClass
	buf := uio.NewBigEndianBuffer(data)
	opt.EnterpriseNumber = buf.Read32()
	for buf.Has(2) {
		len := buf.Read16()
		opt.Data = append(opt.Data, buf.CopyN(int(len)))
	}
	if len(opt.Data) < 1 {
		return nil, errors.New("ParseOptVendorClass: at least one vendor class data is required")
	}
	return &opt, buf.FinError()
}
