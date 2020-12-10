package dhcpv6

import (
	"fmt"
	"net"

	"github.com/u-root/u-root/pkg/uio"
)

// Opt4RD represents a 4RD option. It is only a container for 4RD_*_RULE options
type Opt4RD Options

// Code returns the Option Code for this option
func (op *Opt4RD) Code() OptionCode {
	return Option4RD
}

// ToBytes serializes this option
func (op *Opt4RD) ToBytes() []byte {
	return (*Options)(op).ToBytes()
}

// String returns a human-readable representation of the option
func (op *Opt4RD) String() string {
	return fmt.Sprintf("Opt4RD{%v}", (*Options)(op))
}

// ParseOpt4RD builds an Opt4RD structure from a sequence of bytes.
// The input data does not include option code and length bytes
func ParseOpt4RD(data []byte) (*Opt4RD, error) {
	var opt Options
	err := opt.FromBytes(data)
	return (*Opt4RD)(&opt), err
}

// Opt4RDMapRule represents a 4RD Mapping Rule option
// The option is described in https://tools.ietf.org/html/rfc7600#section-4.9
// The 4RD mapping rules are described in https://tools.ietf.org/html/rfc7600#section-4.2
type Opt4RDMapRule struct {
	// Prefix4 is the IPv4 prefix mapped by this rule
	Prefix4 net.IPNet
	// Prefix6 is the IPv6 prefix mapped by this rule
	Prefix6 net.IPNet
	// EABitsLength is the number of bits of an address used in constructing the mapped address
	EABitsLength uint8
	// WKPAuthorized determines if well-known ports are assigned to addresses in an A+P mapping
	// It can only be set if the length of Prefix4 + EABits > 32
	WKPAuthorized bool
}

const (
	// opt4RDWKPAuthorizedMask is the mask for the WKPAuthorized flag in its
	// byte in Opt4RDMapRule
	opt4RDWKPAuthorizedMask = 1 << 7
	// opt4RDHubAndSpokeMask is the mask for the HubAndSpoke flag in its
	// byte in Opt4RDNonMapRule
	opt4RDHubAndSpokeMask = 1 << 7
	// opt4RDTrafficClassMask is the mask for the TrafficClass flag in its
	// byte in Opt4RDNonMapRule
	opt4RDTrafficClassMask = 1 << 0
)

// Code returns the option code representing this option
func (op *Opt4RDMapRule) Code() OptionCode { return Option4RDMapRule }

// ToBytes serializes this option
func (op *Opt4RDMapRule) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	p4Len, _ := op.Prefix4.Mask.Size()
	p6Len, _ := op.Prefix6.Mask.Size()
	buf.Write8(uint8(p4Len))
	buf.Write8(uint8(p6Len))
	buf.Write8(op.EABitsLength)
	if op.WKPAuthorized {
		buf.Write8(opt4RDWKPAuthorizedMask)
	} else {
		buf.Write8(0)
	}
	if op.Prefix4.IP.To4() == nil {
		// The API prevents us from returning an error here
		// We just write zeros instead, which is pretty bad behaviour
		buf.Write32(0)
	} else {
		buf.WriteBytes(op.Prefix4.IP.To4())
	}
	if op.Prefix6.IP.To16() == nil {
		buf.Write64(0)
		buf.Write64(0)
	} else {
		buf.WriteBytes(op.Prefix6.IP.To16())
	}
	return buf.Data()
}

// String returns a human-readable description of this option
func (op *Opt4RDMapRule) String() string {
	return fmt.Sprintf("Opt4RDMapRule{Prefix4=%s, Prefix6=%s, EA-Bits=%d, WKPAuthorized=%t}",
		op.Prefix4.String(), op.Prefix6.String(), op.EABitsLength, op.WKPAuthorized)
}

// ParseOpt4RDMapRule builds an Opt4RDMapRule structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func ParseOpt4RDMapRule(data []byte) (*Opt4RDMapRule, error) {
	var opt Opt4RDMapRule
	buf := uio.NewBigEndianBuffer(data)
	opt.Prefix4.Mask = net.CIDRMask(int(buf.Read8()), 32)
	opt.Prefix6.Mask = net.CIDRMask(int(buf.Read8()), 128)
	opt.EABitsLength = buf.Read8()
	opt.WKPAuthorized = (buf.Read8() & opt4RDWKPAuthorizedMask) != 0
	opt.Prefix4.IP = net.IP(buf.CopyN(net.IPv4len))
	opt.Prefix6.IP = net.IP(buf.CopyN(net.IPv6len))
	return &opt, buf.FinError()
}

// Opt4RDNonMapRule represents 4RD parameters other than mapping rules
type Opt4RDNonMapRule struct {
	// HubAndSpoke is whether the network topology is hub-and-spoke or meshed
	HubAndSpoke bool
	// TrafficClass is an optional 8-bit tunnel traffic class identifier
	TrafficClass *uint8
	// DomainPMTU is the Path MTU for this 4RD domain
	DomainPMTU uint16
}

// Code returns the option code for this option
func (op *Opt4RDNonMapRule) Code() OptionCode {
	return Option4RDNonMapRule
}

// ToBytes serializes this option
func (op *Opt4RDNonMapRule) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	var flags uint8
	var trafficClassValue uint8
	if op.HubAndSpoke {
		flags |= opt4RDHubAndSpokeMask
	}
	if op.TrafficClass != nil {
		flags |= opt4RDTrafficClassMask
		trafficClassValue = *op.TrafficClass
	}

	buf.Write8(flags)
	buf.Write8(trafficClassValue)
	buf.Write16(op.DomainPMTU)

	return buf.Data()
}

// String returns a human-readable description of this option
func (op *Opt4RDNonMapRule) String() string {
	var tClass interface{} = false
	if op.TrafficClass != nil {
		tClass = *op.TrafficClass
	}

	return fmt.Sprintf("Opt4RDNonMapRule{HubAndSpoke=%t, TrafficClass=%v, DomainPMTU=%d}",
		op.HubAndSpoke, tClass, op.DomainPMTU)
}

// ParseOpt4RDNonMapRule builds an Opt4RDNonMapRule structure from a sequence of bytes.
// The input data does not include option code and length bytes
func ParseOpt4RDNonMapRule(data []byte) (*Opt4RDNonMapRule, error) {
	var opt Opt4RDNonMapRule
	buf := uio.NewBigEndianBuffer(data)
	flags := buf.Read8()

	opt.HubAndSpoke = flags&opt4RDHubAndSpokeMask != 0

	tClass := buf.Read8()
	if flags&opt4RDTrafficClassMask != 0 {
		opt.TrafficClass = &tClass
	}

	opt.DomainPMTU = buf.Read16()

	return &opt, buf.FinError()
}
