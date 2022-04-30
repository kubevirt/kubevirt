package flavor

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	apiflavor "kubevirt.io/api/flavor"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
)

type Methods interface {
	FindFlavorSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorSpec, error)
	ApplyToVmi(field *k8sfield.Path, flavorspec *flavorv1alpha1.VirtualMachineFlavorSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts
}

type Conflicts []*k8sfield.Path

func (c Conflicts) String() string {
	pathStrings := make([]string, 0, len(c))
	for _, path := range c {
		pathStrings = append(pathStrings, path.String())
	}
	return strings.Join(pathStrings, ", ")
}

type methods struct {
	flavorStore        cache.Store
	clusterFlavorStore cache.Store
}

var _ Methods = &methods{}

func NewMethods(flavorStore, clusterFlavorStore cache.Store) Methods {
	return &methods{
		flavorStore:        flavorStore,
		clusterFlavorStore: clusterFlavorStore,
	}
}

func (m *methods) ApplyToVmi(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	var conflicts Conflicts

	conflicts = append(conflicts, applyCpu(field, flavorSpec, vmiSpec)...)

	return conflicts

}

func (m *methods) FindFlavorSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorSpec, error) {

	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	switch strings.ToLower(vm.Spec.Flavor.Kind) {
	case apiflavor.SingularResourceName, apiflavor.PluralResourceName:
		flavor, err := m.findFlavor(vm)
		if err != nil {
			return nil, err
		}
		return &flavor.Spec, nil
	case apiflavor.ClusterSingularResourceName, apiflavor.ClusterPluralResourceName, "":
		clusterFlavor, err := m.findClusterFlavor(vm)
		if err != nil {
			return nil, err
		}
		return &clusterFlavor.Spec, nil
	default:
		return nil, fmt.Errorf("got unexpected kind in FlavorMatcher: %s", vm.Spec.Flavor.Kind)
	}
}

func (m *methods) findFlavor(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavor, error) {

	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	key := vm.Namespace + "/" + vm.Spec.Flavor.Name
	obj, exists, err := m.flavorStore.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(flavorv1alpha1.Resource(apiflavor.SingularResourceName), key)
	}
	flavor := obj.(*flavorv1alpha1.VirtualMachineFlavor)
	return flavor, nil
}

func (m *methods) findClusterFlavor(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineClusterFlavor, error) {

	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	key := vm.Spec.Flavor.Name
	obj, exists, err := m.clusterFlavorStore.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(flavorv1alpha1.Resource(apiflavor.ClusterSingularResourceName), key)
	}
	flavor := obj.(*flavorv1alpha1.VirtualMachineClusterFlavor)
	return flavor, nil
}

func applyCpu(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if flavorSpec.CPU == nil {
		return nil
	}
	if vmiSpec.Domain.CPU != nil {
		return Conflicts{field.Child("domain", "cpu")}
	}

	vmiSpec.Domain.CPU = flavorSpec.CPU.DeepCopy()

	return nil
}

func AddFlavorNameAnnotations(vm *virtv1.VirtualMachine, target metav1.Object) {
	if vm.Spec.Flavor == nil {
		return
	}

	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}
	switch strings.ToLower(vm.Spec.Flavor.Kind) {
	case apiflavor.PluralResourceName, apiflavor.SingularResourceName:
		target.GetAnnotations()[virtv1.FlavorAnnotation] = vm.Spec.Flavor.Name
	case "", apiflavor.ClusterPluralResourceName, apiflavor.ClusterSingularResourceName:
		target.GetAnnotations()[virtv1.ClusterFlavorAnnotation] = vm.Spec.Flavor.Name
	}
}
