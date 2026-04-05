/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package requirements

import (
	"fmt"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

const (
	requiredArchitectureNotUsedErrFmt = "preference requires architecture %s but %s is being requested"
)

func checkArch(preferenceSpec *v1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) (conflict.Conflicts, error) {
	if vmiSpec.Architecture != *preferenceSpec.Requirements.Architecture {
		return conflict.Conflicts{conflict.New("spec", "template", "spec", "architecture")},
			fmt.Errorf(requiredArchitectureNotUsedErrFmt, *preferenceSpec.Requirements.Architecture, vmiSpec.Architecture)
	}
	return nil, nil
}
