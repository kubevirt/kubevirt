package sysctl

import (
	"fmt"
	"strconv"

	"kubevirt.io/kubevirt/pkg/util/sysctl"
)

type sysControl struct{}

var sysCtl = sysctl.New()

func New() sysControl {
	return sysControl{}
}

func (sysControl) IPv4SetPingGroupRange(from, to int) error {
	return sysCtl.SetSysctl(sysctl.PingGroupRange, fmt.Sprintf("%d %d", from, to))
}

func (sysControl) IPv4SetUnprivilegedPortStart(port int) error {
	return sysCtl.SetSysctl(sysctl.UnprivilegedPortStart, strconv.Itoa(port))
}
