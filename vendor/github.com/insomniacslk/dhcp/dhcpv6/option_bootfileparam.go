package dhcpv6

import (
	"fmt"

	"github.com/u-root/uio/uio"
)

// OptBootFileParam returns a BootfileParam option as defined in RFC 5970
// Section 3.2.
func OptBootFileParam(args ...string) Option {
	return &optBootFileParam{args}
}

type optBootFileParam struct {
	params []string
}

// Code returns the option code
func (optBootFileParam) Code() OptionCode {
	return OptionBootfileParam
}

// ToBytes serializes the option and returns it as a sequence of bytes
func (op optBootFileParam) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	for _, param := range op.params {
		if len(param) >= 1<<16 {
			// TODO: say something here instead of silently ignoring a parameter
			continue
		}
		buf.Write16(uint16(len(param)))
		buf.WriteBytes([]byte(param))
		/*if err := buf.Error(); err != nil {
			// TODO: description of `WriteBytes` says it could return
			// an error via `buf.Error()`. But a quick look into implementation of
			// `WriteBytes` at the moment of this comment showed it does not set any
			// errors to `Error()` output. It's required to make a decision:
			// to fix `WriteBytes` or it's description or
			// to find a way to handle an error here.
		}*/
	}
	return buf.Data()
}

func (op optBootFileParam) String() string {
	return fmt.Sprintf("%s: %v", op.Code(), op.params)
}

// FromBytes builds an OptBootFileParam structure from a sequence
// of bytes. The input data does not include option code and length bytes.
func (op *optBootFileParam) FromBytes(data []byte) error {
	buf := uio.NewBigEndianBuffer(data)
	for buf.Has(2) {
		length := buf.Read16()
		op.params = append(op.params, string(buf.CopyN(int(length))))
	}
	return buf.FinError()
}
