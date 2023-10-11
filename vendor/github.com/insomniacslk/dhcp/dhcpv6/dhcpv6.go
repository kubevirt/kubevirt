// Package dhcpv6 provides encoding and decoding of DHCPv6 messages and
// options.
package dhcpv6

import (
	"fmt"
	"net"

	"github.com/u-root/uio/uio"
)

type DHCPv6 interface {
	Type() MessageType
	ToBytes() []byte
	String() string
	Summary() string
	LongString(indent int) string
	IsRelay() bool

	// GetInnerMessage returns the innermost encapsulated DHCPv6 message.
	//
	// If it is already a message, it will be returned. If it is a relay
	// message, the encapsulated message will be recursively extracted.
	GetInnerMessage() (*Message, error)

	GetOption(code OptionCode) []Option
	GetOneOption(code OptionCode) Option
	AddOption(Option)
	UpdateOption(Option)
}

// Modifier defines the signature for functions that can modify DHCPv6
// structures. This is used to simplify packet manipulation
type Modifier func(d DHCPv6)

// MessageFromBytes parses a DHCPv6 message from a byte stream.
func MessageFromBytes(data []byte) (*Message, error) {
	buf := uio.NewBigEndianBuffer(data)
	messageType := MessageType(buf.Read8())

	if messageType == MessageTypeRelayForward || messageType == MessageTypeRelayReply {
		return nil, fmt.Errorf("wrong message type")
	}

	d := &Message{
		MessageType: messageType,
	}
	buf.ReadBytes(d.TransactionID[:])
	if buf.Error() != nil {
		return nil, fmt.Errorf("failed to parse DHCPv6 header: %w", buf.Error())
	}
	if err := d.Options.FromBytes(buf.Data()); err != nil {
		return nil, err
	}
	return d, nil
}

// RelayMessageFromBytes parses a relay message from a byte stream.
func RelayMessageFromBytes(data []byte) (*RelayMessage, error) {
	buf := uio.NewBigEndianBuffer(data)
	messageType := MessageType(buf.Read8())

	if messageType != MessageTypeRelayForward && messageType != MessageTypeRelayReply {
		return nil, fmt.Errorf("wrong message type")
	}

	d := &RelayMessage{
		MessageType: messageType,
		HopCount:    buf.Read8(),
	}
	d.LinkAddr = net.IP(buf.CopyN(net.IPv6len))
	d.PeerAddr = net.IP(buf.CopyN(net.IPv6len))

	if buf.Error() != nil {
		return nil, fmt.Errorf("Error parsing RelayMessage header: %v", buf.Error())
	}
	// TODO: fail if no OptRelayMessage is present.
	if err := d.Options.FromBytes(buf.Data()); err != nil {
		return nil, err
	}
	return d, nil
}

// FromBytes reads a DHCPv6 message from a byte stream.
func FromBytes(data []byte) (DHCPv6, error) {
	buf := uio.NewBigEndianBuffer(data)
	messageType := MessageType(buf.Read8())
	if buf.Error() != nil {
		return nil, buf.Error()
	}

	if messageType == MessageTypeRelayForward || messageType == MessageTypeRelayReply {
		return RelayMessageFromBytes(data)
	} else {
		return MessageFromBytes(data)
	}
}

// NewMessage creates a new DHCPv6 message with default options
func NewMessage(modifiers ...Modifier) (*Message, error) {
	tid, err := GenerateTransactionID()
	if err != nil {
		return nil, err
	}
	msg := &Message{
		MessageType:   MessageTypeSolicit,
		TransactionID: tid,
	}
	// apply modifiers
	for _, mod := range modifiers {
		mod(msg)
	}
	return msg, nil
}

// DecapsulateRelay extracts the content of a relay message. It does not recurse
// if there are nested relay messages. Returns the original packet if is not not
// a relay message
func DecapsulateRelay(l DHCPv6) (DHCPv6, error) {
	if !l.IsRelay() {
		return l, nil
	}
	if rm := l.(*RelayMessage).Options.RelayMessage(); rm != nil {
		return rm, nil
	}
	return nil, fmt.Errorf("malformed Relay message: no embedded message found")
}

// DecapsulateRelayIndex extracts the content of a relay message. It takes an
// integer as index (e.g. if 0 return the outermost relay, 1 returns the
// second, etc, and -1 returns the last). Returns the original packet if
// it is not not a relay message.
func DecapsulateRelayIndex(l DHCPv6, index int) (DHCPv6, error) {
	if !l.IsRelay() {
		return l, nil
	}
	if index < -1 {
		return nil, fmt.Errorf("Invalid index: %d", index)
	} else if index == -1 {
		for {
			d, err := DecapsulateRelay(l)
			if err != nil {
				return nil, err
			}
			if !d.IsRelay() {
				return l, nil
			}
			l = d
		}
	}
	for i := 0; i <= index; i++ {
		d, err := DecapsulateRelay(l)
		if err != nil {
			return nil, err
		}
		l = d
	}
	return l, nil
}

// EncapsulateRelay creates a RelayMessage message containing the passed DHCPv6
// message as payload. The passed message type must be  either RELAY_FORW or
// RELAY_REPL
func EncapsulateRelay(d DHCPv6, mType MessageType, linkAddr, peerAddr net.IP) (*RelayMessage, error) {
	if mType != MessageTypeRelayForward && mType != MessageTypeRelayReply {
		return nil, fmt.Errorf("Message type must be either RELAY_FORW or RELAY_REPL")
	}
	outer := RelayMessage{
		MessageType: mType,
		LinkAddr:    linkAddr,
		PeerAddr:    peerAddr,
	}
	if d.IsRelay() {
		relay := d.(*RelayMessage)
		outer.HopCount = relay.HopCount + 1
	} else {
		outer.HopCount = 0
	}
	outer.AddOption(OptRelayMessage(d))
	return &outer, nil
}

// GetTransactionID returns a transactionID of a message or its inner message
// in case of relay
func GetTransactionID(packet DHCPv6) (TransactionID, error) {
	m, err := packet.GetInnerMessage()
	if err != nil {
		return TransactionID{0, 0, 0}, err
	}
	return m.TransactionID, nil
}
