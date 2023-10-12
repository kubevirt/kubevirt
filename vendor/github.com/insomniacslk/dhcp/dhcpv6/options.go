package dhcpv6

import (
	"fmt"
	"strings"

	"github.com/u-root/uio/uio"
)

// Option is an interface that all DHCPv6 options adhere to.
type Option interface {
	Code() OptionCode
	ToBytes() []byte
	String() string
	FromBytes([]byte) error
}

type OptionGeneric struct {
	OptionCode OptionCode
	OptionData []byte
}

func (og *OptionGeneric) Code() OptionCode {
	return og.OptionCode
}

func (og *OptionGeneric) ToBytes() []byte {
	return og.OptionData
}

func (og *OptionGeneric) String() string {
	if len(og.OptionData) == 0 {
		return og.OptionCode.String()
	}
	return fmt.Sprintf("%s: %v", og.OptionCode, og.OptionData)
}

// FromBytes resets OptionData to p.
func (og *OptionGeneric) FromBytes(p []byte) error {
	og.OptionData = append([]byte(nil), p...)
	return nil
}

// ParseOption parses data according to the given code.
//
// Parse a sequence of bytes as a single DHCPv6 option.
// Returns the option structure, or an error if any.
func ParseOption(code OptionCode, optData []byte) (Option, error) {
	var opt Option
	switch code {
	case OptionClientID:
		opt = &optClientID{}
	case OptionServerID:
		opt = &optServerID{}
	case OptionIANA:
		opt = &OptIANA{}
	case OptionIATA:
		opt = &OptIATA{}
	case OptionIAAddr:
		opt = &OptIAAddress{}
	case OptionORO:
		opt = &optRequestedOption{}
	case OptionElapsedTime:
		opt = &optElapsedTime{}
	case OptionRelayMsg:
		opt = &optRelayMsg{}
	case OptionStatusCode:
		opt = &OptStatusCode{}
	case OptionUserClass:
		opt = &OptUserClass{}
	case OptionVendorClass:
		opt = &OptVendorClass{}
	case OptionVendorOpts:
		opt = &OptVendorOpts{}
	case OptionInterfaceID:
		opt = &optInterfaceID{}
	case OptionDNSRecursiveNameServer:
		opt = &optDNS{}
	case OptionDomainSearchList:
		opt = &optDomainSearchList{}
	case OptionIAPD:
		opt = &OptIAPD{}
	case OptionIAPrefix:
		opt = &OptIAPrefix{}
	case OptionInformationRefreshTime:
		opt = &optInformationRefreshTime{}
	case OptionRemoteID:
		opt = &OptRemoteID{}
	case OptionFQDN:
		opt = &OptFQDN{}
	case OptionNTPServer:
		opt = &OptNTPServer{}
	case OptionBootfileURL:
		opt = &optBootFileURL{}
	case OptionBootfileParam:
		opt = &optBootFileParam{}
	case OptionClientArchType:
		opt = &optClientArchType{}
	case OptionNII:
		opt = &OptNetworkInterfaceID{}
	case OptionClientLinkLayerAddr:
		opt = &optClientLinkLayerAddress{}
	case OptionDHCPv4Msg:
		opt = &OptDHCPv4Msg{}
	case OptionDHCP4oDHCP6Server:
		opt = &OptDHCP4oDHCP6Server{}
	case Option4RD:
		opt = &Opt4RD{}
	case Option4RDMapRule:
		opt = &Opt4RDMapRule{}
	case Option4RDNonMapRule:
		opt = &Opt4RDNonMapRule{}
	case OptionRelayPort:
		opt = &optRelayPort{}
	default:
		opt = &OptionGeneric{OptionCode: code}
	}
	return opt, opt.FromBytes(optData)
}

type longStringer interface {
	LongString(spaceIndent int) string
}

// Options is a collection of options.
type Options []Option

// LongString prints options with indentation of at least spaceIndent spaces.
func (o Options) LongString(spaceIndent int) string {
	indent := strings.Repeat(" ", spaceIndent)
	var s strings.Builder
	if len(o) == 0 {
		s.WriteString("[]")
	} else {
		s.WriteString("[\n")
		for _, opt := range o {
			s.WriteString(indent)
			s.WriteString("  ")
			if ls, ok := opt.(longStringer); ok {
				s.WriteString(ls.LongString(spaceIndent + 2))
			} else {
				s.WriteString(opt.String())
			}
			s.WriteString("\n")
		}
		s.WriteString(indent)
		s.WriteString("]")
	}
	return s.String()
}

// Get returns all options matching the option code.
func (o Options) Get(code OptionCode) []Option {
	var ret []Option
	for _, opt := range o {
		if opt.Code() == code {
			ret = append(ret, opt)
		}
	}
	return ret
}

// GetOne returns the first option matching the option code.
func (o Options) GetOne(code OptionCode) Option {
	for _, opt := range o {
		if opt.Code() == code {
			return opt
		}
	}
	return nil
}

// Add appends one option.
func (o *Options) Add(option Option) {
	*o = append(*o, option)
}

// Del deletes all options matching the option code.
func (o *Options) Del(code OptionCode) {
	newOpts := make(Options, 0, len(*o))
	for _, opt := range *o {
		if opt.Code() != code {
			newOpts = append(newOpts, opt)
		}
	}
	*o = newOpts
}

// Update replaces the first option of the same type as the specified one.
func (o *Options) Update(option Option) {
	for idx, opt := range *o {
		if opt.Code() == option.Code() {
			(*o)[idx] = option
			// don't look further
			return
		}
	}
	// if not found, add it
	o.Add(option)
}

// ToBytes marshals all options to bytes.
func (o Options) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	for _, opt := range o {
		buf.Write16(uint16(opt.Code()))

		val := opt.ToBytes()
		buf.Write16(uint16(len(val)))
		buf.WriteBytes(val)
	}
	return buf.Data()
}

// FromBytes reads data into o and returns an error if the options are not a
// valid serialized representation of DHCPv6 options per RFC 3315.
func (o *Options) FromBytes(data []byte) error {
	return o.FromBytesWithParser(data, ParseOption)
}

// OptionParser is a function signature for option parsing
type OptionParser func(code OptionCode, data []byte) (Option, error)

// FromBytesWithParser parses Options from byte sequences using the parsing
// function that is passed in as a paremeter
func (o *Options) FromBytesWithParser(data []byte, parser OptionParser) error {
	if *o == nil {
		*o = make(Options, 0, 10)
	}
	if len(data) == 0 {
		// no options, no party
		return nil
	}

	buf := uio.NewBigEndianBuffer(data)
	for buf.Has(4) {
		code := OptionCode(buf.Read16())
		length := int(buf.Read16())

		// Consume, but do not Copy. Each parser will make a copy of
		// pertinent data.
		optData := buf.Consume(length)

		opt, err := parser(code, optData)
		if err != nil {
			return err
		}
		*o = append(*o, opt)
	}
	return buf.FinError()
}
