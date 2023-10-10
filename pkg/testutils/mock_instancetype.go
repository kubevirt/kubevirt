package testutils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"

	"kubevirt.io/kubevirt/pkg/instancetype"
)

type MockInstancetypeMethods struct {
	FindInstancetypeSpecFunc        func(vm *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error)
	ApplyToVmiFunc                  func(field *k8sfield.Path, instancetypespec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) instancetype.Conflicts
	FindPreferenceSpecFunc          func(vm *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error)
	StoreControllerRevisionsFunc    func(vm *v1.VirtualMachine) error
	InferDefaultInstancetypeFunc    func(vm *v1.VirtualMachine) error
	InferDefaultPreferenceFunc      func(vm *v1.VirtualMachine) error
	CheckPreferenceRequirementsFunc func(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) (instancetype.Conflicts, error)
}

var _ instancetype.Methods = &MockInstancetypeMethods{}

func (m *MockInstancetypeMethods) FindInstancetypeSpec(vm *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
	return m.FindInstancetypeSpecFunc(vm)
}

func (m *MockInstancetypeMethods) ApplyToVmi(field *k8sfield.Path, instancetypespec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec, vmiMetadata *metav1.ObjectMeta) instancetype.Conflicts {
	return m.ApplyToVmiFunc(field, instancetypespec, preferenceSpec, vmiSpec, vmiMetadata)
}

func (m *MockInstancetypeMethods) FindPreferenceSpec(vm *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
	return m.FindPreferenceSpecFunc(vm)
}

func (m *MockInstancetypeMethods) StoreControllerRevisions(vm *v1.VirtualMachine) error {
	return m.StoreControllerRevisionsFunc(vm)
}

func (m *MockInstancetypeMethods) InferDefaultInstancetype(vm *v1.VirtualMachine) error {
	return m.InferDefaultInstancetypeFunc(vm)
}

func (m *MockInstancetypeMethods) InferDefaultPreference(vm *v1.VirtualMachine) error {
	return m.InferDefaultPreferenceFunc(vm)
}

func (m *MockInstancetypeMethods) CheckPreferenceRequirements(instancetypeSpec *instancetypev1beta1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1beta1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) (instancetype.Conflicts, error) {
	return m.CheckPreferenceRequirementsFunc(instancetypeSpec, preferenceSpec, vmiSpec)
}

func NewMockInstancetypeMethods() *MockInstancetypeMethods {
	return &MockInstancetypeMethods{
		FindInstancetypeSpecFunc: func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachineInstancetypeSpec, error) {
			return nil, nil
		},
		ApplyToVmiFunc: func(_ *k8sfield.Path, _ *instancetypev1beta1.VirtualMachineInstancetypeSpec, _ *instancetypev1beta1.VirtualMachinePreferenceSpec, _ *v1.VirtualMachineInstanceSpec, _ *metav1.ObjectMeta) instancetype.Conflicts {
			return nil
		},
		FindPreferenceSpecFunc: func(_ *v1.VirtualMachine) (*instancetypev1beta1.VirtualMachinePreferenceSpec, error) {
			return nil, nil
		},
		StoreControllerRevisionsFunc: func(_ *v1.VirtualMachine) error {
			return nil
		},
		InferDefaultInstancetypeFunc: func(_ *v1.VirtualMachine) error {
			return nil
		},
		InferDefaultPreferenceFunc: func(_ *v1.VirtualMachine) error {
			return nil
		},
		CheckPreferenceRequirementsFunc: func(_ *instancetypev1beta1.VirtualMachineInstancetypeSpec, _ *instancetypev1beta1.VirtualMachinePreferenceSpec, _ *v1.VirtualMachineInstanceSpec) (instancetype.Conflicts, error) {
			return nil, nil
		},
	}
}
