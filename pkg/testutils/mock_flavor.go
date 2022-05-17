package testutils

import (
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/kubevirt/pkg/flavor"
)

type MockFlavorMethods struct {
	FindFlavorSpecFunc     func(flavorMatcher *v1.FlavorMatcher, namespace string) (*flavorv1alpha1.VirtualMachineFlavorSpec, error)
	ApplyToVmiFunc         func(field *k8sfield.Path, flavorspec *flavorv1alpha1.VirtualMachineFlavorSpec, preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) flavor.Conflicts
	FindPreferenceSpecFunc func(preferenceMatcher *v1.PreferenceMatcher, namespace string) (*flavorv1alpha1.VirtualMachinePreferenceSpec, error)
}

var _ flavor.Methods = &MockFlavorMethods{}

func (m *MockFlavorMethods) FindFlavorSpec(flavorMatcher *v1.FlavorMatcher, namespace string) (*flavorv1alpha1.VirtualMachineFlavorSpec, error) {
	return m.FindFlavorSpecFunc(flavorMatcher, namespace)
}

func (m *MockFlavorMethods) ApplyToVmi(field *k8sfield.Path, flavorspec *flavorv1alpha1.VirtualMachineFlavorSpec, preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *v1.VirtualMachineInstanceSpec) flavor.Conflicts {
	return m.ApplyToVmiFunc(field, flavorspec, preferenceSpec, vmiSpec)
}

func (m *MockFlavorMethods) FindPreferenceSpec(preferenceMatcher *v1.PreferenceMatcher, namespace string) (*flavorv1alpha1.VirtualMachinePreferenceSpec, error) {
	return m.FindPreferenceSpecFunc(preferenceMatcher, namespace)
}

func NewMockFlavorMethods() *MockFlavorMethods {
	return &MockFlavorMethods{
		FindFlavorSpecFunc: func(_ *v1.FlavorMatcher, _ string) (*flavorv1alpha1.VirtualMachineFlavorSpec, error) {
			return nil, nil
		},
		ApplyToVmiFunc: func(_ *k8sfield.Path, _ *flavorv1alpha1.VirtualMachineFlavorSpec, _ *flavorv1alpha1.VirtualMachinePreferenceSpec, _ *v1.VirtualMachineInstanceSpec) flavor.Conflicts {
			return nil
		},
		FindPreferenceSpecFunc: func(_ *v1.PreferenceMatcher, _ string) (*flavorv1alpha1.VirtualMachinePreferenceSpec, error) {
			return nil, nil
		},
	}
}
