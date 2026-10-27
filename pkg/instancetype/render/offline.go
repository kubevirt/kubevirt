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

	virtv1 "kubevirt.io/api/core/v1"
	v1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

type offlineRenderer struct {
	*renderer
	instancetype *v1beta1.VirtualMachineInstancetypeSpec
	preference   *v1beta1.VirtualMachinePreferenceSpec
}

// NewOffline uses supplied specs with the same VMI application flow as New.
// ValidateVM checks that the specs match the VM's matchers.
func NewOffline(
	instancetype *v1beta1.VirtualMachineInstancetypeSpec,
	preference *v1beta1.VirtualMachinePreferenceSpec,
) *offlineRenderer {
	h := &offlineRenderer{instancetype: instancetype, preference: preference}
	h.renderer = New(h, h)
	return h
}

func (h *offlineRenderer) ValidateVM(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Instancetype != nil && vm.Spec.Instancetype.InferFromVolume != "" {
		return fmt.Errorf("offline render does not infer instancetype from a volume")
	}
	if vm.Spec.Preference != nil && vm.Spec.Preference.InferFromVolume != "" {
		return fmt.Errorf("offline render does not infer preference from a volume")
	}
	if vm.Spec.Instancetype != nil && h.instancetype == nil {
		return fmt.Errorf("offline render does not apply instancetype matchers")
	}
	if vm.Spec.Instancetype == nil && h.instancetype != nil {
		return fmt.Errorf("instancetype spec was provided but the VM has no instancetype matcher")
	}
	if vm.Spec.Preference != nil && h.preference == nil {
		return fmt.Errorf("offline render does not apply preference matchers")
	}
	if vm.Spec.Preference == nil && h.preference != nil {
		return fmt.Errorf("preference spec was provided but the VM has no preference matcher")
	}
	return nil
}

func (h *offlineRenderer) Find(_ *virtv1.VirtualMachine) (*v1beta1.VirtualMachineInstancetypeSpec, error) {
	return h.instancetype.DeepCopy(), nil
}

func (h *offlineRenderer) FindPreference(_ *virtv1.VirtualMachine) (*v1beta1.VirtualMachinePreferenceSpec, error) {
	return h.preference.DeepCopy(), nil
}
