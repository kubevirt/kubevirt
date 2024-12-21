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
 * Copyright The KubeVirt Authors
 *
 */
package apply

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype/conflict"
	preferenceApply "kubevirt.io/kubevirt/pkg/instancetype/preference/apply"
)

type preferenceApplier interface {
	Apply(*v1beta1.VirtualMachinePreferenceSpec, *virtv1.VirtualMachineInstanceSpec, *metav1.ObjectMeta)
}

type vmiApplier struct {
	preferenceApplier preferenceApplier
}

func NewVMIApplier() *vmiApplier {
	return &vmiApplier{
		preferenceApplier: preferenceApply.New(),
	}
}

func (a *vmiApplier) ApplyToVMI(
	field *k8sfield.Path,
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	preferenceSpec *v1beta1.VirtualMachinePreferenceSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
	vmiMetadata *metav1.ObjectMeta,
) conflict.Conflicts {
	if instancetypeSpec == nil && preferenceSpec == nil {
		return nil
	}

	if instancetypeSpec != nil {
		baseConflict := conflict.NewFromPath(field)
		conflicts := conflict.Conflicts{}
		conflicts = append(conflicts, applyNodeSelector(baseConflict, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applySchedulerName(baseConflict, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyCPU(baseConflict, instancetypeSpec, preferenceSpec, vmiSpec)...)
		conflicts = append(conflicts, applyMemory(baseConflict, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyIOThreadPolicy(baseConflict, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyLaunchSecurity(baseConflict, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyGPUs(baseConflict, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyHostDevices(baseConflict, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyInstanceTypeAnnotations(instancetypeSpec.Annotations, vmiMetadata)...)
		if len(conflicts) > 0 {
			return conflicts
		}
	}

	a.preferenceApplier.Apply(preferenceSpec, vmiSpec, vmiMetadata)

	return nil
}
