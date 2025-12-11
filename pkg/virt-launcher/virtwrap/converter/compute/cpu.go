package compute

import (
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
)

type CpuConfigurator struct {
	architecture       string
	isHotplugSupported bool
}

func NewCpuConfigurator(architecture string, isHotplugSupported bool) CpuConfigurator {
	return CpuConfigurator{
		architecture:       architecture,
		isHotplugSupported: isHotplugSupported,
	}
}

func (c CpuConfigurator) Configure(vmi *v1.VirtualMachineInstance, domain *api.Domain) error {
	// Set VM CPU cores
	// CPU topology will be created everytime, because user can specify
	// number of cores in vmi.Spec.Domain.Resources.Requests/Limits, not only
	// in vmi.Spec.Domain.CPU
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}
	// set the maximum number of sockets here to allow hot-plug CPUs
	if vmiCPU := vmi.Spec.Domain.CPU; vmiCPU != nil && vmiCPU.MaxSockets != 0 && c.isHotplugSupported {
		domainVCPUTopologyForHotplug(vmi, domain)
	}
	return nil
}

func domainVCPUTopologyForHotplug(vmi *v1.VirtualMachineInstance, domain *api.Domain) {
	cpuTopology := vcpu.GetCPUTopology(vmi)
	cpuCount := vcpu.CalculateRequestedVCPUs(cpuTopology)
	// Always allow to hotplug to minimum of 1 socket
	minEnabledCpuCount := cpuTopology.Cores * cpuTopology.Threads
	// Total vCPU count
	enabledCpuCount := cpuCount
	cpuTopology.Sockets = vmi.Spec.Domain.CPU.MaxSockets
	cpuCount = vcpu.CalculateRequestedVCPUs(cpuTopology)
	VCPUs := &api.VCPUs{}
	for id := uint32(0); id < cpuCount; id++ {
		// Enable all requestd vCPUs
		isEnabled := id < enabledCpuCount
		// There should not be fewer vCPU than cores and threads within a single socket
		isHotpluggable := id >= minEnabledCpuCount
		vcpu := api.VCPUsVCPU{
			ID:           id,
			Enabled:      boolToYesNo(&isEnabled, true),
			Hotpluggable: boolToYesNo(&isHotpluggable, false),
		}
		VCPUs.VCPU = append(VCPUs.VCPU, vcpu)
	}

	domain.Spec.VCPUs = VCPUs
	domain.Spec.CPU.Topology = cpuTopology
	domain.Spec.VCPU = &api.VCPU{
		Placement: "static",
		CPUs:      cpuCount,
	}
}
