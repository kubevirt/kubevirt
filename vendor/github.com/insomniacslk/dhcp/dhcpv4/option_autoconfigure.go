package dhcpv4

import (
	"fmt"
)

// AutoConfiguration implements encoding and decoding functions for a
// byte enumeration as used in RFC 2563, Section 2.
type AutoConfiguration byte

const (
	DoNotAutoConfigure AutoConfiguration = 0
	AutoConfigure      AutoConfiguration = 1
)

var autoConfigureToString = map[AutoConfiguration]string{
	DoNotAutoConfigure: "DoNotAutoConfigure",
	AutoConfigure:      "AutoConfigure",
}

// ToBytes returns a serialized stream of bytes for this option.
func (o AutoConfiguration) ToBytes() []byte {
	return []byte{byte(o)}
}

// String returns a human-readable string for this option.
func (o AutoConfiguration) String() string {
	s := autoConfigureToString[o]
	if s != "" {
		return s
	}
	return fmt.Sprintf("UNKNOWN (%d)", byte(o))
}

// FromBytes parses a a single byte into AutoConfiguration
func (o *AutoConfiguration) FromBytes(data []byte) error {
	if len(data) == 1 {
		*o = AutoConfiguration(data[0])
		return nil
	}
	return fmt.Errorf("Invalid buffer length (%d)", len(data))
}

// GetByte parses any single-byte option
func GetByte(code OptionCode, o Options) (byte, error) {
	data := o.Get(code)
	if data == nil {
		return 0, fmt.Errorf("option not present")
	}
	if len(data) != 1 {
		return 0, fmt.Errorf("Invalid buffer length (%d)", len(data))
	}
	return data[0], nil
}

// OptAutoConfigure returns a new AutoConfigure option.
//
// The AutoConfigure option is described by RFC 2563, Section 2.
func OptAutoConfigure(autoconf AutoConfiguration) Option {
	return Option{Code: OptionAutoConfigure, Value: autoconf}
}
