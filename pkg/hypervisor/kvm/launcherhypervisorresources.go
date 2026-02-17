/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package kvm

import (
	"strconv"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/tpm"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/util/hardware"
)

const (
	VirtLauncherMonitorOverhead = "25Mi"  // The `ps` RSS for virt-launcher-monitor
	VirtLauncherOverhead        = "100Mi" // The `ps` RSS for the virt-launcher process
	VirtlogdOverhead            = "25Mi"  // The `ps` RSS for virtlogd
	VirtqemudOverhead           = "40Mi"  // The `ps` RSS for virtqemud
	QemuOverhead                = "30Mi"  // The `ps` RSS for qemu, minus the RAM of its (stressed) guest, minus the virtual page table

	KvmHypervisorDevice = "kvm"

	pageSize = 512 // Hardware-defined page size in bytes for pagetable calculations
)

type KvmLauncherHypervisorResources struct{}

func NewKvmLauncherHypervisorResources() *KvmLauncherHypervisorResources {
	return &KvmLauncherHypervisorResources{}
}

func (k *KvmLauncherHypervisorResources) GetHypervisorDevice() string {
	return KvmHypervisorDevice
}

// GetMemoryOverhead computes the estimation of total
// memory needed for the domain to operate properly.
// This includes the memory needed for the guest and memory
// for Qemu and OS overhead.
// The return value is overhead memory quantity
//
// Note: The overhead memory is a calculated estimation, the values are not to be assumed accurate.
//
//nolint:gocyclo // complexity is inherent to memory overhead calculation
func (k *KvmLauncherHypervisorResources) GetMemoryOverhead(
	vmi *v1.VirtualMachineInstance, cpuArch string, additionalOverheadRatio *string,
) resource.Quantity {
	domain := vmi.Spec.Domain
	vmiMemoryReq := domain.Resources.Requests.Memory()

	overhead := *resource.NewScaledQuantity(0, resource.Kilo)

	overhead.Add(calculatePagetableMemory(vmiMemoryReq))

	// Add fixed overhead for KubeVirt components, as seen in a random run, rounded up to the nearest MiB
	// Note: shared libraries are included in the size, so every library is counted (wrongly) as many times as there are
	//   processes using it. However, the extra memory is only in the order of 10MiB and makes for a nice safety margin.
	overhead.Add(resource.MustParse(VirtLauncherMonitorOverhead))
	overhead.Add(resource.MustParse(VirtLauncherOverhead))
	overhead.Add(resource.MustParse(VirtlogdOverhead))
	overhead.Add(resource.MustParse(VirtqemudOverhead))
	overhead.Add(resource.MustParse(QemuOverhead))

	// Add CPU overhead (8 MiB per vCPU)
	overhead.Add(calculateVCPUOverhead(vmi))

	// static overhead for IOThread
	overhead.Add(resource.MustParse("8Mi"))

	// Add video RAM overhead
	if domain.Devices.AutoattachGraphicsDevice == nil || *domain.Devices.AutoattachGraphicsDevice {
		overhead.Add(resource.MustParse("32Mi"))
	}

	// When use uefi boot on aarch64 with edk2 package, qemu will create 2 pflash(64Mi each, 128Mi in total)
	// it should be considered for memory overhead
	// Additional information can be found here: https://github.com/qemu/qemu/blob/master/hw/arm/virt.c#L120
	if cpuArch == "arm64" {
		overhead.Add(resource.MustParse("128Mi"))
	}

	// Additional overhead of 1G for VFIO devices. VFIO requires all guest RAM to be locked
	// in addition to MMIO memory space to allow DMA. 1G is often the size of reserved MMIO space on x86 systems.
	// Additial information can be found here: https://www.redhat.com/archives/libvir-list/2015-November/msg00329.html
	if util.IsVFIOVMI(vmi) {
		overhead.Add(resource.MustParse("1Gi"))
	}

	// DownardMetrics volumes are using emptyDirs backed by memory.
	// the max. disk size is only 256Ki.
	if downwardmetrics.HasDownwardMetricDisk(vmi) {
		overhead.Add(resource.MustParse("1Mi"))
	}

	addProbeOverheads(vmi, &overhead)

	// Consider memory overhead for SEV guests.
	// Additional information can be found here: https://libvirt.org/kbase/launch_security_sev.html#memory
	if util.IsSEVVMI(vmi) || util.IsSEVSNPVMI(vmi) || util.IsSEVESVMI(vmi) {
		overhead.Add(resource.MustParse("256Mi"))
	}

	// Having a TPM device will spawn a swtpm process
	// In `ps`, swtpm has VSZ of 53808 and RSS of 3496, so 53Mi should do
	if tpm.HasDevice(&vmi.Spec) {
		overhead.Add(resource.MustParse("53Mi"))
	}

	if vmi.IsCPUDedicated() || vmi.WantsToHaveQOSGuaranteed() {
		overhead.Add(resource.MustParse("100Mi"))
	}

	// Multiplying the ratio is expected to be the last calculation before returning overhead
	if additionalOverheadRatio != nil && *additionalOverheadRatio != "" {
		ratio, err := strconv.ParseFloat(*additionalOverheadRatio, 64)
		if err != nil {
			// This error should never happen as it's already validated by webhooks
			log.Log.Warningf("cannot add additional overhead to virt infra overhead calculation: %v", err)
			return overhead
		}

		overhead = multiplyMemory(overhead, ratio)
	}

	return overhead
}

