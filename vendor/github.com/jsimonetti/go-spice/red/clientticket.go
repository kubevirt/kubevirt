package red

import (
	"bytes"
	"encoding/binary"
)

// ClientTicket is a spice packet send by the client
// that contains a ticket
type ClientTicket struct {
	// Ticket is the RSA encrypted ticket
	Ticket [ClientTicketBytes]byte
}

// MarshalBinary marshals a Packet into a byte slice.
func (p *ClientTicket) MarshalBinary() ([]byte, error) {
	p.finish()

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, p.Ticket); err != nil {
		return nil, err
	}

	return buf.Bytes()[0:ClientTicketBytes], nil
}

// UnmarshalBinary unmarshals the contents of a byte slice into a Packet.
func (p *ClientTicket) UnmarshalBinary(b []byte) error {
	if len(b) != ClientTicketBytes {
		return errInvalidPacket
	}

	buf := bytes.NewReader(b[0:ClientTicketBytes])
	if err := binary.Read(buf, binary.LittleEndian, p.Ticket[:]); err != nil {
		return err
	}

	return p.validate()
}

// validate is used to validate the Packet.
func (p *ClientTicket) validate() error {
	return nil
}

// finish is used to finish the Packet for sending.
func (p *ClientTicket) finish() {
}
