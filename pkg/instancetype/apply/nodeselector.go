/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package apply

import (
	"maps"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

func applyNodeSelector(
	baseConflict *conflict.Conflict,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) conflict.Conflicts {
	if instancetypeSpec.NodeSelector == nil {
		return nil
	}

	if vmiSpec.NodeSelector != nil {
		return conflict.Conflicts{baseConflict.NewChild("nodeSelector")}
	}

	vmiSpec.NodeSelector = maps.Clone(instancetypeSpec.NodeSelector)

	return nil
}
