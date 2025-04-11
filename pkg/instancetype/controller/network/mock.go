package network

import (
	virtv1 "kubevirt.io/api/core/v1"
)

type mockController struct {
	applyFunc func(vm *virtv1.VirtualMachine, spec *virtv1.VirtualMachineInstanceSpec) *virtv1.VirtualMachineInstanceSpec
}

func NewMockController() *mockController {
	return &mockController{
		applyFunc: func(_ *virtv1.VirtualMachine, spec *virtv1.VirtualMachineInstanceSpec) *virtv1.VirtualMachineInstanceSpec {
			return spec.DeepCopy()
		},
	}
}

func (m *mockController) ApplyInterfacePreferencesToVMI(
	vm *virtv1.VirtualMachine,
	spec *virtv1.VirtualMachineInstanceSpec,
) *virtv1.VirtualMachineInstanceSpec {
	return m.applyFunc(vm, spec)
}
