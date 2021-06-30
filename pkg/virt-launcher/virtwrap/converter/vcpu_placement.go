package converter

import (
	"fmt"
	"strconv"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

func calculateRequestedVCPUs(cpuTopology *api.CPUTopology) uint32 {
	return cpuTopology.Cores * cpuTopology.Sockets * cpuTopology.Threads
}

func formatDomainCPUTune(domain *api.Domain, c *ConverterContext) error {
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
