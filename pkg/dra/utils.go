package dra

import v1 "kubevirt.io/api/core/v1"

// IsAllDRAGPUsReconciled checks if all GPUs with DRA in the VMI spec have corresponding status entries populated
// with either a PCI address (pGPU) or an mdev UUID (vGPU).  It is used by both virt-handler and virt-controller
// to decide whether GPU-related DRA reconciliation is complete.
func IsAllDRAGPUsReconciled(vmi *v1.VirtualMachineInstance, status *v1.DeviceStatus) bool {
	draGPUNames := make(map[string]struct{})
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if gpu.ClaimRequest != nil {
			draGPUNames[gpu.Name] = struct{}{}
		}
	}
	if len(draGPUNames) == 0 {
		return true
	}

	reconciledCount := 0
	if status != nil {
		for _, gpuStatus := range status.GPUStatuses {
			if _, isDRAGPU := draGPUNames[gpuStatus.Name]; !isDRAGPU {
				continue
			}

			if gpuStatus.DeviceResourceClaimStatus != nil &&
				gpuStatus.DeviceResourceClaimStatus.ResourceClaimName != nil &&
				gpuStatus.DeviceResourceClaimStatus.Name != nil &&
				gpuStatus.DeviceResourceClaimStatus.Attributes != nil &&
				(gpuStatus.DeviceResourceClaimStatus.Attributes.PCIAddress != nil ||
					gpuStatus.DeviceResourceClaimStatus.Attributes.MDevUUID != nil) {
				reconciledCount++
			}
		}
	}
	return reconciledCount == len(draGPUNames)
}
