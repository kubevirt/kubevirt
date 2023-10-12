package dhcpv6

import (
	"fmt"

	"github.com/insomniacslk/dhcp/iana"
	"github.com/u-root/uio/uio"
)

// OptStatusCode represents a DHCPv6 Status Code option
//
// This module defines the OptStatusCode structure.
// https://www.ietf.org/rfc/rfc3315.txt
type OptStatusCode struct {
	StatusCode    iana.StatusCode
	StatusMessage string
}

// Code returns the option code.
func (op *OptStatusCode) Code() OptionCode {
	return OptionStatusCode
}

// ToBytes serializes the option and returns it as a sequence of bytes.
func (op *OptStatusCode) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write16(uint16(op.StatusCode))
	buf.WriteBytes([]byte(op.StatusMessage))
	return buf.Data()
}

// String returns a human-readable option.
func (op *OptStatusCode) String() string {
	return fmt.Sprintf("%s: {Code=%s (%d); Message=%s}",
		op.Code(), op.StatusCode, op.StatusCode, op.StatusMessage)
}

// FromBytes builds an OptStatusCode structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func (op *OptStatusCode) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	op.StatusCode = iana.StatusCode(buf.Read16())
	op.StatusMessage = string(buf.ReadAll())
	return buf.FinError()
}
