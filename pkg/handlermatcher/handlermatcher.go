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

package handlermatcher

import (
	v1 "kubevirt.io/api/core/v1"
)

// MatchVMIToHandlerPool evaluates pools in order and returns the first matching
// pool for the given VMI, or nil if no pool matches.
func MatchVMIToHandlerPool(pools []v1.VirtHandlerPoolConfig, vmi *v1.VirtualMachineInstance) *v1.VirtHandlerPoolConfig {
	for i := range pools {
		if matchesPool(&pools[i], vmi) {
			return &pools[i]
		}
	}
	return nil
}

// GetLauncherImageForVMI returns the virt-launcher image for the given VMI.
// If a pool matches and has a custom launcher image, that image is returned.
// Otherwise the default launcher image is returned.
func GetLauncherImageForVMI(pools []v1.VirtHandlerPoolConfig, vmi *v1.VirtualMachineInstance, defaultImage string) string {
	pool := MatchVMIToHandlerPool(pools, vmi)
	if pool != nil && pool.VirtLauncherImage != "" {
		return pool.VirtLauncherImage
	}
	return defaultImage
}

// matchesPool checks if a VMI matches a single pool's selector.
// DeviceNames and VMLabels are OR'd: either matching is sufficient.
func matchesPool(pool *v1.VirtHandlerPoolConfig, vmi *v1.VirtualMachineInstance) bool {
	return matchesDeviceNames(pool.Selector.DeviceNames, vmi) ||
		matchesVMLabels(pool.Selector.VMLabels, vmi)
}

// matchesDeviceNames checks if any GPU or HostDevice DeviceName in the VMI
// matches one of the pool's configured device names.
func matchesDeviceNames(deviceNames []string, vmi *v1.VirtualMachineInstance) bool {
	if len(deviceNames) == 0 {
		return false
	}
	nameSet := make(map[string]struct{}, len(deviceNames))
	for _, n := range deviceNames {
		nameSet[n] = struct{}{}
	}
	for _, gpu := range vmi.Spec.Domain.Devices.GPUs {
		if _, ok := nameSet[gpu.DeviceName]; ok {
			return true
		}
	}
	for _, hd := range vmi.Spec.Domain.Devices.HostDevices {
		if _, ok := nameSet[hd.DeviceName]; ok {
			return true
		}
	}
	return false
}

// matchesVMLabels checks if all matchLabels are present on the VMI.
func matchesVMLabels(vmLabels *v1.VirtHandlerPoolVMLabels, vmi *v1.VirtualMachineInstance) bool {
	if vmLabels == nil || len(vmLabels.MatchLabels) == 0 {
		return false
	}
	vmiLabels := vmi.GetLabels()
	if vmiLabels == nil {
		return false
	}
	for k, v := range vmLabels.MatchLabels {
		if vmiLabels[k] != v {
			return false
		}
	}
	return true
}
