package red

import "encoding/binary"

// MiniDataHeader is a header to a data packet
type MiniDataHeader struct {
	// MessageType is type of message
	MessageType uint16

	// Size in bytes following this field to the end of this message
	Size uint32
}

// MarshalBinary marshals an Packet into a byte slice.
func (p *MiniDataHeader) MarshalBinary() ([]byte, error) {
	p.finish()
	b := make([]byte, 6)
	binary.LittleEndian.PutUint16(b[0:2], p.MessageType)
	binary.LittleEndian.PutUint32(b[2:6], p.Size)
	return b, nil
}

// UnmarshalBinary unmarshals the contents of a byte slice into a Packet.
func (p *MiniDataHeader) UnmarshalBinary(b []byte) error {
	if len(b) < 6 {
		return errInvalidPacket
	}
	p.MessageType = binary.LittleEndian.Uint16(b[0:2])
	p.Size = binary.LittleEndian.Uint32(b[2:6])
	return p.validate()
}

// validate is used to validate the Packet.
func (p *MiniDataHeader) validate() error {
	return nil
}

// finish is used to finish the Packet for sending.
func (p *MiniDataHeader) finish() {
}
