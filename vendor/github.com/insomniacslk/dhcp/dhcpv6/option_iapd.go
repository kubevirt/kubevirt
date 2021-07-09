package dhcpv6

import (
	"fmt"
	"time"

	"github.com/u-root/u-root/pkg/uio"
)

// PDOptions are options used with the IAPD (prefix delegation) option.
//
// RFC 3633 describes that IA_PD-options may contain the IAPrefix option and
// the StatusCode option.
type PDOptions struct {
	Options
}

// Prefixes are the prefixes associated with this delegation.
func (po PDOptions) Prefixes() []*OptIAPrefix {
	opts := po.Options.Get(OptionIAPrefix)
	pre := make([]*OptIAPrefix, 0, len(opts))
	for _, o := range opts {
		if iap, ok := o.(*OptIAPrefix); ok {
			pre = append(pre, iap)
		}
	}
	return pre
}

// Status returns the status code associated with this option.
func (po PDOptions) Status() *OptStatusCode {
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

// OptIAPD implements the identity association for prefix
// delegation option defined by RFC 3633, Section 9.
type OptIAPD struct {
	IaId    [4]byte
	T1      time.Duration
	T2      time.Duration
	Options PDOptions
}

// Code returns the option code
func (op *OptIAPD) Code() OptionCode {
	return OptionIAPD
}

// ToBytes serializes the option and returns it as a sequence of bytes
func (op *OptIAPD) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.WriteBytes(op.IaId[:])

	t1 := Duration{op.T1}
	t1.Marshal(buf)
	t2 := Duration{op.T2}
	t2.Marshal(buf)

	buf.WriteBytes(op.Options.ToBytes())
	return buf.Data()
}

// String returns a string representation of the OptIAPD data
func (op *OptIAPD) String() string {
	return fmt.Sprintf("IAPD: {IAID=%v, t1=%v, t2=%v, Options=[%v]}",
		op.IaId, op.T1, op.T2, op.Options)
}

// ParseOptIAPD builds an OptIAPD structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func ParseOptIAPD(data []byte) (*OptIAPD, error) {
	var opt OptIAPD
	buf := uio.NewBigEndianBuffer(data)
	buf.ReadBytes(opt.IaId[:])

	var t1, t2 Duration
	t1.Unmarshal(buf)
	t2.Unmarshal(buf)
	opt.T1 = t1.Duration
	opt.T2 = t2.Duration

	if err := opt.Options.FromBytes(buf.ReadAll()); err != nil {
		return nil, err
	}
	return &opt, buf.FinError()
}
