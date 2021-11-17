package converter

import (
	"fmt"
	"strconv"
	"strings"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func calculateRequestedVCPUs(cpuTopology *api.CPUTopology) uint32 {
	return cpuTopology.Cores * cpuTopology.Sockets * cpuTopology.Threads
}

func formatVCPUScheduler(domain *api.Domain, vmi *v1.VirtualMachineInstance) {

	var mask string
	if len(strings.TrimSpace(vmi.Spec.Domain.CPU.Realtime.Mask)) > 0 {
		mask = vmi.Spec.Domain.CPU.Realtime.Mask
	} else {
		mask = "0"
		if len(domain.Spec.CPUTune.VCPUPin) > 1 {
			mask = fmt.Sprintf("0-%d", len(domain.Spec.CPUTune.VCPUPin)-1)
		}
	}
	domain.Spec.CPUTune.VCPUScheduler = &api.VCPUScheduler{Scheduler: api.SchedulerFIFO, Priority: uint(1), VCPUs: mask}
}

func formatDomainCPUTune(domain *api.Domain, c *ConverterContext, vmi *v1.VirtualMachineInstance) error {
	if len(c.CPUSet) == 0 {
		return fmt.Errorf("failed for get pods pinned cpus")
	}
	vcpus := calculateRequestedVCPUs(domain.Spec.CPU.Topology)
	cpuTune := api.CPUTune{}
	for idx := 0; idx < int(vcpus); idx++ {
		vcpupin := api.CPUTuneVCPUPin{}
		vcpupin.VCPU = uint32(idx)
		vcpupin.CPUSet = strconv.Itoa(c.CPUSet[idx])
		cpuTune.VCPUPin = append(cpuTune.VCPUPin, vcpupin)
	}
	domain.Spec.CPUTune = &cpuTune
	return nil
}
