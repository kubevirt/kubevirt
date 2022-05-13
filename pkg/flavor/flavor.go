package flavor

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/api/core/v1"
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
		conflicts = append(conflicts, applyIOThreadPolicy(field, flavorSpec, vmiSpec)...)
		conflicts = append(conflicts, applyLaunchSecurity(field, flavorSpec, vmiSpec)...)
		conflicts = append(conflicts, applyGPUs(field, flavorSpec, vmiSpec)...)
		conflicts = append(conflicts, applyHostDevices(field, flavorSpec, vmiSpec)...)
	}

	if preferenceSpec != nil {
		// By design Preferences can't conflict with the VMI so we don't return any
		applyDevicePreferences(preferenceSpec, vmiSpec)
		applyFeaturePreferences(preferenceSpec, vmiSpec)
		applyFirmwarePreferences(preferenceSpec, vmiSpec)
		applyMachinePreferences(preferenceSpec, vmiSpec)
		applyClockPreferences(preferenceSpec, vmiSpec)
	}

	return conflicts
}

func (m *methods) FindPreferenceSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachinePreferenceSpec, error) {

	var err error
	var preference *flavorv1alpha1.VirtualMachinePreference
	var clusterPreference *flavorv1alpha1.VirtualMachineClusterPreference

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiflavor.SingularPreferenceResourceName, apiflavor.PluralPreferenceResourceName:
		preference, err = m.findPreference(vm)
	case apiflavor.ClusterSingularPreferenceResourceName, apiflavor.ClusterPluralPreferenceResourceName:
		clusterPreference, err = m.findClusterPreference(vm)
	default:
		err = fmt.Errorf("got unexpected kind in PreferenceMatcher: %s", vm.Spec.Preference.Kind)
	}

	if err != nil {
		return nil, err
	}

	if preference != nil {
		return &preference.Spec, nil
	}

	if clusterPreference != nil {
		return &clusterPreference.Spec, nil
	}

	return nil, nil
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

	var err error
	var flavor *flavorv1alpha1.VirtualMachineFlavor
	var clusterFlavor *flavorv1alpha1.VirtualMachineClusterFlavor

	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	switch strings.ToLower(vm.Spec.Flavor.Kind) {
	case apiflavor.SingularResourceName, apiflavor.PluralResourceName:
		flavor, err = m.findFlavor(vm)
	case apiflavor.ClusterSingularResourceName, apiflavor.ClusterPluralResourceName, "":
		clusterFlavor, err = m.findClusterFlavor(vm)
	default:
		err = fmt.Errorf("got unexpected kind in FlavorMatcher: %s", vm.Spec.Flavor.Kind)
	}

	if err != nil {
		return nil, err
	}

	if flavor != nil {
		return &flavor.Spec, nil
	}

	if clusterFlavor != nil {
		return &clusterFlavor.Spec, nil
	}

	return nil, nil
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

func AddPreferenceNameAnnotations(vm *virtv1.VirtualMachine, target metav1.Object) {
	if vm.Spec.Preference == nil {
		return
	}

	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}
	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiflavor.PluralPreferenceResourceName, apiflavor.SingularPreferenceResourceName:
		target.GetAnnotations()[virtv1.PreferenceAnnotation] = vm.Spec.Preference.Name
	case "", apiflavor.ClusterPluralPreferenceResourceName, apiflavor.ClusterSingularPreferenceResourceName:
		target.GetAnnotations()[virtv1.ClusterPreferenceAnnotation] = vm.Spec.Preference.Name
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
		vmiSpec.Domain.Memory.Hugepages = flavorSpec.Memory.Hugepages.DeepCopy()
	}

	return nil
}

func applyIOThreadPolicy(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if flavorSpec.IOThreadsPolicy == nil {
		return nil
	}

	if vmiSpec.Domain.IOThreadsPolicy != nil {
		return Conflicts{field.Child("domain", "ioThreadsPolicy")}
	}

	flavorIOThreadPolicy := *flavorSpec.IOThreadsPolicy
	vmiSpec.Domain.IOThreadsPolicy = &flavorIOThreadPolicy

	return nil
}

