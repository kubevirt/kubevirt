package tests

import (
	"fmt"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libnet"
)

// LoginToCirros performs a console login to a Cirros base VM
func LoginToCirros(vmi *v1.VirtualMachineInstance) error {
	err := console.LoginToCirros(vmi)
	if err != nil {
		return fmt.Errorf("failed login into cirros console: %w", err)
	}

	err = libnet.ConfigureIPv6OnVMI(vmi)
	if err != nil {
		return fmt.Errorf("failed configuring ipv6 on cirros console: %w", err)
	}
	return nil
}

// LoginToAlpine performs a console login to an Alpine base VM
func LoginToAlpine(vmi *v1.VirtualMachineInstance) error {
	err := console.LoginToAlpine(vmi)
	if err != nil {
		return fmt.Errorf("failed login into alpine console: %w", err)
	}
	return nil
}

// LoginToFedora performs a console login to a Fedora base VM
func LoginToFedora(vmi *v1.VirtualMachineInstance) error {
	err := console.LoginToFedora(vmi)
	if err != nil {
		return fmt.Errorf("failed login into fedora console: %w", err)
	}

	err = libnet.ConfigureIPv6OnVMI(vmi)
	if err != nil {
		return fmt.Errorf("failed configuring ipv6 on fedora console: %w", err)
	}
	return nil
}
