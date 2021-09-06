package flavor

import (
	"fmt"
	"reflect"
	"strings"

	apiflavor "kubevirt.io/api/flavor"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
)

type Methods interface {
	FindProfile(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorProfile, error)
	ApplyToVmi(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) Conflicts
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

func (m *methods) ApplyToVmi(field *k8sfield.Path, profile *flavorv1alpha1.VirtualMachineFlavorProfile, vm *virtv1.VirtualMachine, vmi *virtv1.VirtualMachineInstance) Conflicts {
	var conflicts Conflicts
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
		vmi.Annotations[virtv1.FlavorAnnotation] = vm.Spec.Flavor.Name
	case "virtualmachineclusterflavors", "virtualmachineclusterflavor":
		vmi.Annotations[virtv1.ClusterFlavorAnnotation] = vm.Spec.Flavor.Name
	}

	conflicts = append(conflicts, patchDomainSpec(field.Child("domain"), profile.DomainTemplate, &vmi.Spec.Domain)...)

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
	case apiflavor.PluralResourceName, apiflavor.SingularResourceName:
		key := getKey(namespace, name)
		obj, exists, err := flavorStore.GetByKey(key)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.NewNotFound(flavorv1alpha1.Resource(apiflavor.SingularResourceName), key)
		}
		flavor := obj.(*flavorv1alpha1.VirtualMachineFlavor)
		return flavor.Profiles, nil

	case "", apiflavor.ClusterPluralResourceName, apiflavor.ClusterSingularResourceName:
		obj, exists, err := clusterFlavorStore.GetByKey(name)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errors.NewNotFound(flavorv1alpha1.Resource(apiflavor.ClusterSingularResourceName), name)
		}
		flavor := obj.(*flavorv1alpha1.VirtualMachineClusterFlavor)
		return flavor.Profiles, nil
	default:
		return nil, fmt.Errorf("got unexpected kind in FlavorMatcher: %s", kind)
	}
}

func patchDomainSpec(field *k8sfield.Path, profileDomain *flavorv1alpha1.VirtualMachineFlavorDomainTemplateSpec, vmiDomain *virtv1.DomainSpec) Conflicts {
	if profileDomain == nil {
		return nil
	}

	// Not using reflection, to make it easier to understand and less likely to contain a bug.
	conflicts := patchResourceRequirements(field.Child("resources"), &profileDomain.Resources, &vmiDomain.Resources)
	conflicts = append(conflicts, patchPtr(field.Child("cpu"), &profileDomain.CPU, &vmiDomain.CPU)...)
	conflicts = append(conflicts, patchMemory(field.Child("memory"), &profileDomain.Memory, &vmiDomain.Memory)...)
	conflicts = append(conflicts, patchPtr(field.Child("machine"), &profileDomain.Machine, &vmiDomain.Machine)...)
	conflicts = append(conflicts, patchPtr(field.Child("firmware"), &profileDomain.Firmware, &vmiDomain.Firmware)...)
	conflicts = append(conflicts, patchPtr(field.Child("clock"), &profileDomain.Clock, &vmiDomain.Clock)...)
	conflicts = append(conflicts, patchPtr(field.Child("features"), &profileDomain.Features, &vmiDomain.Features)...)
	conflicts = append(conflicts, patchPtr(field.Child("ioThreadsPolicy"), &profileDomain.IOThreadsPolicy, &vmiDomain.IOThreadsPolicy)...)
	conflicts = append(conflicts, patchPtr(field.Child("chassis"), &profileDomain.Chassis, &vmiDomain.Chassis)...)

	// The "devices" field is not applied to VMI in full, we do however want to apply some of the bools
	conflicts = append(conflicts, patchPtr(field.Child("devices.UseVirtioTransitional"), &profileDomain.Devices.UseVirtioTransitional, &vmiDomain.Devices.UseVirtioTransitional)...)
	conflicts = append(conflicts, patchPtr(field.Child("devices.DisableHotplug"), &profileDomain.Devices.DisableHotplug, &vmiDomain.Devices.DisableHotplug)...)
	conflicts = append(conflicts, patchPtr(field.Child("devices.AutoattachPodInterface"), &profileDomain.Devices.AutoattachPodInterface, &vmiDomain.Devices.AutoattachPodInterface)...)
	conflicts = append(conflicts, patchPtr(field.Child("devices.AutoattachGraphicsDevice"), &profileDomain.Devices.AutoattachGraphicsDevice, &vmiDomain.Devices.AutoattachGraphicsDevice)...)
	conflicts = append(conflicts, patchPtr(field.Child("devices.AutoattachSerialConsole"), &profileDomain.Devices.AutoattachSerialConsole, &vmiDomain.Devices.AutoattachSerialConsole)...)
	conflicts = append(conflicts, patchPtr(field.Child("devices.AutoattachMemBalloon"), &profileDomain.Devices.AutoattachMemBalloon, &vmiDomain.Devices.AutoattachMemBalloon)...)
	conflicts = append(conflicts, patchPtr(field.Child("devices.BlockMultiQueue"), &profileDomain.Devices.BlockMultiQueue, &vmiDomain.Devices.BlockMultiQueue)...)
	conflicts = append(conflicts, patchPtr(field.Child("devices.NetworkInterfaceMultiQueue"), &profileDomain.Devices.NetworkInterfaceMultiQueue, &vmiDomain.Devices.NetworkInterfaceMultiQueue)...)

	return conflicts
}

