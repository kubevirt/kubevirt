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

package render

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

// NewVMI copies a VirtualMachine template into a VirtualMachineInstance:
// name, namespace, labels, and owner references. The template firmware
// UUID is copied as-is. Volume-disk pairing happens after memory-dump
// cleanup in the VM controller, and again in VMI mutations, so a
// completed dump cannot leave an orphan disk.
//
// The caller must ensure vm.Spec.Template is non-nil.
func NewVMI(vm *v1.VirtualMachine) *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.GroupVersion.String(),
			Kind:       "VirtualMachineInstance",
		},
	}
	vmi.ObjectMeta = *vm.Spec.Template.ObjectMeta.DeepCopy()
	vmi.ObjectMeta.Name = vm.ObjectMeta.Name
	vmi.ObjectMeta.GenerateName = ""
	vmi.ObjectMeta.Namespace = vm.ObjectMeta.Namespace
	vmi.Spec = *vm.Spec.Template.Spec.DeepCopy()

	vmi.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(vm, v1.VirtualMachineGroupVersionKind),
	}

	return vmi
}

// AutoAttachInputDevice adds a default input device when AutoattachInputDevice
// is true and the VMI has no inputs yet. Bus and type defaults are applied
// later by the VMI mutation webhook.
func AutoAttachInputDevice(vmi *v1.VirtualMachineInstance) {
	autoAttachInput := vmi.Spec.Domain.Devices.AutoattachInputDevice
	if autoAttachInput == nil || !*autoAttachInput || len(vmi.Spec.Domain.Devices.Inputs) > 0 {
		return
	}
	vmi.Spec.Domain.Devices.Inputs = append(
		vmi.Spec.Domain.Devices.Inputs,
		v1.Input{Name: "default-0"},
	)
}
