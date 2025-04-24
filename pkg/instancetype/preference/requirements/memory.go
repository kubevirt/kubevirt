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

	"k8s.io/apimachinery/pkg/api/resource"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	InsufficientInstanceTypeMemoryResourcesErrorFmt = "insufficient Memory resources of %s provided by instance type, preference requires %s"
	InsufficientVMMemoryResourcesErrorFmt           = "insufficient Memory resources of %s provided by VirtualMachine, preference requires %s"
)

func checkMemory(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
) (conflict.Conflicts, error) {
	errFmt := InsufficientVMMemoryResourcesErrorFmt
	errConflict := conflict.New("spec", "template", "spec", "domain", "memory")
	providedMemory := pointer.P(resource.MustParse("0Mi"))

	if instancetypeSpec != nil {
		errConflict = conflict.New("spec", "instancetype")
		errFmt = InsufficientInstanceTypeMemoryResourcesErrorFmt
		providedMemory = &instancetypeSpec.Memory.Guest
	}

	if vmiSpec != nil && vmiSpec.Domain.Memory != nil && vmiSpec.Domain.Memory.Guest != nil {
		providedMemory = vmiSpec.Domain.Memory.Guest
	}

	if providedMemory.Cmp(preferenceSpec.Requirements.Memory.Guest) < 0 {
		return conflict.Conflicts{errConflict}, fmt.Errorf(errFmt, providedMemory.String(), preferenceSpec.Requirements.Memory.Guest.String())
	}

	return nil, nil
}
