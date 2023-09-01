package libvm

import (
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/tests/libvmi"
)

func NewCirros(opts ...Option) *virtv1.VirtualMachine {
	cirrosOpts := []Option{
		WithVMITemplateSpec(libvmi.CirrosOpts...),
	}
	cirrosOpts = append(cirrosOpts, opts...)
	return New(cirrosOpts...)
}
