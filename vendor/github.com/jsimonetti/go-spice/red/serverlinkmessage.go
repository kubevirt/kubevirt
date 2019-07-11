package red

import (
	"bytes"
	"encoding/binary"
)

// ServerLinkMessage is a spice packet send by the server in response to
// a ClientLinkMessage
type ServerLinkMessage struct {
	// Error codes (i.e., RED_ERROR_?)
	Error ErrorCode

	// PubKey is a 1024 bit RSA public key in X.509 SubjectPublicKeyInfo format
	PubKey [TicketPubkeyBytes]uint8

	// CommonCaps is the number of common client channel capabilities words
	CommonCaps uint32

	// ChannelCaps is the number of specific client channel capabilities words
	ChannelCaps uint32

	// CapsOffset is the location of the start of the capabilities vector given by the
	// bytes offset from the “ size” member (i.e., from the address of the “connection_id”
	// member).
	CapsOffset uint32

	// Capabilities hold the variable length capabilities
	CommonCapabilities  []Capability
	ChannelCapabilities []Capability
}

// MarshalBinary marshals a Packet into a byte slice.
func (p *ServerLinkMessage) MarshalBinary() ([]byte, error) {
	p.finish()

	b := make([]byte, int(p.CapsOffset)+4*len(p.CommonCapabilities)+4*len(p.ChannelCapabilities))
	binary.LittleEndian.PutUint32(b[0:4], uint32(p.Error))

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, p.PubKey); err != nil {
		return nil, err
	}
	copy(b[4:4+TicketPubkeyBytes], buf.Bytes())

	binary.LittleEndian.PutUint32(b[4+TicketPubkeyBytes:8+TicketPubkeyBytes], p.CommonCaps)
	binary.LittleEndian.PutUint32(b[8+TicketPubkeyBytes:12+TicketPubkeyBytes], p.ChannelCaps)
	binary.LittleEndian.PutUint32(b[12+TicketPubkeyBytes:16+TicketPubkeyBytes], p.CapsOffset)

	offset := 16 + TicketPubkeyBytes
	for i := 0; i < len(p.CommonCapabilities); i += 4 {
		binary.LittleEndian.PutUint32(b[i+offset:i+offset+4], uint32(p.CommonCapabilities[i]))
	}

	offset = 16 + TicketPubkeyBytes + 4*len(p.CommonCapabilities)
	for i := 0; i < len(p.ChannelCapabilities); i += 4 {
		binary.LittleEndian.PutUint32(b[i+offset:i+offset+4], uint32(p.ChannelCapabilities[i]))
	}

	return b, nil
}

// UnmarshalBinary unmarshals the contents of a byte slice into a Packet.
func (p *ServerLinkMessage) UnmarshalBinary(b []byte) error {
	if len(b) < 178 {
		return errInvalidPacket
	}

	p.Error = ErrorCode(binary.LittleEndian.Uint32(b[0:4]))

	buf := bytes.NewReader(b[4 : 4+TicketPubkeyBytes])
	if err := binary.Read(buf, binary.LittleEndian, p.PubKey[:]); err != nil {
		return err
	}

	p.CommonCaps = binary.LittleEndian.Uint32(b[4+TicketPubkeyBytes : 8+TicketPubkeyBytes])
	p.ChannelCaps = binary.LittleEndian.Uint32(b[8+TicketPubkeyBytes : 12+TicketPubkeyBytes])
	p.CapsOffset = binary.LittleEndian.Uint32(b[12+TicketPubkeyBytes : 16+TicketPubkeyBytes])

	if len(b) < 178+int(p.CommonCaps)*4+int(p.ChannelCaps)*4 {
		return errInvalidPacket
	}

	for i := 178; i < 178+int(p.CommonCaps)*4; i += 4 {
		if len(b) < i+4 {
			return errInvalidPacket
		}
		p.CommonCapabilities = append(p.CommonCapabilities, Capability(binary.LittleEndian.Uint32(b[i:i+4])))
	}

	for i := 178 + len(p.CommonCapabilities)*4; i < 178+int(p.CommonCaps)*4+int(p.ChannelCaps)*4; i += 4 {
		if len(b) < i+4 {
			return errInvalidPacket
		}
		p.ChannelCapabilities = append(p.ChannelCapabilities, Capability(binary.LittleEndian.Uint32(b[i:i+4])))
	}

	return p.validate()
}

// validate is used to validate the Packet.
func (p *ServerLinkMessage) validate() error {
	return nil
}

// finish is used to finish the Packet for sending.
func (p *ServerLinkMessage) finish() {
	p.CapsOffset = 16 + TicketPubkeyBytes
	p.CommonCaps = uint32(len(p.CommonCapabilities))
	p.ChannelCaps = uint32(len(p.ChannelCapabilities))
}
