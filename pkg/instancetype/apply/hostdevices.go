/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