func patchPtr(field *k8sfield.Path, profilePtr interface{}, vmiPtr interface{}) Conflicts {
	profileVal := reflect.ValueOf(profilePtr).Elem()
	vmiVal := reflect.ValueOf(vmiPtr).Elem()

	if profileVal.Type() != vmiVal.Type() {
		panic("patchPtr requires the same type")
	}

	if profileVal.IsZero() {
		return nil
	}
	if vmiVal.IsZero() {
		vmiVal.Set(profileVal)
		return nil
	}

	return Conflicts{field}
}

func patchResourceRequirements(field *k8sfield.Path, profileObj *virtv1.ResourceRequirements, vmiObj *virtv1.ResourceRequirements) Conflicts {
	conflicts := patchResourceList(field.Child("requests"), &profileObj.Requests, &vmiObj.Requests)
	conflicts = append(conflicts, patchResourceList(field.Child("limits"), &profileObj.Limits, &vmiObj.Limits)...)
	if profileObj.OvercommitGuestOverhead {
		vmiObj.OvercommitGuestOverhead = true
	}
	return conflicts
}

func patchResourceList(field *k8sfield.Path, profileObj *corev1.ResourceList, vmiObj *corev1.ResourceList) Conflicts {
	if *profileObj == nil {
		return nil
	}
	if *vmiObj == nil {
		*vmiObj = *profileObj
		return nil
	}

	var conflicts Conflicts
	for name, quantity := range *profileObj {
		if _, ok := (*vmiObj)[name]; ok {
			conflicts = append(conflicts, field.Child(string(name)))
			continue
		}
		(*vmiObj)[name] = quantity
	}
	return conflicts
}

func patchMemory(field *k8sfield.Path, profileObj **virtv1.Memory, vmiObj **virtv1.Memory) Conflicts {
	if (*profileObj) == nil {
		return nil
	}
	if (*vmiObj) == nil {
		*vmiObj = *profileObj
		return nil
	}

	profileMem := *profileObj
	vmiMem := *vmiObj
	conflicts := patchPtr(field.Child("hugepages"), &profileMem.Hugepages, &vmiMem.Hugepages)
	conflicts = append(conflicts, patchPtr(field.Child("guest"), &profileMem.Guest, &vmiMem.Guest)...)

	return conflicts
}
