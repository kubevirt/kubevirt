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

package iothreads

import (
	"slices"

	v1 "kubevirt.io/api/core/v1"
)

func HasIOThreads(vmi *v1.VirtualMachineInstance) bool {
	if vmi.Spec.Domain.IOThreadsPolicy != nil {
		return true
	}
	return slices.ContainsFunc(vmi.Spec.Domain.Devices.Disks, HasDedicatedIOThread)
}

func HasDedicatedIOThread(disk v1.Disk) bool {
	return disk.DedicatedIOThread != nil && *disk.DedicatedIOThread
}

func SupplementalPoolThreadCount(vmi *v1.VirtualMachineInstance) int {
	if vmi.Spec.Domain.IOThreads == nil || vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount == nil {
		return 0
	}
	return int(*vmi.Spec.Domain.IOThreads.SupplementalPoolThreadCount)
}
