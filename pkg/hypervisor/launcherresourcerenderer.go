package hypervisor

import (
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/hypervisor/kvm"
)

type LauncherResourceRenderer interface {
	GetHypervisorDevice() string
	GetMemoryOverhead(vmi *v1.VirtualMachineInstance, arch string, additionalOverheadRatio *string) resource.Quantity
	GetVirtType() string
	GetHypervisorDeviceMinorNumber() int64
}

func NewLauncherResourceRenderer(hypervisor string) LauncherResourceRenderer {
	switch hypervisor {
	// Other hypervisors can be added here
	default:
		return kvm.NewKvmLauncherResourceRenderer() // Default to KVM
	}
}
