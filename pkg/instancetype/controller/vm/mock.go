package controller

import (
	virtv1 "kubevirt.io/api/core/v1"
)

type mockController struct {
	syncFunc                   func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
	applyToVMFunc              func(*virtv1.VirtualMachine) error
	applyToVMIFunc             func(*virtv1.VirtualMachine, *virtv1.VirtualMachineInstance) error
	applyDevicePreferencesFunc func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error
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
		applyDevicePreferencesFunc: func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
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

func (m *mockController) ApplyDevicePreferences(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) error {
	return m.applyDevicePreferencesFunc(vm, vmi)
}
