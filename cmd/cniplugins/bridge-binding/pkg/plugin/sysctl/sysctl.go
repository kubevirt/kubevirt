package sysctl

import (
	"strconv"

	"kubevirt.io/kubevirt/pkg/util/sysctl"
)

type sysControl struct{}

var sysCtl = sysctl.New()

func New() sysControl {
	return sysControl{}
}

func (_ sysControl) IPv4SetUnprivilegedPortStart(port int) error {
	return sysCtl.SetSysctl(sysctl.UnprivilegedPortStart, strconv.Itoa(port))
}
