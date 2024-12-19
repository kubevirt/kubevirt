package controller

import (
	virtv1 "kubevirt.io/api/core/v1"
)

type mockController struct {
	syncFunc func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error)
}

func NewMockController() *mockController {
	return &mockController{
		syncFunc: func(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
			return vm, nil
		},
	}
}

func (m *mockController) Sync(vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) (*virtv1.VirtualMachine, error) {
	return m.syncFunc(vm, vmi)
}
