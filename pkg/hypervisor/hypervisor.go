package hypervisor

import v1 "kubevirt.io/api/core/v1"

// Hypervisor interface defines functions needed to tune the virt-launcher pod spec and the libvirt domain XML for a specific hypervisor
type Hypervisor interface {
	// GetK8sResourceName returns the name of the K8s resource representing the hypervisor
	GetK8sResourceName() string
}

func NewHypervisor(hypervisor string) Hypervisor {
	switch hypervisor {
	case v1.MshvL1vhHypervisorName:
		return &MshvL1vhHypervisor{}
	default:
		return &KVMHypervisor{}
	}
}
