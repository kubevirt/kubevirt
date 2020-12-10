package dhcpv6

// This module defines the optRelayMsg structure.
// https://www.ietf.org/rfc/rfc3315.txt

import (
	"fmt"
)

// OptRelayMessage embeds a message in a relay option.
func OptRelayMessage(msg DHCPv6) Option {
	return &optRelayMsg{Msg: msg}
}

type optRelayMsg struct {
	Msg DHCPv6
}

func (op *optRelayMsg) Code() OptionCode {
	return OptionRelayMsg
}

func (op *optRelayMsg) ToBytes() []byte {
	return op.Msg.ToBytes()
}

func (op *optRelayMsg) String() string {
	return fmt.Sprintf("RelayMsg: %v", op.Msg)
}

// build an optRelayMsg structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func parseOptRelayMsg(data []byte) (*optRelayMsg, error) {
	var err error
	var opt optRelayMsg
	opt.Msg, err = FromBytes(data)
	if err != nil {
		return nil, err
	}
	return &opt, nil
}
