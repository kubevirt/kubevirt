package rest

import (
	"net"

	"k8s.io/apimachinery/pkg/api/errors"
	v1 "kubevirt.io/api/core/v1"
)

type vmiFetcher func(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError)
type validator func(vmi *v1.VirtualMachineInstance) *errors.StatusError

type dialer interface {
	Dial(vmi *v1.VirtualMachineInstance) (net.Conn, *errors.StatusError)
}

type DirectDialer struct {
	fetchVMI    vmiFetcher
	validateVMI validator
	dialer      dialer
}

func NewDirectDialer(fetch vmiFetcher, validate validator, dialer dialer) *DirectDialer {
	return &DirectDialer{
		fetchVMI:    fetch,
		validateVMI: validate,
		dialer:      dialer,
	}
}

func (d *DirectDialer) Dial(namespace, name string) (net.Conn, *errors.StatusError) {
	vmi, err := d.fetchAndValidateVMI(namespace, name)
	if err != nil {
		return nil, err
	}

	return d.dialer.Dial(vmi)
}

func (d *DirectDialer) fetchAndValidateVMI(namespace, name string) (*v1.VirtualMachineInstance, *errors.StatusError) {
	vmi, err := d.fetchVMI(namespace, name)
	if err != nil {
		return nil, err
	}
	if err := d.validateVMI(vmi); err != nil {
		return nil, err
	}
	return vmi, nil
}
