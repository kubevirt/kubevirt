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
 * Copyright The KubeVirt Authors
 *
 */
package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

func applyHostDevices(
	baseConflict *conflict.Conflict,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) conflict.Conflicts {
	if len(instancetypeSpec.HostDevices) == 0 {
		return nil
	}

	if len(vmiSpec.Domain.Devices.HostDevices) > 0 {
		return conflict.Conflicts{baseConflict.NewChild("domain", "devices", "hostDevices")}
	}

	vmiSpec.Domain.Devices.HostDevices = make([]virtv1.HostDevice, len(instancetypeSpec.HostDevices))
	copy(vmiSpec.Domain.Devices.HostDevices, instancetypeSpec.HostDevices)

	return nil
}
