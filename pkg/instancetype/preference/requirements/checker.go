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
