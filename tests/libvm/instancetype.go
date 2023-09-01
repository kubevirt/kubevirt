package libvm

import (
	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
)

func WithInstancetype(name string) Option {
	return func(vm *virtv1.VirtualMachine) {
		removeConflictingResources(vm)
		vm.Spec.Instancetype.Name = name
		vm.Spec.Instancetype.Kind = instancetypeapi.SingularResourceName
	}
}

func WithClusterInstancetype(name string) Option {
	return func(vm *virtv1.VirtualMachine) {
		removeConflictingResources(vm)
		vm.Spec.Instancetype.Name = name
	}
}

func WithPreference(name string) Option {
	return func(vm *virtv1.VirtualMachine) {
		vm.Spec.Preference.Name = name
		vm.Spec.Preference.Kind = instancetypeapi.SingularPreferenceResourceName
	}
}

func WithClusterPreference(name string) Option {
	return func(vm *virtv1.VirtualMachine) {
		vm.Spec.Preference.Name = name
	}
}

func removeConflictingResources(vm *virtv1.VirtualMachine) {
	vm.Spec.Template.Spec.Domain.CPU = nil
	vm.Spec.Template.Spec.Domain.Memory = nil
	vm.Spec.Template.Spec.Domain.Resources = virtv1.ResourceRequirements{}
}
