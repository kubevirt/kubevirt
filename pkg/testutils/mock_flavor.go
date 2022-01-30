package testutils

import (
	"strings"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	v1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/kubevirt/pkg/flavor"
)

type MockFlavorMethods struct {
	FindFlavorFunc func(vm *v1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error)
	ApplyToVmiFunc func(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) flavor.Conflicts
}

func (m *MockFlavorMethods) FindProfile(vm *v1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
	return m.FindFlavorFunc(vm)
}

func (m *MockFlavorMethods) ApplyToVmi(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) flavor.Conflicts {
	var flavor string

	if vm.Spec.Flavor != nil {
		flavor = strings.ToLower(vm.Spec.Flavor.Kind)
		if flavor == "" {
			flavor = "virtualmachineclusterflavor"
		}
	}

	if vmi.Annotations == nil {
		vmi.Annotations = make(map[string]string)
	}
	switch flavor {
	case "virtualmachineflavors", "virtualmachineflavor":
		vmi.Annotations[v1.FlavorAnnotation] = vm.Spec.Flavor.Name
	case "virtualmachineclusterflavors", "virtualmachineclusterflavor":
		vmi.Annotations[v1.ClusterFlavorAnnotation] = vm.Spec.Flavor.Name
	}
	return m.ApplyToVmiFunc(field, profile, vm, vmi)
}

func NewMockFlavorMethods() *MockFlavorMethods {
	return &MockFlavorMethods{
		FindFlavorFunc: func(_ *v1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
			return nil, nil
		},
		ApplyToVmiFunc: func(_ *k8sfield.Path, _ *flavorv1alpha1.VirtualMachineFlavorProfile, _ *v1.VirtualMachine, _ *v1.VirtualMachineInstance) flavor.Conflicts {
			return nil
		},
	}
}
