package dhcpv6

import (
	"fmt"

	"github.com/insomniacslk/dhcp/rfc1035label"
)

// OptDomainSearchList returns a DomainSearchList option as defined by RFC 3646.
func OptDomainSearchList(labels *rfc1035label.Labels) Option {
	return &optDomainSearchList{DomainSearchList: labels}
}

type optDomainSearchList struct {
	DomainSearchList *rfc1035label.Labels
}

func (op *optDomainSearchList) Code() OptionCode {
	return OptionDomainSearchList
}

// ToBytes marshals this option to bytes.
func (op *optDomainSearchList) ToBytes() []byte {
	return op.DomainSearchList.ToBytes()
}

func (op *optDomainSearchList) String() string {
	return fmt.Sprintf("DomainSearchList: %s", op.DomainSearchList)
}

// ParseOptDomainSearchList builds an OptDomainSearchList structure from a sequence
// of bytes. The input data does not include option code and length bytes.
func parseOptDomainSearchList(data []byte) (*optDomainSearchList, error) {
	var opt optDomainSearchList
	var err error
	opt.DomainSearchList, err = rfc1035label.FromBytes(data)
	if err != nil {
		return nil, err
	}
	return &opt, nil
}
