package dhcpv6

import (
	"fmt"
	"time"

	"github.com/u-root/u-root/pkg/uio"
)

// OptInformationRefreshTime implements OptionInformationRefreshTime option.
// https://tools.ietf.org/html/rfc8415#section-21.23
func OptInformationRefreshTime(irt time.Duration) *optInformationRefreshTime {
	return &optInformationRefreshTime{irt}
}

// optInformationRefreshTime represents an OptionInformationRefreshTime.
type optInformationRefreshTime struct {
	InformationRefreshtime time.Duration
}

// Code returns the option's code
func (op *optInformationRefreshTime) Code() OptionCode {
	return OptionInformationRefreshTime
}

// ToBytes serializes the option and returns it as a sequence of bytes
func (op *optInformationRefreshTime) ToBytes() []byte {
	buf := uio.NewBigEndianBuffer(nil)
	irt := Duration{op.InformationRefreshtime}
	irt.Marshal(buf)
	return buf.Data()
}

func (op *optInformationRefreshTime) String() string {
	return fmt.Sprintf("InformationRefreshTime: %v", op.InformationRefreshtime)
}

// parseOptInformationRefreshTime builds an optInformationRefreshTime structure from a sequence
// of bytes. The input data does not include option code and length bytes.
func parseOptInformationRefreshTime(data []byte) (*optInformationRefreshTime, error) {
	var opt optInformationRefreshTime
	buf := uio.NewBigEndianBuffer(data)

	var irt Duration
	irt.Unmarshal(buf)
	opt.InformationRefreshtime = irt.Duration
	return &opt, buf.FinError()
}
