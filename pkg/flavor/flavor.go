package flavor

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	apiflavor "kubevirt.io/api/flavor"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
)

type Methods interface {
	FindFlavorSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorSpec, error)
	ApplyToVmi(field *k8sfield.Path, flavorspec *flavorv1alpha1.VirtualMachineFlavorSpec, prefernceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts
	FindPreferenceSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachinePreferenceSpec, error)
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
	flavorStore            cache.Store
	clusterFlavorStore     cache.Store
	preferenceStore        cache.Store
	clusterPreferenceStore cache.Store
}

var _ Methods = &methods{}

func NewMethods(flavorStore, clusterFlavorStore, preferenceStore, clusterPreferenceStore cache.Store) Methods {
	return &methods{
		flavorStore:            flavorStore,
		clusterFlavorStore:     clusterFlavorStore,
		preferenceStore:        preferenceStore,
		clusterPreferenceStore: clusterPreferenceStore,
	}
}

func (m *methods) ApplyToVmi(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	var conflicts Conflicts

	if flavorSpec != nil {
		conflicts = append(conflicts, applyCpu(field, flavorSpec, preferenceSpec, vmiSpec)...)
		conflicts = append(conflicts, applyMemory(field, flavorSpec, vmiSpec)...)
	}

	if preferenceSpec != nil {
		// By design Preferences can't conflict with the VMI so we don't return any
		applyDevicePreferences(preferenceSpec, vmiSpec)
	}

	return conflicts
}

func (m *methods) FindPreferenceSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachinePreferenceSpec, error) {

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiflavor.SingularPreferenceResourceName, apiflavor.PluralPreferenceResourceName:
		preference, err := m.findPreference(vm)
		if err != nil {
			return nil, err
		}
		return &preference.Spec, nil
	case apiflavor.ClusterSingularPreferenceResourceName, apiflavor.ClusterPluralPreferenceResourceName, "":
		clusterPreference, err := m.findClusterPreference(vm)
		if err != nil {
			return nil, err
		}
		return &clusterPreference.Spec, nil
	default:
		return nil, fmt.Errorf("got unexpected kind in PreferenceMatcher: %s", vm.Spec.Preference.Kind)
	}
}

func (m *methods) findPreference(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachinePreference, error) {

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	key := vm.Namespace + "/" + vm.Spec.Preference.Name
	obj, exists, err := m.preferenceStore.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(flavorv1alpha1.Resource(apiflavor.SingularPreferenceResourceName), key)
	}
	preference := obj.(*flavorv1alpha1.VirtualMachinePreference)
	return preference, nil
}

func (m *methods) findClusterPreference(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineClusterPreference, error) {

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	key := vm.Spec.Preference.Name
	obj, exists, err := m.clusterPreferenceStore.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(flavorv1alpha1.Resource(apiflavor.ClusterSingularPreferenceResourceName), key)
	}
	preference := obj.(*flavorv1alpha1.VirtualMachineClusterPreference)
	return preference, nil
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

