package dhcpv6

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/iana"
)

// DuidType is the DUID type as defined in rfc3315.
type DuidType uint16

// DUID types
const (
	DUID_LLT  DuidType = 1
	DUID_EN   DuidType = 2
	DUID_LL   DuidType = 3
	DUID_UUID DuidType = 4
)

// DuidTypeToString maps a DuidType to a name.
var DuidTypeToString = map[DuidType]string{
	DUID_LL:   "DUID-LL",
	DUID_LLT:  "DUID-LLT",
	DUID_EN:   "DUID-EN",
	DUID_UUID: "DUID-UUID",
}

func (d DuidType) String() string {
	if dtype, ok := DuidTypeToString[d]; ok {
		return dtype
	}
	return "Unknown"
}

// Duid is a DHCP Unique Identifier.
type Duid struct {
	Type                 DuidType
	HwType               iana.HWType // for DUID-LLT and DUID-LL. Ignored otherwise. RFC 826
	Time                 uint32      // for DUID-LLT. Ignored otherwise
	LinkLayerAddr        net.HardwareAddr
	EnterpriseNumber     uint32 // for DUID-EN. Ignored otherwise
	EnterpriseIdentifier []byte // for DUID-EN. Ignored otherwise
	Uuid                 []byte // for DUID-UUID. Ignored otherwise
	Opaque               []byte // for unknown DUIDs
}

// Length returns the DUID length in bytes.
func (d *Duid) Length() int {
	if d.Type == DUID_LLT {
		return 8 + len(d.LinkLayerAddr)
	} else if d.Type == DUID_LL {
		return 4 + len(d.LinkLayerAddr)
	} else if d.Type == DUID_EN {
		return 6 + len(d.EnterpriseIdentifier)
	} else if d.Type == DUID_UUID {
		return 18
	} else {
		return 2 + len(d.Opaque)
	}
}

// Equal compares two Duid objects.
func (d Duid) Equal(o Duid) bool {
	if d.Type != o.Type ||
		d.HwType != o.HwType ||
		d.Time != o.Time ||
		!bytes.Equal(d.LinkLayerAddr, o.LinkLayerAddr) ||
		d.EnterpriseNumber != o.EnterpriseNumber ||
		!bytes.Equal(d.EnterpriseIdentifier, o.EnterpriseIdentifier) ||
		!bytes.Equal(d.Uuid, o.Uuid) ||
		!bytes.Equal(d.Opaque, o.Opaque) {
		return false
	}
	return true
}

// ToBytes serializes a Duid object.
func (d *Duid) ToBytes() []byte {
	if d.Type == DUID_LLT {
		buf := make([]byte, 8)
		binary.BigEndian.PutUint16(buf[0:2], uint16(d.Type))
		binary.BigEndian.PutUint16(buf[2:4], uint16(d.HwType))
		binary.BigEndian.PutUint32(buf[4:8], d.Time)
		return append(buf, d.LinkLayerAddr...)
	} else if d.Type == DUID_LL {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint16(buf[0:2], uint16(d.Type))
		binary.BigEndian.PutUint16(buf[2:4], uint16(d.HwType))
		return append(buf, d.LinkLayerAddr...)
	} else if d.Type == DUID_EN {
		buf := make([]byte, 6)
		binary.BigEndian.PutUint16(buf[0:2], uint16(d.Type))
		binary.BigEndian.PutUint32(buf[2:6], d.EnterpriseNumber)
		return append(buf, d.EnterpriseIdentifier...)
	} else if d.Type == DUID_UUID {
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf[0:2], uint16(d.Type))
		return append(buf, d.Uuid...)
	} else {
		buf := make([]byte, 2)
		binary.BigEndian.PutUint16(buf[0:2], uint16(d.Type))
		return append(buf, d.Opaque...)
	}
}

func (d *Duid) String() string {
	var hwaddr string
	if d.HwType == iana.HWTypeEthernet {
		for _, b := range d.LinkLayerAddr {
			hwaddr += fmt.Sprintf("%02x:", b)
		}
		if len(hwaddr) > 0 && hwaddr[len(hwaddr)-1] == ':' {
			hwaddr = hwaddr[:len(hwaddr)-1]
		}
	}
	return fmt.Sprintf("DUID{type=%v hwtype=%v hwaddr=%v}", d.Type.String(), d.HwType.String(), hwaddr)
}

// DuidFromBytes parses a Duid from a byte slice.
func DuidFromBytes(data []byte) (*Duid, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("Invalid DUID: shorter than 2 bytes")
	}
	d := Duid{}
	d.Type = DuidType(binary.BigEndian.Uint16(data[0:2]))
	if d.Type == DUID_LLT {
		if len(data) < 8 {
			return nil, fmt.Errorf("Invalid DUID-LLT: shorter than 8 bytes")
		}
		d.HwType = iana.HWType(binary.BigEndian.Uint16(data[2:4]))
		d.Time = binary.BigEndian.Uint32(data[4:8])
		d.LinkLayerAddr = data[8:]
	} else if d.Type == DUID_LL {
		if len(data) < 4 {
			return nil, fmt.Errorf("Invalid DUID-LL: shorter than 4 bytes")
		}
		d.HwType = iana.HWType(binary.BigEndian.Uint16(data[2:4]))
		d.LinkLayerAddr = data[4:]
	} else if d.Type == DUID_EN {
		if len(data) < 6 {
			return nil, fmt.Errorf("Invalid DUID-EN: shorter than 6 bytes")
		}
		d.EnterpriseNumber = binary.BigEndian.Uint32(data[2:6])
		d.EnterpriseIdentifier = data[6:]
	} else if d.Type == DUID_UUID {
		if len(data) != 18 {
			return nil, fmt.Errorf("Invalid DUID-UUID length. Expected 18, got %v", len(data))
		}
		d.Uuid = data[2:18]
	} else {
		d.Opaque = data[2:]
	}
	return &d, nil
}
