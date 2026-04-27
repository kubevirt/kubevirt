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

package poolmatcher

import (
	v1 "kubevirt.io/api/core/v1"
)

// MatchVMIToWorkerPool evaluates each pool in order and returns the first
// pool whose selector matches the given VMI, or nil if no pool matches.
func MatchVMIToWorkerPool(pools []v1.WorkerPoolConfig, vmi *v1.VirtualMachineInstance) *v1.WorkerPoolConfig {
	for i := range pools {
		if matchesPool(&pools[i], vmi) {
			return &pools[i]
		}
	}
	return nil
}

// GetLauncherImageForVMI returns the pool-specific launcher image if the VMI
// matches a pool with a VirtLauncherImage override, otherwise returns the
// default image.
func GetLauncherImageForVMI(pools []v1.WorkerPoolConfig, vmi *v1.VirtualMachineInstance, defaultImage string) string {
	pool := MatchVMIToWorkerPool(pools, vmi)
	if pool != nil && pool.VirtLauncherImage != "" {
		return pool.VirtLauncherImage
	}
	return defaultImage
}

func matchesPool(pool *v1.WorkerPoolConfig, vmi *v1.VirtualMachineInstance) bool {
	return matchesDeviceNames(pool.Selector.DeviceNames, vmi) ||
		matchesVMLabels(pool.Selector.VMLabels, vmi)
}

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

func matchesVMLabels(vmLabels *v1.WorkerPoolVMLabels, vmi *v1.VirtualMachineInstance) bool {
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