func applyLaunchSecurity(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if flavorSpec.LaunchSecurity == nil {
		return nil
	}

	if vmiSpec.Domain.LaunchSecurity != nil {
		return Conflicts{field.Child("domain", "launchSecurity")}
	}

	vmiSpec.Domain.LaunchSecurity = flavorSpec.LaunchSecurity.DeepCopy()

	return nil
}

func applyGPUs(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if len(flavorSpec.GPUs) == 0 {
		return nil
	}

	if len(vmiSpec.Domain.Devices.GPUs) >= 1 {
		return Conflicts{field.Child("domain", "devices", "gpus")}
	}

	vmiSpec.Domain.Devices.GPUs = make([]v1.GPU, len(flavorSpec.GPUs))
	copy(vmiSpec.Domain.Devices.GPUs, flavorSpec.GPUs)

	return nil
}

func applyHostDevices(field *k8sfield.Path, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if len(flavorSpec.HostDevices) == 0 {
		return nil
	}

	if len(vmiSpec.Domain.Devices.HostDevices) >= 1 {
		return Conflicts{field.Child("domain", "devices", "hostDevices")}
	}

	vmiSpec.Domain.Devices.HostDevices = make([]v1.HostDevice, len(flavorSpec.HostDevices))
	copy(vmiSpec.Domain.Devices.HostDevices, flavorSpec.HostDevices)

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
		vmiSpec.Domain.Devices.AutoattachGraphicsDevice = pointer.Bool(true)
	}

	if preferenceSpec.Devices.PreferredAutoattachMemBalloon != nil && *preferenceSpec.Devices.PreferredAutoattachMemBalloon && vmiSpec.Domain.Devices.AutoattachMemBalloon == nil {
		vmiSpec.Domain.Devices.AutoattachMemBalloon = pointer.Bool(true)
	}

	if preferenceSpec.Devices.PreferredAutoattachPodInterface != nil && *preferenceSpec.Devices.PreferredAutoattachPodInterface && vmiSpec.Domain.Devices.AutoattachPodInterface == nil {
		vmiSpec.Domain.Devices.AutoattachPodInterface = pointer.Bool(true)
	}

	if preferenceSpec.Devices.PreferredAutoattachSerialConsole != nil && *preferenceSpec.Devices.PreferredAutoattachSerialConsole && vmiSpec.Domain.Devices.AutoattachSerialConsole == nil {
		vmiSpec.Domain.Devices.AutoattachSerialConsole = pointer.Bool(true)
	}

	if preferenceSpec.Devices.PreferredUseVirtioTransitional != nil && *preferenceSpec.Devices.PreferredUseVirtioTransitional && vmiSpec.Domain.Devices.UseVirtioTransitional == nil {
		vmiSpec.Domain.Devices.UseVirtioTransitional = pointer.Bool(true)
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
		vmiSpec.Domain.Devices.BlockMultiQueue = pointer.Bool(true)
	}

	if preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue != nil && *preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue && vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue == nil {
		vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue = pointer.Bool(true)
	}

	if preferenceSpec.Devices.PreferredTPM != nil && vmiSpec.Domain.Devices.TPM == nil {
		vmiSpec.Domain.Devices.TPM = preferenceSpec.Devices.PreferredTPM.DeepCopy()
	}

	applyDiskPreferences(preferenceSpec, vmiSpec)
	applyInterfacePreferences(preferenceSpec, vmiSpec)
	applyInputPreferences(preferenceSpec, vmiSpec)

}

func applyDiskPreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for diskIndex := range vmiSpec.Domain.Devices.Disks {
		vmiDisk := &vmiSpec.Domain.Devices.Disks[diskIndex]
		// If we don't have a target device defined default to a DiskTarget so we can apply preferences
		if vmiDisk.DiskDevice.Disk == nil && vmiDisk.DiskDevice.CDRom == nil && vmiDisk.DiskDevice.LUN == nil {
			vmiDisk.DiskDevice.Disk = &virtv1.DiskTarget{}
		}

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
				vmiDisk.DedicatedIOThread = pointer.Bool(true)
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

func applyFeaturePreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

	if preferenceSpec.Features == nil {
		return
	}

	if vmiSpec.Domain.Features == nil {
		vmiSpec.Domain.Features = &v1.Features{}
	}

	// FIXME vmiSpec.Domain.Features.ACPI isn't a FeatureState pointer so just overwrite if we have a preference for now.
	if preferenceSpec.Features.PreferredAcpi != nil {
		vmiSpec.Domain.Features.ACPI = *preferenceSpec.Features.PreferredAcpi.DeepCopy()
	}

	if preferenceSpec.Features.PreferredApic != nil && vmiSpec.Domain.Features.APIC == nil {
		vmiSpec.Domain.Features.APIC = preferenceSpec.Features.PreferredApic.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv != nil {
		applyHyperVFeaturePreferences(preferenceSpec, vmiSpec)
	}

	if preferenceSpec.Features.PreferredKvm != nil && vmiSpec.Domain.Features.KVM == nil {
		vmiSpec.Domain.Features.KVM = preferenceSpec.Features.PreferredKvm.DeepCopy()
	}

	if preferenceSpec.Features.PreferredPvspinlock != nil && vmiSpec.Domain.Features.Pvspinlock == nil {
		vmiSpec.Domain.Features.Pvspinlock = preferenceSpec.Features.PreferredPvspinlock.DeepCopy()
	}

	if preferenceSpec.Features.PreferredSmm != nil && vmiSpec.Domain.Features.SMM == nil {
		vmiSpec.Domain.Features.SMM = preferenceSpec.Features.PreferredSmm.DeepCopy()
	}

}

func applyHyperVFeaturePreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

	if vmiSpec.Domain.Features.Hyperv == nil {
		vmiSpec.Domain.Features.Hyperv = &v1.FeatureHyperv{}
	}

	// TODO clean this up with reflection?
	if preferenceSpec.Features.PreferredHyperv.EVMCS != nil && vmiSpec.Domain.Features.Hyperv.EVMCS == nil {
		vmiSpec.Domain.Features.Hyperv.EVMCS = preferenceSpec.Features.PreferredHyperv.EVMCS.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.Frequencies != nil && vmiSpec.Domain.Features.Hyperv.Frequencies == nil {
		vmiSpec.Domain.Features.Hyperv.Frequencies = preferenceSpec.Features.PreferredHyperv.Frequencies.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.IPI != nil && vmiSpec.Domain.Features.Hyperv.IPI == nil {
		vmiSpec.Domain.Features.Hyperv.IPI = preferenceSpec.Features.PreferredHyperv.IPI.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.Reenlightenment != nil && vmiSpec.Domain.Features.Hyperv.Reenlightenment == nil {
		vmiSpec.Domain.Features.Hyperv.Reenlightenment = preferenceSpec.Features.PreferredHyperv.Reenlightenment.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.Relaxed != nil && vmiSpec.Domain.Features.Hyperv.Relaxed == nil {
		vmiSpec.Domain.Features.Hyperv.Relaxed = preferenceSpec.Features.PreferredHyperv.Relaxed.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.Reset != nil && vmiSpec.Domain.Features.Hyperv.Reset == nil {
		vmiSpec.Domain.Features.Hyperv.Reset = preferenceSpec.Features.PreferredHyperv.Reset.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.Runtime != nil && vmiSpec.Domain.Features.Hyperv.Runtime == nil {
		vmiSpec.Domain.Features.Hyperv.Runtime = preferenceSpec.Features.PreferredHyperv.Runtime.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.Spinlocks != nil && vmiSpec.Domain.Features.Hyperv.Spinlocks == nil {
		vmiSpec.Domain.Features.Hyperv.Spinlocks = preferenceSpec.Features.PreferredHyperv.Spinlocks.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.SyNIC != nil && vmiSpec.Domain.Features.Hyperv.SyNIC == nil {
		vmiSpec.Domain.Features.Hyperv.SyNIC = preferenceSpec.Features.PreferredHyperv.SyNIC.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.SyNICTimer != nil && vmiSpec.Domain.Features.Hyperv.SyNICTimer == nil {
		vmiSpec.Domain.Features.Hyperv.SyNICTimer = preferenceSpec.Features.PreferredHyperv.SyNICTimer.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.TLBFlush != nil && vmiSpec.Domain.Features.Hyperv.TLBFlush == nil {
		vmiSpec.Domain.Features.Hyperv.TLBFlush = preferenceSpec.Features.PreferredHyperv.TLBFlush.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.VAPIC != nil && vmiSpec.Domain.Features.Hyperv.VAPIC == nil {
		vmiSpec.Domain.Features.Hyperv.VAPIC = preferenceSpec.Features.PreferredHyperv.VAPIC.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.VPIndex != nil && vmiSpec.Domain.Features.Hyperv.VPIndex == nil {
		vmiSpec.Domain.Features.Hyperv.VPIndex = preferenceSpec.Features.PreferredHyperv.VPIndex.DeepCopy()
	}

	if preferenceSpec.Features.PreferredHyperv.VendorID != nil && vmiSpec.Domain.Features.Hyperv.VendorID == nil {
		vmiSpec.Domain.Features.Hyperv.VendorID = preferenceSpec.Features.PreferredHyperv.VendorID.DeepCopy()
	}
}

func applyFirmwarePreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

	if preferenceSpec.Firmware == nil {
		return
	}

	if vmiSpec.Domain.Firmware == nil {
		vmiSpec.Domain.Firmware = &v1.Firmware{}
	}

	if vmiSpec.Domain.Firmware.Bootloader == nil {
		vmiSpec.Domain.Firmware.Bootloader = &v1.Bootloader{}
	}

	if preferenceSpec.Firmware.PreferredUseBios != nil && *preferenceSpec.Firmware.PreferredUseBios && vmiSpec.Domain.Firmware.Bootloader.BIOS == nil && vmiSpec.Domain.Firmware.Bootloader.EFI == nil {
		vmiSpec.Domain.Firmware.Bootloader.BIOS = &v1.BIOS{}
	}

	if preferenceSpec.Firmware.PreferredUseBiosSerial != nil && *preferenceSpec.Firmware.PreferredUseBiosSerial && vmiSpec.Domain.Firmware.Bootloader.BIOS != nil {
		vmiSpec.Domain.Firmware.Bootloader.BIOS.UseSerial = pointer.Bool(true)
	}

	if preferenceSpec.Firmware.PreferredUseEfi != nil && *preferenceSpec.Firmware.PreferredUseEfi && vmiSpec.Domain.Firmware.Bootloader.EFI == nil && vmiSpec.Domain.Firmware.Bootloader.BIOS == nil {
		vmiSpec.Domain.Firmware.Bootloader.EFI = &v1.EFI{}
	}

	if preferenceSpec.Firmware.PreferredUseSecureBoot != nil && *preferenceSpec.Firmware.PreferredUseSecureBoot && vmiSpec.Domain.Firmware.Bootloader.EFI != nil {
		vmiSpec.Domain.Firmware.Bootloader.EFI.SecureBoot = pointer.Bool(true)
	}
}

func applyMachinePreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

	if preferenceSpec.Machine == nil {
		return
	}

	if vmiSpec.Domain.Machine == nil {
		vmiSpec.Domain.Machine = &v1.Machine{}
	}

	if preferenceSpec.Machine.PreferredMachineType != "" && vmiSpec.Domain.Machine.Type == "" {
		vmiSpec.Domain.Machine.Type = preferenceSpec.Machine.PreferredMachineType
	}
}

func applyClockPreferences(preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	if preferenceSpec.Clock == nil {
		return
	}

	if vmiSpec.Domain.Clock == nil {
		vmiSpec.Domain.Clock = &v1.Clock{}
	}

	// We don't want to allow a partial overwrite here as that could lead to some unexpected behaviour for users so only replace when nothing is set
	if preferenceSpec.Clock.PreferredClockOffset != nil && vmiSpec.Domain.Clock.ClockOffset.UTC == nil && vmiSpec.Domain.Clock.ClockOffset.Timezone == nil {
		vmiSpec.Domain.Clock.ClockOffset = *preferenceSpec.Clock.PreferredClockOffset.DeepCopy()
	}

	if preferenceSpec.Clock.PreferredTimer != nil && vmiSpec.Domain.Clock.Timer == nil {
		vmiSpec.Domain.Clock.Timer = preferenceSpec.Clock.PreferredTimer.DeepCopy()
	}
}
