package dhcpv6

import (
	"fmt"
	"strings"

	"github.com/u-root/uio/uio"
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
	return fmt.Sprintf("%s: {EnterpriseNumber=%d Data=[%s]}", op.Code(), op.EnterpriseNumber, strings.Join(vcStrings, ", "))
}

// FromBytes builds an OptVendorClass structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func (op *OptVendorClass) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	*op = OptVendorClass{}
	op.EnterpriseNumber = buf.Read32()
	for buf.Has(2) {
		len := buf.Read16()
		op.Data = append(op.Data, buf.CopyN(int(len)))
	}
	if len(op.Data) == 0 {
		return fmt.Errorf("%w: vendor class data should not be empty", uio.ErrBufferTooShort)
	}
	return buf.FinError()
}
