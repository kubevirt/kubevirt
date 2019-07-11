package red

import "encoding/binary"

// ClientLinkMessage is a spice packet send by the client
// to start a connection.
type ClientLinkMessage struct {
	// SessionID In   case   of   a   new   session   (i.e.,   channel   type   is
	// ChannelMain) this field is set to zero, and in response the server will
	// allocate session id and will send it via the RedLinkReply message. In case of all other
	// channel types, this field will be equal to the allocated session id.
	SessionID uint32

	// ChannelType is one of RED_CHANNEL_?
	ChannelType ChannelType

	// ChannelID to connect to. This enables having multiple channels of the same type
	ChannelID uint8

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
func (p *ClientLinkMessage) MarshalBinary() ([]byte, error) {
	p.finish()

	b := make([]byte, int(p.CapsOffset)+4*len(p.CommonCapabilities)+4*len(p.ChannelCapabilities))
	binary.LittleEndian.PutUint32(b[0:4], uint32(p.SessionID))
	b[4] = uint8(p.ChannelType)
	b[5] = p.ChannelID
	binary.LittleEndian.PutUint32(b[6:10], p.CommonCaps)
	binary.LittleEndian.PutUint32(b[10:14], p.ChannelCaps)
	binary.LittleEndian.PutUint32(b[14:18], p.CapsOffset)

	offset := 18
	for i := 0; i < len(p.CommonCapabilities); i += 4 {
		binary.LittleEndian.PutUint32(b[i+offset:i+offset+4], uint32(p.CommonCapabilities[i]))
	}

	offset = 18 + 4*len(p.CommonCapabilities)
	for i := 0; i < len(p.ChannelCapabilities); i += 4 {
		binary.LittleEndian.PutUint32(b[i+offset:i+offset+4], uint32(p.ChannelCapabilities[i]))
	}

	return b, nil
}

// UnmarshalBinary unmarshals the contents of a byte slice into a Packet.
func (p *ClientLinkMessage) UnmarshalBinary(b []byte) error {
	if len(b) < 18 {
		return errInvalidPacket
	}

	p.SessionID = binary.LittleEndian.Uint32(b[0:4])
	p.ChannelType = ChannelType(b[4])
	p.ChannelID = b[5]
	p.CommonCaps = binary.LittleEndian.Uint32(b[6:10])
	p.ChannelCaps = binary.LittleEndian.Uint32(b[10:14])
	p.CapsOffset = binary.LittleEndian.Uint32(b[14:18])

	if len(b) < 18+int(p.CommonCaps)*4+int(p.ChannelCaps)*4 {
		return errInvalidPacket
	}

	for i := 18; i < 18+int(p.CommonCaps)*4; i += 4 {
		if len(b) < i+4 {
			return errInvalidPacket
		}
		p.CommonCapabilities = append(p.CommonCapabilities, Capability(binary.LittleEndian.Uint32(b[i:i+4])))
	}

	for i := 18 + len(p.CommonCapabilities)*4; i < 18+int(p.CommonCaps)*4+int(p.ChannelCaps)*4; i += 4 {
		if len(b) < i+4 {
			return errInvalidPacket
		}
		p.ChannelCapabilities = append(p.ChannelCapabilities, Capability(binary.LittleEndian.Uint32(b[i:i+4])))
	}

	return p.validate()
}

// validate is used to validate the Packet.
func (p *ClientLinkMessage) validate() error {
	return nil
}

// finish is used to finish the Packet for sending.
func (p *ClientLinkMessage) finish() {
	p.CapsOffset = 18
	p.CommonCaps = uint32(len(p.CommonCapabilities))
	p.ChannelCaps = uint32(len(p.ChannelCapabilities))
}
