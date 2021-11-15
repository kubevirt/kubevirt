package flavor

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"

	"k8s.io/client-go/tools/cache"

	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"

	virtv1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
)

type Methods interface {
	FindProfile(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error)
	ApplyToVmi(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts
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

func (m *methods) FindProfile(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error) {
	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	profiles, err := getProfiles(vm.Spec.Flavor.Name, vm.Namespace, vm.Spec.Flavor.Kind, m.flavorStore, m.clusterFlavorStore)
	if err != nil {
		return nil, err
	}

	if vm.Spec.Flavor.Profile == "" {
		profile := findFirstProfile(profiles, func(profile *flavorv1alpha1.VirtualMachineFlavorProfile) bool {
			return profile.Default
		})
		if profile == nil {
			return nil, fmt.Errorf("flavor does not specify a default profile")
		}
		return profile, nil
	} else {
		profile := findFirstProfile(profiles, func(profile *flavorv1alpha1.VirtualMachineFlavorProfile) bool {
			return profile.Name == vm.Spec.Flavor.Profile
		})
		if profile == nil {
			return nil, fmt.Errorf("flavor does not have a profile with name: %v", vm.Spec.Flavor.Profile)
		}
		return profile, nil
	}
}

func (m *methods) ApplyToVmi(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	var conflicts Conflicts

	conflicts = append(conflicts, applyCpu(field, profile, vmiSpec)...)

	return conflicts
}

func findFirstProfile(profiles []flavorv1alpha1.VirtualMachineFlavorProfile, predicate func(profile *flavorv1alpha1.VirtualMachineFlavorProfile) bool) *flavorv1alpha1.VirtualMachineFlavorProfile {
	for i := range profiles {
		profile := &profiles[i]
		if predicate(profile) {
			return profile
		}
	}
	return nil
}

func getKey(namespace string, name string) string {
	return namespace + "/" + name
}

func getProfiles(name string, namespace string, kind string, flavorStore, clusterFlavorStore cache.Store) ([]flavorv1alpha1.VirtualMachineFlavorProfile, error) {
	switch strings.ToLower(kind) {
	case "virtualmachineflavors", "virtualmachineflavor":
		key := getKey(namespace, name)
		obj, exists, err := flavorStore.GetByKey(key)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.NewNotFound(flavorv1alpha1.Resource("virtualmachineflavor"), key)
		}
		flavor := obj.(*flavorv1alpha1.VirtualMachineFlavor)
		return flavor.Profiles, nil

	case "", "virtualmachineclusterflavors", "virtualmachineclusterflavor":
		obj, exists, err := clusterFlavorStore.GetByKey(name)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.NewNotFound(flavorv1alpha1.Resource("virtualmachineclusterflavor"), name)
		}
		flavor := obj.(*flavorv1alpha1.VirtualMachineClusterFlavor)
		return flavor.Profiles, nil
	default:
		return nil, fmt.Errorf("got unexpected kind in FlavorMatcher: %s", kind)
	}
}

func applyCpu(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	if profile.CPU == nil {
		return nil
	}
	if vmiSpec.Domain.CPU != nil {
		return Conflicts{field.Child("domain", "cpu")}
	}

	vmiSpec.Domain.CPU = profile.CPU.DeepCopy()
	return nil
}