func applyCpu(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	if vmiSpec.Domain.CPU != nil {
		return Conflicts{field.Child("domain", "cpu")}
	}

	vmiSpec.Domain.CPU = &virtv1.CPU{
		Sockets:               uint32(1),
		Cores:                 uint32(1),
		Threads:               uint32(1),
		Model:                 flavorSpec.CPU.Model,
		DedicatedCPUPlacement: flavorSpec.CPU.DedicatedCPUPlacement,
		IsolateEmulatorThread: flavorSpec.CPU.IsolateEmulatorThread,
		NUMA:                  flavorSpec.CPU.NUMA.DeepCopy(),
		Realtime:              flavorSpec.CPU.Realtime.DeepCopy(),
	}

	// Default to PreferCores when a PreferredCPUTopology isn't provided
	preferredTopology := flavorv1alpha1.PreferCores
	if preferenceSpec != nil && preferenceSpec.CPU != nil && preferenceSpec.CPU.PreferredCPUTopology != "" {
		preferredTopology = preferenceSpec.CPU.PreferredCPUTopology
	}

	switch preferredTopology {
	case flavorv1alpha1.PreferCores:
		vmiSpec.Domain.CPU.Cores = flavorSpec.CPU.Guest
	case flavorv1alpha1.PreferSockets:
		vmiSpec.Domain.CPU.Sockets = flavorSpec.CPU.Guest
	case flavorv1alpha1.PreferThreads:
		vmiSpec.Domain.CPU.Threads = flavorSpec.CPU.Guest
	}

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

func applyMemory(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if flavorSpec.Memory.Guest == nil {
		return nil
	}

	if vmiSpec.Domain.Memory != nil {
		return Conflicts{field.Child("domain", "memory")}
	}

	flavorMemoryGuest := flavorSpec.Memory.Guest.DeepCopy()
	vmiSpec.Domain.Memory = &virtv1.Memory{
		Guest: &flavorMemoryGuest,
	}

	if flavorSpec.Memory.Hugepages != nil {
		flavorHugePages := flavorSpec.Memory.Hugepages.DeepCopy()
		vmiSpec.Domain.Memory.Hugepages = flavorHugePages
	}

	return nil
}

func applyDevicePreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

	if preferenceSpec.Devices == nil {
		return
	}

	// We only want to apply a preference bool when...
	//
	// 1. A preference has actually been provided
	// 2. The preference is true
	// 3. The user hasn't defined the corresponding attribute already within the VMI
	//
	if preferenceSpec.Devices.PreferredAutoattachGraphicsDevice != nil && *preferenceSpec.Devices.PreferredAutoattachGraphicsDevice && vmiSpec.Domain.Devices.AutoattachGraphicsDevice == nil {
		vmiSpec.Domain.Devices.AutoattachGraphicsDevice = pointer.BoolPtr(true)
	}

	if preferenceSpec.Devices.PreferredAutoattachMemBalloon != nil && *preferenceSpec.Devices.PreferredAutoattachMemBalloon && vmiSpec.Domain.Devices.AutoattachMemBalloon == nil {
		vmiSpec.Domain.Devices.AutoattachMemBalloon = pointer.BoolPtr(true)
	}

	if preferenceSpec.Devices.PreferredAutoattachPodInterface != nil && *preferenceSpec.Devices.PreferredAutoattachPodInterface && vmiSpec.Domain.Devices.AutoattachPodInterface == nil {
		vmiSpec.Domain.Devices.AutoattachPodInterface = pointer.BoolPtr(true)
	}

	if preferenceSpec.Devices.PreferredAutoattachSerialConsole != nil && *preferenceSpec.Devices.PreferredAutoattachSerialConsole && vmiSpec.Domain.Devices.AutoattachSerialConsole == nil {
		vmiSpec.Domain.Devices.AutoattachSerialConsole = pointer.BoolPtr(true)
	}

	if preferenceSpec.Devices.PreferredUseVirtioTransitional != nil && *preferenceSpec.Devices.PreferredUseVirtioTransitional && vmiSpec.Domain.Devices.UseVirtioTransitional == nil {
		vmiSpec.Domain.Devices.UseVirtioTransitional = pointer.BoolPtr(true)
	}

	// FIXME DisableHotplug isn't a pointer bool so we don't have a way to tell if a user has actually set it, for now override.
	if preferenceSpec.Devices.PreferredDisableHotplug != nil && *preferenceSpec.Devices.PreferredDisableHotplug {
		vmiSpec.Domain.Devices.DisableHotplug = true
	}

	if preferenceSpec.Devices.PreferredSoundModel != "" && vmiSpec.Domain.Devices.Sound != nil && vmiSpec.Domain.Devices.Sound.Model == "" {
		vmiSpec.Domain.Devices.Sound.Model = preferenceSpec.Devices.PreferredSoundModel
	}

	if preferenceSpec.Devices.PreferredRng != nil && vmiSpec.Domain.Devices.Rng == nil {
		vmiSpec.Domain.Devices.Rng = preferenceSpec.Devices.PreferredRng.DeepCopy()
	}

	if preferenceSpec.Devices.PreferredBlockMultiQueue != nil && *preferenceSpec.Devices.PreferredBlockMultiQueue && vmiSpec.Domain.Devices.BlockMultiQueue == nil {
		vmiSpec.Domain.Devices.BlockMultiQueue = pointer.BoolPtr(true)
	}

	if preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue != nil && *preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue && vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue == nil {
		vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue = pointer.BoolPtr(true)
	}

	applyDiskPreferences(preferenceSpec, vmiSpec)
	applyInterfacePreferences(preferenceSpec, vmiSpec)
	applyInputPreferences(preferenceSpec, vmiSpec)

}

func applyDiskPreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for diskIndex := range vmiSpec.Domain.Devices.Disks {
		vmiDisk := &vmiSpec.Domain.Devices.Disks[diskIndex]
		if vmiDisk.DiskDevice.Disk != nil {
			if preferenceSpec.Devices.PreferredDiskBus != "" && vmiDisk.DiskDevice.Disk.Bus == "" {
				vmiDisk.DiskDevice.Disk.Bus = preferenceSpec.Devices.PreferredDiskBus
			}

			if preferenceSpec.Devices.PreferredDiskBlockSize != nil && vmiDisk.BlockSize == nil {
				vmiDisk.BlockSize = preferenceSpec.Devices.PreferredDiskBlockSize.DeepCopy()
			}

			if preferenceSpec.Devices.PreferredDiskCache != "" && vmiDisk.Cache == "" {
				vmiDisk.Cache = preferenceSpec.Devices.PreferredDiskCache
			}

			if preferenceSpec.Devices.PreferredDiskIO != "" && vmiDisk.IO == "" {
				vmiDisk.IO = preferenceSpec.Devices.PreferredDiskIO
			}

			if preferenceSpec.Devices.PreferredDiskDedicatedIoThread != nil && *preferenceSpec.Devices.PreferredDiskDedicatedIoThread && vmiDisk.DedicatedIOThread == nil {
				vmiDisk.DedicatedIOThread = pointer.BoolPtr(true)
			}

		} else if vmiDisk.DiskDevice.CDRom != nil {
			if preferenceSpec.Devices.PreferredCdromBus != "" && vmiDisk.DiskDevice.CDRom.Bus == "" {
				vmiDisk.DiskDevice.CDRom.Bus = preferenceSpec.Devices.PreferredCdromBus
			}

		} else if vmiDisk.DiskDevice.LUN != nil {
			if preferenceSpec.Devices.PreferredLunBus != "" && vmiDisk.DiskDevice.LUN.Bus == "" {
				vmiDisk.DiskDevice.LUN.Bus = preferenceSpec.Devices.PreferredLunBus
			}
		}
	}
}

func applyInterfacePreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for ifaceIndex := range vmiSpec.Domain.Devices.Interfaces {
		vmiIface := &vmiSpec.Domain.Devices.Interfaces[ifaceIndex]
		if preferenceSpec.Devices.PreferredInterfaceModel != "" && vmiIface.Model == "" {
			vmiIface.Model = preferenceSpec.Devices.PreferredInterfaceModel
		}
	}
}

func applyInputPreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for inputIndex := range vmiSpec.Domain.Devices.Inputs {
		vmiInput := &vmiSpec.Domain.Devices.Inputs[inputIndex]
		if preferenceSpec.Devices.PreferredInputBus != "" && vmiInput.Bus == "" {
			vmiInput.Bus = preferenceSpec.Devices.PreferredInputBus
		}

		if preferenceSpec.Devices.PreferredInputType != "" && vmiInput.Type == "" {
			vmiInput.Type = preferenceSpec.Devices.PreferredInputType
		}
	}
}
