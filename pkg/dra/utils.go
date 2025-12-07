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

// IsAllDRAHostDevicesReconciled checks if all HostDevices with DRA in the VMI spec have corresponding status entries populated
// with either a PCI address (e.g., SR-IOV) or an mdev UUID when mediated devices are used. It mirrors the semantics of
// IsAllDRAGPUsReconciled but operates on spec.domain.devices.hostDevices instead of GPUs.
func IsAllDRAHostDevicesReconciled(vmi *v1.VirtualMachineInstance, status *v1.DeviceStatus) bool {
	draDeviceNames := make(map[string]struct{})

	// Collect DRA host devices
	for _, hd := range vmi.Spec.Domain.Devices.HostDevices {
		if hd.ClaimRequest != nil {
			draDeviceNames[hd.Name] = struct{}{}
		}
	}

	// Collect DRA networks
	for _, net := range vmi.Spec.Networks {
		if IsNetworkDRA(net) {
			draDeviceNames[net.Name] = struct{}{}
		}
	}

	if len(draDeviceNames) == 0 {
		return true
	}

	reconciledCount := 0
	if status != nil {
		for _, hdStatus := range status.HostDeviceStatuses {
			if _, isDRADev := draDeviceNames[hdStatus.Name]; !isDRADev {
				continue
			}
			if hdStatus.DeviceResourceClaimStatus != nil &&
				hdStatus.DeviceResourceClaimStatus.ResourceClaimName != nil &&
				hdStatus.DeviceResourceClaimStatus.Name != nil &&
				hdStatus.DeviceResourceClaimStatus.Attributes != nil &&
				(hdStatus.DeviceResourceClaimStatus.Attributes.PCIAddress != nil ||
					hdStatus.DeviceResourceClaimStatus.Attributes.MDevUUID != nil) {
				reconciledCount++
			}
		}
	}
	return reconciledCount == len(draDeviceNames)
}

// IsGPUDRA returns true if the GPU is a DRA GPU
func IsGPUDRA(gpu v1.GPU) bool {
	return gpu.DeviceName == "" && gpu.ClaimRequest != nil
}

// IsHostDeviceDRA returns true if the HostDevice is a DRA GPU
func IsHostDeviceDRA(hd v1.HostDevice) bool {
	return hd.DeviceName == "" && hd.ClaimRequest != nil
}

// IsNetworkDRA returns true if the Network is a DRA network
func IsNetworkDRA(net v1.Network) bool {
	return net.NetworkSource.ResourceClaim != nil
}

// HasNetworkDRA returns true if the VMI has any DRA networks
func HasNetworkDRA(vmi *v1.VirtualMachineInstance) bool {
	for _, net := range vmi.Spec.Networks {
		if IsNetworkDRA(net) {
			return true
		}
	}
	return false
}
