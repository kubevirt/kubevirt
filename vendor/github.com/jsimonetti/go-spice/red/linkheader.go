package red

import (
	"bytes"
	"encoding/binary"
)

// LinkHeader is a header to a client link message packet
type LinkHeader struct {
	// Magic must be equal to Magic
	Magic [4]uint8

	// Major must be equal to RED_VERSION_MAJOR
	Major uint32

	// Minor must be equal to RED_VERSION_MINOR
	Minor uint32

	// Size in bytes following this field to the end of this message
	Size uint32
}

// MarshalBinary marshals a Packet into a byte slice.
func (p *LinkHeader) MarshalBinary() ([]byte, error) {
	p.finish()
	b := make([]byte, 16)

	copy(b[0:4], p.Magic[0:4])
	binary.LittleEndian.PutUint32(b[4:8], p.Major)
	binary.LittleEndian.PutUint32(b[8:12], p.Minor)
	binary.LittleEndian.PutUint32(b[12:16], p.Size)

	return b, nil
}

// UnmarshalBinary unmarshals the contents of a byte slice into a Packet.
func (p *LinkHeader) UnmarshalBinary(b []byte) error {
	if len(b) < 16 {
		return errInvalidPacket
	}

	copy(p.Magic[0:4], b[0:4])

	p.Major = binary.LittleEndian.Uint32(b[4:8])
	p.Minor = binary.LittleEndian.Uint32(b[8:12])
	p.Size = binary.LittleEndian.Uint32(b[12:16])

	return p.validate()
}

// validate is used to validate the Packet.
func (p *LinkHeader) validate() error {
	if !bytes.Equal(p.Magic[:], Magic[:]) {
		return errInvalidPacket
	}
	if p.Major != VersionMajor || p.Minor != VersionMinor {
		return errInvalidVersion
	}
	return nil
}

// finish is used to finish the Packet for sending.
func (p *LinkHeader) finish() {
	p.Magic = Magic
	p.Major = VersionMajor
	p.Minor = VersionMinor
}
