package dhcpv6

import (
	"fmt"
	"net"
	"time"

	"github.com/u-root/u-root/pkg/uio"
)

// PrefixOptions are the options valid for use with IAPrefix option field.
//
// RFC 3633 states that it's just the StatusCode option.
//
// RFC 8415 Appendix C does not list the Status Code option as valid, but it
// does say that the previous text in RFC 8415 Section 21.22 supersedes that
// table. Section 21.22 does mention the Status Code option.
type PrefixOptions struct {
	Options
}

// Status returns the status code associated with this option.
func (po PrefixOptions) Status() *OptStatusCode {
	opt := po.Options.GetOne(OptionStatusCode)
	if opt == nil {
		return nil
	}
	sc, ok := opt.(*OptStatusCode)
	if !ok {
		return nil
	}
	return sc
}

// OptIAPrefix implements the IAPrefix option.
//
// This module defines the OptIAPrefix structure.
// https://www.ietf.org/rfc/rfc3633.txt
type OptIAPrefix struct {
	PreferredLifetime time.Duration
	ValidLifetime     time.Duration
	Prefix            *net.IPNet
	Options           PrefixOptions
}

func (op *OptIAPrefix) Code() OptionCode {
	return OptionIAPrefix
}

// ToBytes marshals this option according to RFC 3633, Section 10.
func (op *OptIAPrefix) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)

	t1 := Duration{op.PreferredLifetime}
	t1.Marshal(buf)
	t2 := Duration{op.ValidLifetime}
	t2.Marshal(buf)

	if op.Prefix != nil {
		// Even if Mask is nil, Size will return 0 without panicking.
		length, _ := op.Prefix.Mask.Size()
		buf.Write8(uint8(length))
		write16(buf, op.Prefix.IP)
	} else {
		buf.Write8(0)
		write16(buf, nil)
	}
	buf.WriteBytes(op.Options.ToBytes())
	return buf.Data()
}

func (op *OptIAPrefix) String() string {
	return fmt.Sprintf("IAPrefix: {PreferredLifetime=%v, ValidLifetime=%v, Prefix=%s, Options=%v}",
		op.PreferredLifetime, op.ValidLifetime, op.Prefix, op.Options)
}

// ParseOptIAPrefix an OptIAPrefix structure from a sequence of bytes. The
// input data does not include option code and length bytes.
func ParseOptIAPrefix(data []byte) (*OptIAPrefix, error) {
	buf := uio.NewBigEndianBuffer(data)
	var opt OptIAPrefix

	var t1, t2 Duration
	t1.Unmarshal(buf)
	t2.Unmarshal(buf)
	opt.PreferredLifetime = t1.Duration
	opt.ValidLifetime = t2.Duration

	length := buf.Read8()
	ip := net.IP(buf.CopyN(net.IPv6len))

	if length == 0 {
		opt.Prefix = nil
	} else {
		opt.Prefix = &net.IPNet{
			Mask: net.CIDRMask(int(length), 128),
			IP:   ip,
		}
	}
	if err := opt.Options.FromBytes(buf.ReadAll()); err != nil {
		return nil, err
	}
	return &opt, buf.FinError()
}
