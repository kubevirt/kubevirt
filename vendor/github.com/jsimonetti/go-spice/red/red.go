package red

import (
	"encoding"
	"errors"
)

// Various errors which may occur when attempting to marshal or unmarshal
// a SpicePacket to and from its binary form.
var (
	errInvalidPacket  = errors.New("invalid Spice packet")
	errInvalidVersion = errors.New("invalid version")
)

// SpicePacket is the interface used for passing around different kinds of packets.
type SpicePacket interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
	validate() error
	finish()
}

// Magic is the spice RED protocol magic bytes
var Magic = [4]uint8{0x52, 0x45, 0x44, 0x51}

const (
	// VersionMajor is the major version of the supported protocol
	VersionMajor uint32 = 2
	// VersionMinor is the minor version of the supported protocol
	VersionMinor uint32 = 2
)

//go:generate stringer -type=AuthMethod

// AuthMethod is the method used for authentication
type AuthMethod uint32

const (
	// AuthMethodSpice is the spice token based authentication method
	AuthMethodSpice AuthMethod = 1
	// AuthMethodSASL is the SASL authentication method
	AuthMethodSASL AuthMethod = 2
)

//go:generate stringer -type=ChannelType

// ChannelType is the packet channel type
type ChannelType uint8

// Channel types
const (
	ChannelMain      ChannelType = 1
	ChannelDisplay   ChannelType = 2
	ChannelInputs    ChannelType = 3
	ChannelCursor    ChannelType = 4
	ChannelPlayback  ChannelType = 5
	ChannelRecord    ChannelType = 6
	ChannelTunnel    ChannelType = 7
	ChannelSmartcard ChannelType = 8
	ChannelUSBRedir  ChannelType = 9
	ChannelPort      ChannelType = 10
	ChannelWebdav    ChannelType = 11
)

//go:generate stringer -type=ErrorCode

// ErrorCode return on error
type ErrorCode uint32

// Error codes
const (
	ErrorOk                  ErrorCode = 0
	ErrorError               ErrorCode = 1
	ErrorInvalidMagic        ErrorCode = 2
	ErrorInvalidData         ErrorCode = 3
	ErrorVersionMismatch     ErrorCode = 4
	ErrorNeedSecured         ErrorCode = 5
	ErrorNeedUnsecured       ErrorCode = 6
	ErrorPermissionDenied    ErrorCode = 7
	ErrorBadConnectionID     ErrorCode = 8
	ErrorChannelNotAvailable ErrorCode = 9
)

// TicketPubkeyBytes is the length of a ticket RSA public key
const TicketPubkeyBytes = 162

// ClientTicketBytes is the length of an encrypted Spice token
const ClientTicketBytes = 128

// PubKey is a red ticket public key
type PubKey [TicketPubkeyBytes]byte
