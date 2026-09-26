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
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"

	storagevmispec "kubevirt.io/kubevirt/pkg/storage/vmispec"
)

// firmwareUUIDns is the namespace used to derive a stable VMI firmware UUID
// from the VM name. It must stay in lockstep with the value historically
// used by the VM controller.
const firmwareUUIDns = "6a1a24a1-4061-4607-8bf4-a3963d0c5895"

var firmwareNamespace = uuid.MustParse(firmwareUUIDns)

// FirmwareUUID returns the stable firmware UUID derived from a VM/VMI name.
// The UUID does not change across reboots of the same VM.
func FirmwareUUID(name string) types.UID {
	return types.UID(uuid.NewSHA1(firmwareNamespace, []byte(name)).String())
}

// NewVMI copies a VirtualMachine template into a VirtualMachineInstance:
// name, namespace, labels, owner references, a stable firmware UUID, and
// volume-disk pairing. Cluster-specific start-paused and memory-dump
// handling stays in the VM controller.
//
// The caller must ensure vm.Spec.Template is non-nil.
func NewVMI(vm *virtv1.VirtualMachine) *virtv1.VirtualMachineInstance {
	vmi := &virtv1.VirtualMachineInstance{
		TypeMeta: metav1.TypeMeta{
			APIVersion: virtv1.GroupVersion.String(),
			Kind:       "VirtualMachineInstance",
		},
	}
	vmi.ObjectMeta = *vm.Spec.Template.ObjectMeta.DeepCopy()
	vmi.ObjectMeta.Name = vm.ObjectMeta.Name
	vmi.ObjectMeta.GenerateName = ""
	vmi.ObjectMeta.Namespace = vm.ObjectMeta.Namespace
	vmi.Spec = *vm.Spec.Template.Spec.DeepCopy()

	vmi.ObjectMeta.Labels = vm.Spec.Template.ObjectMeta.Labels
	vmi.ObjectMeta.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind),
	}

	if vmi.Spec.Domain.Firmware == nil {
		vmi.Spec.Domain.Firmware = &virtv1.Firmware{}
	}
	if vmi.Spec.Domain.Firmware.UUID == "" {
		vmi.Spec.Domain.Firmware.UUID = FirmwareUUID(vmi.Name)
	}

	storagevmispec.SetDefaultVolumeDisk(&vmi.Spec)
	return vmi
}

// AutoAttachInputDevice adds a default input device when AutoattachInputDevice
// is true and the VMI has no inputs yet. Bus and type defaults are applied
// later by the VMI mutation webhook.
func AutoAttachInputDevice(vmi *virtv1.VirtualMachineInstance) {
	autoAttachInput := vmi.Spec.Domain.Devices.AutoattachInputDevice
	if autoAttachInput == nil || !*autoAttachInput || len(vmi.Spec.Domain.Devices.Inputs) > 0 {
		return
	}
	vmi.Spec.Domain.Devices.Inputs = append(
		vmi.Spec.Domain.Devices.Inputs,
		virtv1.Input{Name: "default-0"},
	)
}
