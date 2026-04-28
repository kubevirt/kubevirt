/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
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
