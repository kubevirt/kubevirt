package dhcpv6

import (
	"fmt"
	"net"
	"time"

	"github.com/u-root/u-root/pkg/uio"
)

// AddressOptions are options valid for the IAAddress option field.
//
// RFC 8415 Appendix C lists only the Status Code option as valid.
type AddressOptions struct {
	Options
}

// Status returns the status code associated with this option.
func (ao AddressOptions) Status() *OptStatusCode {
	opt := ao.Options.GetOne(OptionStatusCode)
	if opt == nil {
		return nil
	}
	sc, ok := opt.(*OptStatusCode)
	if !ok {
		return nil
	}
	return sc
}

// OptIAAddress represents an OptionIAAddr.
//
// This module defines the OptIAAddress structure.
// https://www.ietf.org/rfc/rfc3633.txt
type OptIAAddress struct {
	IPv6Addr          net.IP
	PreferredLifetime time.Duration
	ValidLifetime     time.Duration
	Options           AddressOptions
}

// Code returns the option's code
func (op *OptIAAddress) Code() OptionCode {
	return OptionIAAddr
}

// ToBytes serializes the option and returns it as a sequence of bytes
func (op *OptIAAddress) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	write16(buf, op.IPv6Addr)

	t1 := Duration{op.PreferredLifetime}
	t1.Marshal(buf)
	t2 := Duration{op.ValidLifetime}
	t2.Marshal(buf)

	buf.WriteBytes(op.Options.ToBytes())
	return buf.Data()
}

func (op *OptIAAddress) String() string {
	return fmt.Sprintf("IAAddress: IP=%v PreferredLifetime=%v ValidLifetime=%v Options=%v",
		op.IPv6Addr, op.PreferredLifetime, op.ValidLifetime, op.Options)
}

// ParseOptIAAddress builds an OptIAAddress structure from a sequence
// of bytes. The input data does not include option code and length
// bytes.
func ParseOptIAAddress(data []byte) (*OptIAAddress, error) {
	var opt OptIAAddress
	buf := uio.NewBigEndianBuffer(data)
	opt.IPv6Addr = net.IP(buf.CopyN(net.IPv6len))

	var t1, t2 Duration
	t1.Unmarshal(buf)
	t2.Unmarshal(buf)
	opt.PreferredLifetime = t1.Duration
	opt.ValidLifetime = t2.Duration

	if err := opt.Options.FromBytes(buf.ReadAll()); err != nil {
		return nil, err
	}
	return &opt, buf.FinError()
}
