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

func applyGPUs(
	baseConflict *conflict.Conflict,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) conflict.Conflicts {
	if len(instancetypeSpec.GPUs) == 0 {
		return nil
	}

	if len(vmiSpec.Domain.Devices.GPUs) > 0 {
		return conflict.Conflicts{baseConflict.NewChild("domain", "devices", "gpus")}
	}

	vmiSpec.Domain.Devices.GPUs = make([]virtv1.GPU, len(instancetypeSpec.GPUs))
	copy(vmiSpec.Domain.Devices.GPUs, instancetypeSpec.GPUs)

	return nil
}
