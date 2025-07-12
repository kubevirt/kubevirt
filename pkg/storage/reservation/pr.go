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

package reservation

import (
	"path/filepath"

	v1 "kubevirt.io/api/core/v1"
)

const (
	sourceDaemonsPath     = "/var/run/kubevirt/daemons"
	hostSourceDaemonsPath = "/proc/1/root" + sourceDaemonsPath
	prHelperDir           = "pr"
	prHelperSocket        = "pr-helper.sock"
	prResourceName        = "pr-helper"
)

func GetPrResourceName() string {
	return prResourceName
}

func GetPrHelperSocketDir() string {
	return filepath.Join(sourceDaemonsPath, prHelperDir)
}

func GetPrHelperHostSocketDir() string {
	return filepath.Join(hostSourceDaemonsPath, prHelperDir)
}

func GetPrHelperSocketPath() string {
	return filepath.Join(GetPrHelperSocketDir(), prHelperSocket)
}

func GetPrHelperSocket() string {
	return prHelperSocket
}

func HasVMIPersistentReservation(vmi *v1.VirtualMachineInstance) bool {
	return HasVMISpecPersistentReservation(&vmi.Spec)
}

func HasVMISpecPersistentReservation(vmiSpec *v1.VirtualMachineInstanceSpec) bool {
	for _, disk := range vmiSpec.Domain.Devices.Disks {
		if disk.DiskDevice.LUN != nil && disk.DiskDevice.LUN.Reservation {
			return true
		}
	}
	return false
}
