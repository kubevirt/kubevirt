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
 */

package vm

import (
	virtv1 "kubevirt.io/api/core/v1"
)

type mockController struct {
	syncFunc                       func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
	applyToVMFunc                  func(*virtv1.VirtualMachine) error
	applyToVMIFunc                 func(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) error
	applyAutoAttachPreferencesFunc func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error
}

func NewMockController() *mockController {
	return &mockController{
		syncFunc: func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
			return vm, nil
		},
		applyToVMFunc: func(*virtv1.VirtualMachine) error {
			return nil
		},
		applyToVMIFunc: func(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) error {
			return nil
		},
		applyAutoAttachPreferencesFunc: func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
			return nil
		},
	}
}

func (m *mockController) ApplyToVM(vm *virtv1.VirtualMachine) error {
	return m.applyToVMFunc(vm)
}

func (m *mockController) ApplyToVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	return m.applyToVMIFunc(vm, vmi)
}

func (m *mockController) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
	return m.syncFunc(vm, vmi)
}

func (m *mockController) ApplyAutoAttachPreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	return m.applyAutoAttachPreferencesFunc(vm, vmi)
}