// calculatePagetableMemory calculates memory overhead for page tables (one bit for every 512b of RAM size)
func calculatePagetableMemory(vmiMemoryReq *resource.Quantity) resource.Quantity {
	pagetableMemory := resource.NewScaledQuantity(vmiMemoryReq.ScaledValue(resource.Kilo), resource.Kilo)
	pagetableMemory.Set(pagetableMemory.Value() / pageSize)
	return *pagetableMemory
}

// calculateVCPUOverhead calculates memory overhead based on vCPU count (8 MiB per vCPU)
func calculateVCPUOverhead(vmi *v1.VirtualMachineInstance) resource.Quantity {
	coresMemory := resource.MustParse("8Mi")
	vcpus := determineVCPUCount(vmi)
	value := coresMemory.Value() * vcpus
	return *resource.NewQuantity(value, coresMemory.Format)
}

// determineVCPUCount returns the number of vCPUs for the VMI
func determineVCPUCount(vmi *v1.VirtualMachineInstance) int64 {
	var vcpus int64

	if vmi.Spec.Domain.CPU != nil {
		vcpus = hardware.GetNumberOfVCPUs(vmi.Spec.Domain.CPU)
	} else {
		// Currently, a default guest CPU topology is set by the API webhook mutator, if not set by a user.
		// However, this wasn't always the case.
		// In case when the guest topology isn't set, take value from resources request or limits.
		resources := vmi.Spec.Domain.Resources
		if cpuLimit, ok := resources.Limits[k8sv1.ResourceCPU]; ok {
			vcpus = cpuLimit.Value()
		} else if cpuRequests, ok := resources.Requests[k8sv1.ResourceCPU]; ok {
			vcpus = cpuRequests.Value()
		}
	}

	// if neither CPU topology nor request or limits provided, set vcpus to 1
	if vcpus < 1 {
		vcpus = 1
	}
	return vcpus
}

func addProbeOverheads(vmi *v1.VirtualMachineInstance, quantity *resource.Quantity) {
	// We need to add this overhead due to potential issues when using exec probes.
	// In certain situations depending on things like node size and kernel versions
	// the exec probe can cause a significant memory overhead that results in the pod getting OOM killed.
	// To prevent this, we add this overhead until we have a better way of doing exec probes.
	// The virtProbeTotalAdditionalOverhead is added for the virt-probe binary we use for probing and
	// only added once, while the virtProbeOverhead is the general memory consumption of virt-probe
	// that we add per added probe.
	virtProbeTotalAdditionalOverhead := resource.MustParse("100Mi")
	virtProbeOverhead := resource.MustParse("10Mi")
	hasLiveness := vmi.Spec.LivenessProbe != nil && vmi.Spec.LivenessProbe.Exec != nil
	hasReadiness := vmi.Spec.ReadinessProbe != nil && vmi.Spec.ReadinessProbe.Exec != nil
	if hasLiveness {
		quantity.Add(virtProbeOverhead)
	}
	if hasReadiness {
		quantity.Add(virtProbeOverhead)
	}
	if hasLiveness || hasReadiness {
		quantity.Add(virtProbeTotalAdditionalOverhead)
	}
}

func multiplyMemory(mem resource.Quantity, multiplication float64) resource.Quantity {
	overheadAddition := float64(mem.ScaledValue(resource.Kilo)) * (multiplication - 1.0)
	additionalOverhead := resource.NewScaledQuantity(int64(overheadAddition), resource.Kilo)

	mem.Add(*additionalOverhead)
	return mem
}
