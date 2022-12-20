package testutils

import (
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"

	"kubevirt.io/kubevirt/pkg/instancetype"
)

type MockInstancetypeMethods struct {
	FindInstancetypeSpecFunc     func(vm *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error)
	ApplyToVmiFunc               func(field *k8sfield.Path, instancetypespec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) instancetype.Conflicts
	FindPreferenceSpecFunc       func(vm *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error)
	StoreControllerRevisionsFunc func(vm *v1.VirtualMachine) error
	InferDefaultInstancetypeFunc func(vm *v1.VirtualMachine) (*v1.InstancetypeMatcher, error)
	InferDefaultPreferenceFunc   func(vm *v1.VirtualMachine) (*v1.PreferenceMatcher, error)
}

var _ instancetype.Methods = &MockInstancetypeMethods{}

func (m *MockInstancetypeMethods) FindInstancetypeSpec(vm *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error) {
	return m.FindInstancetypeSpecFunc(vm)
}

func (m *MockInstancetypeMethods) ApplyToVmi(field *k8sfield.Path, instancetypespec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) instancetype.Conflicts {
	return m.ApplyToVmiFunc(field, instancetypespec, preferenceSpec, vmiSpec)
}

func (m *MockInstancetypeMethods) FindPreferenceSpec(vm *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error) {
	return m.FindPreferenceSpecFunc(vm)
}

func (m *MockInstancetypeMethods) StoreControllerRevisions(vm *v1.VirtualMachine) error {
	return m.StoreControllerRevisionsFunc(vm)
}

func (m *MockInstancetypeMethods) InferDefaultInstancetype(vm *v1.VirtualMachine) (*v1.InstancetypeMatcher, error) {
	return m.InferDefaultInstancetypeFunc(vm)
}

func (m *MockInstancetypeMethods) InferDefaultPreference(vm *v1.VirtualMachine) (*v1.PreferenceMatcher, error) {
	return m.InferDefaultPreferenceFunc(vm)
}

func NewMockInstancetypeMethods() *MockInstancetypeMethods {
	return &MockInstancetypeMethods{
		FindInstancetypeSpecFunc: func(_ *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error) {
			return nil, nil
		},
		ApplyToVmiFunc: func(_ *k8sfield.Path, _ *instancetypev1alpha2.VirtualMachineInstancetypeSpec, _ *instancetypev1alpha2.VirtualMachinePreferenceSpec, _ *v1.VirtualMachineInstanceSpec) instancetype.Conflicts {
			return nil
		},
		FindPreferenceSpecFunc: func(_ *v1.VirtualMachine) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error) {
			return nil, nil
		},
		StoreControllerRevisionsFunc: func(_ *v1.VirtualMachine) error {
			return nil
		},
		InferDefaultInstancetypeFunc: func(_ *v1.VirtualMachine) (*v1.InstancetypeMatcher, error) {
			return nil, nil
		},
		InferDefaultPreferenceFunc: func(_ *v1.VirtualMachine) (*v1.PreferenceMatcher, error) {
			return nil, nil
		},
	}
}
