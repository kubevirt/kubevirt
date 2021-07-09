package dhcpv6

import (
	"fmt"

	"github.com/u-root/u-root/pkg/uio"
)

// Option is an interface that all DHCPv6 options adhere to.
type Option interface {
	Code() OptionCode
	ToBytes() []byte
	String() string
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
	return fmt.Sprintf("%s -> %v", og.OptionCode, og.OptionData)
}

// ParseOption parses data according to the given code.
func ParseOption(code OptionCode, optData []byte) (Option, error) {
	// Parse a sequence of bytes as a single DHCPv6 option.
	// Returns the option structure, or an error if any.
	var (
		err error
		opt Option
	)
	switch code {
	case OptionClientID:
		opt, err = parseOptClientID(optData)
	case OptionServerID:
		opt, err = parseOptServerID(optData)
	case OptionIANA:
		opt, err = ParseOptIANA(optData)
	case OptionIATA:
		opt, err = ParseOptIATA(optData)
	case OptionIAAddr:
		opt, err = ParseOptIAAddress(optData)
	case OptionORO:
		var o optRequestedOption
		err = o.FromBytes(optData)
		opt = &o
	case OptionElapsedTime:
		opt, err = parseOptElapsedTime(optData)
	case OptionRelayMsg:
		opt, err = parseOptRelayMsg(optData)
	case OptionStatusCode:
		opt, err = ParseOptStatusCode(optData)
	case OptionUserClass:
		opt, err = ParseOptUserClass(optData)
	case OptionVendorClass:
		opt, err = ParseOptVendorClass(optData)
	case OptionVendorOpts:
		opt, err = ParseOptVendorOpts(optData)
	case OptionInterfaceID:
		opt, err = parseOptInterfaceID(optData)
	case OptionDNSRecursiveNameServer:
		opt, err = parseOptDNS(optData)
	case OptionDomainSearchList:
		opt, err = parseOptDomainSearchList(optData)
	case OptionIAPD:
		opt, err = ParseOptIAPD(optData)
	case OptionIAPrefix:
		opt, err = ParseOptIAPrefix(optData)
	case OptionInformationRefreshTime:
		opt, err = parseOptInformationRefreshTime(optData)
	case OptionRemoteID:
		opt, err = ParseOptRemoteID(optData)
	case OptionFQDN:
		opt, err = ParseOptFQDN(optData)
	case OptionBootfileURL:
		opt, err = parseOptBootFileURL(optData)
	case OptionBootfileParam:
		opt, err = parseOptBootFileParam(optData)
	case OptionClientArchType:
		opt, err = parseOptClientArchType(optData)
	case OptionNII:
		var o OptNetworkInterfaceID
		err = o.FromBytes(optData)
		opt = &o
	case OptionClientLinkLayerAddr:
		opt, err = parseOptClientLinkLayerAddress(optData)
	case OptionDHCPv4Msg:
		opt, err = ParseOptDHCPv4Msg(optData)
	case OptionDHCP4oDHCP6Server:
		opt, err = ParseOptDHCP4oDHCP6Server(optData)
	case Option4RD:
		opt, err = ParseOpt4RD(optData)
	case Option4RDMapRule:
		opt, err = ParseOpt4RDMapRule(optData)
	case Option4RDNonMapRule:
		opt, err = ParseOpt4RDNonMapRule(optData)
	default:
		opt = &OptionGeneric{OptionCode: code, OptionData: optData}
	}
	if err != nil {
		return nil, err
	}
	return opt, nil
}

// Options is a collection of options.
type Options []Option

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
	*o = make(Options, 0, 10)
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
