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

func applyIOThreadPolicy(
	baseConflict *conflict.Conflict,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) conflict.Conflicts {
	if instancetypeSpec.IOThreadsPolicy == nil {
		return nil
	}

	if vmiSpec.Domain.IOThreadsPolicy != nil {
		return conflict.Conflicts{baseConflict.NewChild("domain", "ioThreadsPolicy")}
	}

	instancetypeIOThreadPolicy := *instancetypeSpec.IOThreadsPolicy
	vmiSpec.Domain.IOThreadsPolicy = &instancetypeIOThreadPolicy

	return nil
}
