/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package vm

import (
	virtv1 "kubevirt.io/api/core/v1"
)

type controllerStub struct {
	syncFunc                       func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
	applyToVMFunc                  func(*virtv1.VirtualMachine) error
	applyToVMIFunc                 func(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) error
	applyAutoAttachPreferencesFunc func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error
}

func NewControllerStub() *controllerStub {
	return &controllerStub{
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

func (m *controllerStub) ApplyToVM(vm *virtv1.VirtualMachine) error {
	return m.applyToVMFunc(vm)
}

func (m *controllerStub) ApplyToVMI(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	return m.applyToVMIFunc(vm, vmi)
}

func (m *controllerStub) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
	return m.syncFunc(vm, vmi)
}

func (m *controllerStub) ApplyAutoAttachPreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	return m.applyAutoAttachPreferencesFunc(vm, vmi)
}
