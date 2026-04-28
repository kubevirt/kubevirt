/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package requirements

import (
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
)

type checker struct{}

func New() *checker {
	return &checker{}
}

func (c *checker) Check(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflict.Conflicts, error) {
	if preferenceSpec == nil || preferenceSpec.Requirements == nil {
		return nil, nil
	}

	if preferenceSpec.Requirements.CPU != nil {
		if conflicts, err := checkCPU(instancetypeSpec, preferenceSpec, vmiSpec); err != nil {
			return conflicts, err
		}
	}

	if preferenceSpec.Requirements.Memory != nil {
		if conflicts, err := checkMemory(instancetypeSpec, preferenceSpec, vmiSpec); err != nil {
			return conflicts, err
		}
	}

	if preferenceSpec.Requirements.Architecture != nil {
		if conflicts, err := checkArch(preferenceSpec, vmiSpec); err != nil {
			return conflicts, err
		}
	}

	return nil, nil
}
