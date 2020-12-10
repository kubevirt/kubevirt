package dhcpv6

import (
	"fmt"
	"time"

	"github.com/u-root/u-root/pkg/uio"
)

// OptElapsedTime returns an Elapsed Time option as defined by RFC 3315 Section
// 22.9.
func OptElapsedTime(dur time.Duration) Option {
	return &optElapsedTime{ElapsedTime: dur}
}

type optElapsedTime struct {
	ElapsedTime time.Duration
}

func (*optElapsedTime) Code() OptionCode {
	return OptionElapsedTime
}

// ToBytes marshals this option to bytes.
func (op *optElapsedTime) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	buf.Write16(uint16(op.ElapsedTime.Round(10*time.Millisecond) / (10 * time.Millisecond)))
	return buf.Data()
}

func (op *optElapsedTime) String() string {
	return fmt.Sprintf("ElapsedTime: %s", op.ElapsedTime)
}

// build an optElapsedTime structure from a sequence of bytes.
// The input data does not include option code and length bytes.
func parseOptElapsedTime(data []byte) (*optElapsedTime, error) {
	var opt optElapsedTime
	buf := uio.NewBigEndianBuffer(data)
	opt.ElapsedTime = time.Duration(buf.Read16()) * 10 * time.Millisecond
	return &opt, buf.FinError()
}
