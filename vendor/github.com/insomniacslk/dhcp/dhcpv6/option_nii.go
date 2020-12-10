package dhcpv6

import (
	"fmt"

	"github.com/u-root/u-root/pkg/uio"
)

// NetworkInterfaceType is the NIC type as defined by RFC 4578 Section 2.2
type NetworkInterfaceType uint8

// see rfc4578
const (
	NII_LANDESK_NOPXE   NetworkInterfaceType = 0
	NII_PXE_GEN_I       NetworkInterfaceType = 1
	NII_PXE_GEN_II      NetworkInterfaceType = 2
	NII_UNDI_NOEFI      NetworkInterfaceType = 3
	NII_UNDI_EFI_GEN_I  NetworkInterfaceType = 4
	NII_UNDI_EFI_GEN_II NetworkInterfaceType = 5
)

func (nit NetworkInterfaceType) String() string {
	if s, ok := niiToStringMap[nit]; ok {
		return s
	}
	return fmt.Sprintf("NetworkInterfaceType(%d, unknown)", nit)
}

var niiToStringMap = map[NetworkInterfaceType]string{
	NII_LANDESK_NOPXE:   "LANDesk service agent boot ROMs. No PXE",
	NII_PXE_GEN_I:       "First gen. PXE boot ROMs",
	NII_PXE_GEN_II:      "Second gen. PXE boot ROMs",
	NII_UNDI_NOEFI:      "UNDI 32/64 bit. UEFI drivers, no UEFI runtime",
	NII_UNDI_EFI_GEN_I:  "UNDI 32/64 bit. UEFI runtime 1st gen",
	NII_UNDI_EFI_GEN_II: "UNDI 32/64 bit. UEFI runtime 2nd gen",
}

// OptNetworkInterfaceID implements the NIC ID option for network booting as
// defined by RFC 4578 Section 2.2 and RFC 5970 Section 3.4.
type OptNetworkInterfaceID struct {
	Typ NetworkInterfaceType

	// Revision number
	Major, Minor uint8
}

// Code implements Option.Code.
func (*OptNetworkInterfaceID) Code() OptionCode {
	return OptionNII
}

// ToBytes implements Option.ToBytes.
func (op *OptNetworkInterfaceID) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write8(uint8(op.Typ))
	buf.Write8(op.Major)
	buf.Write8(op.Minor)
	return buf.Data()
}

func (op *OptNetworkInterfaceID) String() string {
	return fmt.Sprintf("NetworkInterfaceID: %s (Revision %d.%d)", op.Typ, op.Major, op.Minor)
}

// FromBytes builds an OptNetworkInterfaceID structure from a sequence of
// bytes. The input data does not include option code and length bytes.
func (op *OptNetworkInterfaceID) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	op.Typ = NetworkInterfaceType(buf.Read8())
	op.Major = buf.Read8()
	op.Minor = buf.Read8()
	return buf.FinError()
}
