/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package apply

import (
	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/pointer"
)

func applyIOThreads(
	baseConflict *conflict.Conflict,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) conflict.Conflicts {
	if instancetypeSpec.IOThreads == nil || instancetypeSpec.IOThreads.SupplementalPoolThreadCount == nil {
		return nil
	}

	if vmiSpec.Domain.IOThreads != nil && vmiSpec.Domain.IOThreads.SupplementalPoolThreadCount != nil {
		return conflict.Conflicts{baseConflict.NewChild("domain", "ioThreads", "supplementalPoolThreadCount")}
	}

	if vmiSpec.Domain.IOThreads == nil {
		vmiSpec.Domain.IOThreads = &virtv1.DiskIOThreads{}
	}

	vmiSpec.Domain.IOThreads.SupplementalPoolThreadCount = pointer.P(*instancetypeSpec.IOThreads.SupplementalPoolThreadCount)

	return nil
}
