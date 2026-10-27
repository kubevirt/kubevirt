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
	"fmt"

	v1 "kubevirt.io/api/core/v1"

	netvmispec "kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/storage/memorydump"
	storagevmispec "kubevirt.io/kubevirt/pkg/storage/vmispec"
)

// networkConfig provides the configuration needed during VMI preparation.
// *virtconfig.ClusterConfig and the offline config both satisfy it.
type networkConfig interface {
	GetDefaultNetworkInterface() string
	IsBridgeInterfaceOnPodNetworkEnabled() bool
}

// instancetypeHandler applies preferences before attaching default devices,
// then expands the instancetype and preference onto the prepared VMI.
type instancetypeHandler interface {
	ApplyAutoAttachPreferences(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error
	ApplyToVMI(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error
}

type vmiRenderer struct {
	config       networkConfig
	instancetype instancetypeHandler
}

func validateVM(vm *v1.VirtualMachine) error {
	if vm == nil {
		return fmt.Errorf("VM is nil")
	}
	if vm.Spec.Template == nil {
		return fmt.Errorf("VM %q has no template spec", vm.Name)
	}
	return nil
}

// vmiFromVM owns the preparation order for both renderers. Admission belongs
// outside this flow: the online caller uses the API server, and the offline
// renderer applies VM mutations before this call and VMI mutations after it.
func (r *vmiRenderer) vmiFromVM(vm *v1.VirtualMachine) (*v1.VirtualMachineInstance, error) {
	if err := validateVM(vm); err != nil {
		return nil, err
	}

	vmi := NewVMI(vm)
	prepareControllerVMI(vmi)
	if err := r.instancetype.ApplyAutoAttachPreferences(vm, vmi); err != nil {
		return nil, fmt.Errorf("failed to apply device preferences: %w", err)
	}
	if err := attachDefaultDevices(vmi, r.config); err != nil {
		return nil, err
	}
	if err := r.instancetype.ApplyToVMI(vm, vmi); err != nil {
		return nil, fmt.Errorf("failed to apply instancetype and preference: %w", err)
	}
	return vmi, nil
}

// prepareControllerVMI drops deprecated memory-dump volumes and pairs disks
// before either renderer applies preferences to those disks.
func prepareControllerVMI(vmi *v1.VirtualMachineInstance) {
	memorydump.RemoveDeprecatedVolumes(&vmi.Spec)
	storagevmispec.SetDefaultVolumeDisk(&vmi.Spec)
}

// attachDefaultDevices adds the default input and pod interface. Call it
// after auto-attach preferences so those preferences are already visible.
func attachDefaultDevices(vmi *v1.VirtualMachineInstance, cfg networkConfig) error {
	AutoAttachInputDevice(vmi)
	if err := netvmispec.SetDefaultNetworkInterface(cfg, &vmi.Spec); err != nil {
		return fmt.Errorf("failed to set default network interface: %w", err)
	}
	return nil
}
