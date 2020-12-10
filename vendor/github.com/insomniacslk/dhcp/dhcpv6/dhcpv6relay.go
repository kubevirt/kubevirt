package dhcpv6

import (
	"errors"
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/iana"
	"github.com/u-root/u-root/pkg/uio"
)

const RelayHeaderSize = 34

// RelayOptions are the options valid for RelayForw and RelayRepl messages.
//
// RFC 3315 Appendix B defines them to be InterfaceID and RelayMsg options; RFC
// 4649 also adds the RemoteID option.
type RelayOptions struct {
	Options
}

// RelayMessage returns the message embedded.
func (ro RelayOptions) RelayMessage() DHCPv6 {
	opt := ro.Options.GetOne(OptionRelayMsg)
	if opt == nil {
		return nil
	}
	if relayOpt, ok := opt.(*optRelayMsg); ok {
		return relayOpt.Msg
	}
	return nil
}

// InterfaceID returns the interface ID of this relay message.
func (ro RelayOptions) InterfaceID() []byte {
	opt := ro.Options.GetOne(OptionInterfaceID)
	if opt == nil {
		return nil
	}
	if iid, ok := opt.(*optInterfaceID); ok {
		return iid.ID
	}
	return nil
}

// RemoteID returns the remote ID in this relay message.
func (ro RelayOptions) RemoteID() *OptRemoteID {
	opt := ro.Options.GetOne(OptionRemoteID)
	if opt == nil {
		return nil
	}
	if rid, ok := opt.(*OptRemoteID); ok {
		return rid
	}
	return nil
}

// ClientLinkLayerAddress returns the Hardware Type and
// Link Layer Address of the requesting client in this relay message.
func (ro RelayOptions) ClientLinkLayerAddress() (iana.HWType, net.HardwareAddr) {
	opt := ro.Options.GetOne(OptionClientLinkLayerAddr)
	if opt == nil {
		return 0, nil
	}
	if lla, ok := opt.(*optClientLinkLayerAddress); ok {
		return lla.LinkLayerType, lla.LinkLayerAddress
	}
	return 0, nil
}

// RelayMessage is a DHCPv6 relay agent message as defined by RFC 3315 Section
// 7.
type RelayMessage struct {
	MessageType MessageType
	HopCount    uint8
	LinkAddr    net.IP
	PeerAddr    net.IP
	Options     RelayOptions
}

func write16(b *uio.Lexer, ip net.IP) {
	if ip == nil || ip.To16() == nil {
		var zeros [net.IPv6len]byte
		b.WriteBytes(zeros[:])
	} else {
		b.WriteBytes(ip.To16())
	}
}

// Type is this relay message's types.
func (r *RelayMessage) Type() MessageType {
	return r.MessageType
}

// String prints a short human-readable relay message.
func (r *RelayMessage) String() string {
	ret := fmt.Sprintf(
		"RelayMessage(messageType=%s hopcount=%d, linkaddr=%s, peeraddr=%s, %d options)",
		r.Type(), r.HopCount, r.LinkAddr, r.PeerAddr, len(r.Options.Options),
	)
	return ret
}

// Summary prints all options associated with this relay message.
func (r *RelayMessage) Summary() string {
	ret := fmt.Sprintf(
		"RelayMessage\n"+
			"  messageType=%v\n"+
			"  hopcount=%v\n"+
			"  linkaddr=%v\n"+
			"  peeraddr=%v\n"+
			"  options=%v\n",
		r.Type(),
		r.HopCount,
		r.LinkAddr,
		r.PeerAddr,
		r.Options,
	)
	return ret
}

// ToBytes returns the serialized version of this relay message as defined by
// RFC 3315, Section 7.
func (r *RelayMessage) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(make([]byte, 0, RelayHeaderSize))
	buf.Write8(byte(r.MessageType))
	buf.Write8(r.HopCount)
	write16(buf, r.LinkAddr)
	write16(buf, r.PeerAddr)
	buf.WriteBytes(r.Options.ToBytes())
	return buf.Data()
}

// GetOption returns the options associated with the code.
func (r *RelayMessage) GetOption(code OptionCode) []Option {
	return r.Options.Get(code)
}

// GetOneOption returns the first associated option with the code from this
// message.
func (r *RelayMessage) GetOneOption(code OptionCode) Option {
	return r.Options.GetOne(code)
}

// AddOption adds an option to this message.
func (r *RelayMessage) AddOption(option Option) {
	r.Options.Add(option)
}

// UpdateOption replaces the first option of the same type as the specified one.
func (r *RelayMessage) UpdateOption(option Option) {
	r.Options.Update(option)
}

// IsRelay returns whether this is a relay message or not.
func (r *RelayMessage) IsRelay() bool {
	return true
}

// GetInnerMessage recurses into a relay message and extract and return the
// inner Message. Return nil if none found (e.g. not a relay message).
func (r *RelayMessage) GetInnerMessage() (*Message, error) {
	var (
		p   DHCPv6
		err error
	)
	p = r
	for {
		p, err = DecapsulateRelay(p)
		if err != nil {
			return nil, err
		}
		if m, ok := p.(*Message); ok {
			return m, nil
		}
	}
}

// NewRelayReplFromRelayForw creates a MessageTypeRelayReply based on a
// MessageTypeRelayForward and replaces the inner message with the passed
// DHCPv6 message. It copies the OptionInterfaceID and OptionRemoteID if the
// options are present in the Relay packet.
func NewRelayReplFromRelayForw(relay *RelayMessage, msg *Message) (DHCPv6, error) {
	var (
		err                error
		linkAddr, peerAddr []net.IP
		optiid             []Option
		optrid             []Option
	)
	if relay == nil {
		return nil, errors.New("Relay message cannot be nil")
	}
	if relay.Type() != MessageTypeRelayForward {
		return nil, errors.New("The passed packet is not of type MessageTypeRelayForward")
	}
	if msg == nil {
		return nil, errors.New("The passed message cannot be nil")
	}
	for {
		linkAddr = append(linkAddr, relay.LinkAddr)
		peerAddr = append(peerAddr, relay.PeerAddr)
		optiid = append(optiid, relay.GetOneOption(OptionInterfaceID))
		optrid = append(optrid, relay.GetOneOption(OptionRemoteID))
		decap, err := DecapsulateRelay(relay)
		if err != nil {
			return nil, err
		}
		if decap.IsRelay() {
			relay = decap.(*RelayMessage)
		} else {
			break
		}
	}
	m := DHCPv6(msg)
	for i := len(linkAddr) - 1; i >= 0; i-- {
		m, err = EncapsulateRelay(m, MessageTypeRelayReply, linkAddr[i], peerAddr[i])
		if err != nil {
			return nil, err
		}
		if opt := optiid[i]; opt != nil {
			m.AddOption(opt)
		}
		if opt := optrid[i]; opt != nil {
			m.AddOption(opt)
		}
	}
	return m, nil
}
