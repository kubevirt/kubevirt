package testutils

import (
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/client-go/apis/core/v1"
	flavorv1alpha1 "kubevirt.io/client-go/apis/flavor/v1alpha1"
	"kubevirt.io/kubevirt/pkg/flavor"
)

type MockFlavorMethods struct {
	FindFlavorFunc func(vm *v1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error)
	ApplyToVmiFunc func(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *v1.VirtualMachineInstanceSpec) flavor.Conflicts
}

func (m *MockFlavorMethods) FindProfile(vm *v1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
	return m.FindFlavorFunc(vm)
}

func (m *MockFlavorMethods) ApplyToVmi(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *v1.VirtualMachineInstanceSpec) flavor.Conflicts {
	return m.ApplyToVmiFunc(field, profile, vmiSpec)
}

func NewMockFlavorMethods() *MockFlavorMethods {
	return &MockFlavorMethods{
		FindFlavorFunc: func(_ *v1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
			return nil, nil
		},
		ApplyToVmiFunc: func(_ *k8sfield.Path, _ *flavorv1alpha1.VirtualMachineFlavorProfile, _ *v1.VirtualMachineInstanceSpec) flavor.Conflicts {
			return nil
		},
	}
}
