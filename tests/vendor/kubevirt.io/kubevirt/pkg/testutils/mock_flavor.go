package testutils

import (
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

	"kubevirt.io/kubevirt/pkg/instancetype"
)

type MockInstancetypeMethods struct {
	FindInstancetypeSpecFunc     func(vm *v1.VirtualMachine) (*instancetypev1alpha1.VirtualMachineInstancetypeSpec, error)
	ApplyToVmiFunc               func(field *k8sfield.Path, instancetypespec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) instancetype.Conflicts
	FindPreferenceSpecFunc       func(vm *v1.VirtualMachine) (*instancetypev1alpha1.VirtualMachinePreferenceSpec, error)
	StoreControllerRevisionsFunc func(vm *v1.VirtualMachine) error
}

var _ instancetype.Methods = &MockInstancetypeMethods{}

func (m *MockInstancetypeMethods) FindInstancetypeSpec(vm *v1.VirtualMachine) (*instancetypev1alpha1.VirtualMachineInstancetypeSpec, error) {
	return m.FindInstancetypeSpecFunc(vm)
}

func (m *MockInstancetypeMethods) ApplyToVmi(field *k8sfield.Path, instancetypespec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) instancetype.Conflicts {
	return m.ApplyToVmiFunc(field, instancetypespec, preferenceSpec, vmiSpec)
}

func (m *MockInstancetypeMethods) FindPreferenceSpec(vm *v1.VirtualMachine) (*instancetypev1alpha1.VirtualMachinePreferenceSpec, error) {
	return m.FindPreferenceSpecFunc(vm)
}

func (m *MockInstancetypeMethods) StoreControllerRevisions(vm *v1.VirtualMachine) error {
	return m.StoreControllerRevisionsFunc(vm)
}

func NewMockInstancetypeMethods() *MockInstancetypeMethods {
	return &MockInstancetypeMethods{
		FindInstancetypeSpecFunc: func(_ *v1.VirtualMachine) (*instancetypev1alpha1.VirtualMachineInstancetypeSpec, error) {
			return nil, nil
		},
		ApplyToVmiFunc: func(_ *k8sfield.Path, _ *instancetypev1alpha1.VirtualMachineInstancetypeSpec, _ *instancetypev1alpha1.VirtualMachinePreferenceSpec, _ *v1.VirtualMachineInstanceSpec) instancetype.Conflicts {
			return nil
		},
		FindPreferenceSpecFunc: func(_ *v1.VirtualMachine) (*instancetypev1alpha1.VirtualMachinePreferenceSpec, error) {
			return nil, nil
		},
		StoreControllerRevisionsFunc: func(_ *v1.VirtualMachine) error {
			return nil
		},
	}
}
