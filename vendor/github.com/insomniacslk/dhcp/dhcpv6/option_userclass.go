package dhcpv6

import (
	"fmt"
	"strings"

	"github.com/u-root/uio/uio"
)

// OptUserClass represent a DHCPv6 User Class option
//
// This module defines the OptUserClass structure.
// https://www.ietf.org/rfc/rfc3315.txt
type OptUserClass struct {
	UserClasses [][]byte
}

// Code returns the option code
func (op *OptUserClass) Code() OptionCode {
	return OptionUserClass
}

// ToBytes serializes the option and returns it as a sequence of bytes
func (op *OptUserClass) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	for _, uc := range op.UserClasses {
		buf.Write16(uint16(len(uc)))
		buf.WriteBytes(uc)
	}
	return buf.Data()
}

func (op *OptUserClass) String() string {
	ucStrings := make([]string, 0, len(op.UserClasses))
	for _, uc := range op.UserClasses {
		ucStrings = append(ucStrings, string(uc))
	}
	return fmt.Sprintf("%s: [%s]", op.Code(), strings.Join(ucStrings, ", "))
}

// FromBytes builds an OptUserClass structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func (op *OptUserClass) FromBytes(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: user class option must not be empty", uio.ErrBufferTooShort)
	}
	buf := uio.NewBigEndianBuffer(data)
	for buf.Has(2) {
		len := buf.Read16()
		op.UserClasses = append(op.UserClasses, buf.CopyN(int(len)))
	}
	return buf.FinError()
}
