package dhcpv6

import (
	"fmt"
	"time"

	"github.com/u-root/uio/uio"
)

// Duration is a duration as embedded in IA messages (IAPD, IANA, IATA).
type Duration struct {
	time.Duration
}

// Marshal encodes the time in uint32 seconds as defined by RFC 3315 for IANA
// messages.
func (d Duration) Marshal(buf *uio.Lexer) {
	buf.Write32(uint32(d.Duration.Round(time.Second) / time.Second))
}

// Unmarshal decodes time from uint32 seconds as defined by RFC 3315 for IANA
// messages.
func (d *Duration) Unmarshal(buf *uio.Lexer) {
	t := buf.Read32()
	d.Duration = time.Duration(t) * time.Second
}

// IdentityOptions implement the options allowed for IA_NA and IA_TA messages.
//
// The allowed options are identified in RFC 3315 Appendix B.
type IdentityOptions struct {
	Options
}

// Addresses returns the addresses assigned to the identity.
func (io IdentityOptions) Addresses() []*OptIAAddress {
	opts := io.Options.Get(OptionIAAddr)
	var iaAddrs []*OptIAAddress
	for _, o := range opts {
		iaAddrs = append(iaAddrs, o.(*OptIAAddress))
	}
	return iaAddrs
}

// OneAddress returns one address (of potentially many) assigned to the identity.
func (io IdentityOptions) OneAddress() *OptIAAddress {
	a := io.Addresses()
	if len(a) == 0 {
		return nil
	}
	return a[0]
}

// Status returns the status code associated with this option.
func (io IdentityOptions) Status() *OptStatusCode {
	opt := io.Options.GetOne(OptionStatusCode)
	if opt == nil {
		return nil
	}
	sc, ok := opt.(*OptStatusCode)
	if !ok {
		return nil
	}
	return sc
}

// OptIANA implements the identity association for non-temporary addresses
// option.
//
// This module defines the OptIANA structure.
// https://www.ietf.org/rfc/rfc3633.txt
type OptIANA struct {
	IaId    [4]byte
	T1      time.Duration
	T2      time.Duration
	Options IdentityOptions
}

func (op *OptIANA) Code() OptionCode {
	return OptionIANA
}

// ToBytes serializes IANA to DHCPv6 bytes.
func (op *OptIANA) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.WriteBytes(op.IaId[:])
	t1 := Duration{op.T1}
	t1.Marshal(buf)
	t2 := Duration{op.T2}
	t2.Marshal(buf)
	buf.WriteBytes(op.Options.ToBytes())
	return buf.Data()
}

func (op *OptIANA) String() string {
	return fmt.Sprintf("%s: {IAID=%#x T1=%v T2=%v Options=%v}",
		op.Code(), op.IaId, op.T1, op.T2, op.Options)
}

// LongString returns a multi-line string representation of IANA data.
func (op *OptIANA) LongString(indentSpace int) string {
	return fmt.Sprintf("%s: IAID=%#x T1=%s T2=%s Options=%s", op.Code(), op.IaId, op.T1, op.T2, op.Options.LongString(indentSpace))
}

// FromBytes builds an OptIANA structure from a sequence of bytes.  The
// input data does not include option code and length bytes.
func (op *OptIANA) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	buf.ReadBytes(op.IaId[:])

	var t1, t2 Duration
	t1.Unmarshal(buf)
	t2.Unmarshal(buf)
	op.T1 = t1.Duration
	op.T2 = t2.Duration

	if err := op.Options.FromBytes(buf.ReadAll()); err != nil {
		return err
	}
	return buf.FinError()
}
