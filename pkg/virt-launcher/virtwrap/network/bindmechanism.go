package network

import (
	v1 "kubevirt.io/client-go/api/v1"
)

type BindMechanism interface {
	discoverPodNetworkInterface() error
	preparePodNetworkInterfaces(queueNumber uint32, launcherPID int) error

	loadCachedInterface(pid, name string) (bool, error)
	setCachedInterface(pid, name string) error

	// virt-handler that executes phase1 of network configuration needs to
	// pass details about discovered networking port into phase2 that is
	// executed by virt-launcher. Virt-launcher cannot discover some of
	// these details itself because at this point phase1 is complete and
	// ports are rewired, meaning, routes and IP addresses configured by
	// CNI plugin may be gone. For this matter, we use a cached VIF file to
	// pass discovered information between phases.
	loadCachedVIF(pid, name string) (bool, error)
	setCachedVIF(pid, name string) error

	// The following entry points require domain initialized for the
	// binding and can be used in phase2 only.
	decorateConfig() error
	startDHCP(vmi *v1.VirtualMachineInstance) error
}
