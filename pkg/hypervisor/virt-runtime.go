package hypervisor

import (
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	kvm "kubevirt.io/kubevirt/pkg/hypervisor/kvm"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

type VirtRuntime interface {
	HandleHousekeeping(vmi *v1.VirtualMachineInstance, cgroupManager cgroup.Manager, domain *api.Domain) error
	AdjustResources(podIsoDetector isolation.PodIsolationDetector, vmi *v1.VirtualMachineInstance, config *v1.KubeVirtConfiguration) error
}

func GetVirtRuntime(podIsolationDetector isolation.PodIsolationDetector, hypervisorName string) VirtRuntime {
	switch hypervisorName {
	default:
		return kvm.NewKvmVirtRuntime(podIsolationDetector, log.Log.With("controller", "vm"))
	}
}
