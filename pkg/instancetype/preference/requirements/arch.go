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
