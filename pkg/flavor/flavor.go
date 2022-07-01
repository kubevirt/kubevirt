package flavor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	apiflavor "kubevirt.io/api/flavor"
	flavorv1alpha1 "kubevirt.io/api/flavor/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
)

type Methods interface {
	FindFlavorSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorSpec, error)
	ApplyToVmi(field *k8sfield.Path, flavorspec *flavorv1alpha1.VirtualMachineFlavorSpec, prefernceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts
	FindPreferenceSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachinePreferenceSpec, error)
	StoreControllerRevisions(vm *virtv1.VirtualMachine) error
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
	clientset kubecli.KubevirtClient
}

var _ Methods = &methods{}

func NewMethods(clientset kubecli.KubevirtClient) Methods {
	return &methods{
		clientset: clientset,
	}
}

func GetRevisionName(vmName, resourceName string, resourceUID types.UID, resourceGeneration int64) string {
	return fmt.Sprintf("%s-%s-%s-%d", vmName, resourceName, resourceUID, resourceGeneration)
}

func CreateFlavorControllerRevision(vm *virtv1.VirtualMachine, revisionName string, flavorApiVersion string, flavorSpec *flavorv1alpha1.VirtualMachineFlavorSpec) (*appsv1.ControllerRevision, error) {

	flavorSpecPatch, err := json.Marshal(*flavorSpec)
	if err != nil {
		return nil, err
	}

	specRevision := flavorv1alpha1.VirtualMachineFlavorSpecRevision{
		APIVersion: flavorApiVersion,
		Spec:       flavorSpecPatch,
	}

	revisionPatch, err := json.Marshal(specRevision)
	if err != nil {
		return nil, err
	}

	return &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:            revisionName,
			Namespace:       vm.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
		},
		Data: runtime.RawExtension{Raw: revisionPatch},
	}, nil

}

func (m *methods) createFlavorRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {

	switch strings.ToLower(vm.Spec.Flavor.Kind) {
	case apiflavor.SingularResourceName, apiflavor.PluralResourceName:
		flavor, err := m.findFlavor(vm)
		if err != nil {
			return nil, err
		}
		revision, err := CreateFlavorControllerRevision(vm, GetRevisionName(vm.Name, flavor.Name, flavor.UID, flavor.Generation), flavor.APIVersion, &flavor.Spec)
		if err != nil {
			return nil, err
		}
		return revision, nil

	case apiflavor.ClusterSingularResourceName, apiflavor.ClusterPluralResourceName:
		clusterFlavor, err := m.findClusterFlavor(vm)
		if err != nil {
			return nil, err
		}

		revision, err := CreateFlavorControllerRevision(vm, GetRevisionName(vm.Name, clusterFlavor.Name, clusterFlavor.UID, clusterFlavor.Generation), clusterFlavor.APIVersion, &clusterFlavor.Spec)
		if err != nil {
			return nil, err
		}
		return revision, nil
	default:
		return nil, fmt.Errorf("got unexpected kind in FlavorMatcher: %s", vm.Spec.Flavor.Kind)
	}
}

func (m *methods) storeFlavorRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {

	if vm.Spec.Flavor == nil || len(vm.Spec.Flavor.RevisionName) > 0 {
		return nil, nil
	}

	flavorRevision, err := m.createFlavorRevision(vm)
	if err != nil {
		return nil, err
	}

	_, err = m.clientset.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), flavorRevision, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Grab the existing revision to check the data it contains
			existingRevision, err := m.clientset.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), flavorRevision.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			// If the data between the two differs return an error, otherwise continue and store the name below.
			if bytes.Compare(existingRevision.Data.Raw, flavorRevision.Data.Raw) != 0 {
				return nil, fmt.Errorf("found existing ControllerRevision with unexpected data: %s", flavorRevision.Name)
			}
		} else {
			return nil, err
		}
	}

	vm.Spec.Flavor.RevisionName = flavorRevision.Name

	return flavorRevision, nil

}

func CreatePreferenceControllerRevision(vm *virtv1.VirtualMachine, revisionName string, preferenceApiVersion string, preferenceSpec *flavorv1alpha1.VirtualMachinePreferenceSpec) (*appsv1.ControllerRevision, error) {

	preferenceSpecPatch, err := json.Marshal(*preferenceSpec)
	if err != nil {
		return nil, err
	}

	specRevision := flavorv1alpha1.VirtualMachinePreferenceSpecRevision{
		APIVersion: preferenceApiVersion,
		Spec:       preferenceSpecPatch,
	}

	revisionPatch, err := json.Marshal(specRevision)
	if err != nil {
		return nil, err
	}

	return &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:            revisionName,
			Namespace:       vm.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
		},
		Data: runtime.RawExtension{Raw: revisionPatch},
	}, nil

}

func (m *methods) createPreferenceRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiflavor.SingularPreferenceResourceName, apiflavor.PluralPreferenceResourceName:
		preference, err := m.findPreference(vm)
		if err != nil {
			return nil, err
		}

		revision, err := CreatePreferenceControllerRevision(vm, GetRevisionName(vm.Name, preference.Name, preference.UID, preference.Generation), preference.APIVersion, &preference.Spec)
		if err != nil {
			return nil, err
		}
		return revision, nil
	case apiflavor.ClusterSingularPreferenceResourceName, apiflavor.ClusterPluralPreferenceResourceName:
		clusterPreference, err := m.findClusterPreference(vm)
		if err != nil {
			return nil, err
		}

		revision, err := CreatePreferenceControllerRevision(vm, GetRevisionName(vm.Name, clusterPreference.Name, clusterPreference.UID, clusterPreference.Generation), clusterPreference.APIVersion, &clusterPreference.Spec)
		if err != nil {
			return nil, err
		}
		return revision, nil
	default:
		return nil, fmt.Errorf("got unexpected kind in PreferenceMatcher: %s", vm.Spec.Preference.Kind)
	}
}

func (m *methods) storePreferenceRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {

	if vm.Spec.Preference == nil || len(vm.Spec.Preference.RevisionName) > 0 {
		return nil, nil
	}

	preferenceRevision, err := m.createPreferenceRevision(vm)
	if err != nil {
		return nil, err
	}

	_, err = m.clientset.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), preferenceRevision, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Grab the existing revision to check the data it contains
			existingRevision, err := m.clientset.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), preferenceRevision.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			// If the data between the two differs return an error, otherwise continue and store the name below.
			if bytes.Compare(existingRevision.Data.Raw, preferenceRevision.Data.Raw) != 0 {
				return nil, fmt.Errorf("found existing ControllerRevision with unexpected data: %s", preferenceRevision.Name)
			}
		} else {
			return nil, err
		}
	}

	vm.Spec.Preference.RevisionName = preferenceRevision.Name

	return preferenceRevision, nil

}

func GenerateRevisionNamePatch(flavorRevision, preferenceRevision *appsv1.ControllerRevision) ([]byte, error) {

	var patches []utiltypes.PatchOperation

	if flavorRevision != nil {
		patches = append(patches,
			utiltypes.PatchOperation{
				Op:    utiltypes.PatchTestOp,
				Path:  "/spec/flavor/revisionName",
				Value: nil,
			},
			utiltypes.PatchOperation{
				Op:    utiltypes.PatchAddOp,
				Path:  "/spec/flavor/revisionName",
				Value: flavorRevision.Name,
			},
		)
	}

	if preferenceRevision != nil {
		patches = append(patches,
			utiltypes.PatchOperation{
				Op:    utiltypes.PatchTestOp,
				Path:  "/spec/preference/revisionName",
				Value: nil,
			},
			utiltypes.PatchOperation{
				Op:    utiltypes.PatchAddOp,
				Path:  "/spec/preference/revisionName",
				Value: preferenceRevision.Name,
			},
		)
	}
	return utiltypes.GeneratePatchPayload(patches...)
}

func (m *methods) StoreControllerRevisions(vm *virtv1.VirtualMachine) error {

	logger := log.Log.Object(vm)
	flavorRevision, err := m.storeFlavorRevision(vm)
	if err != nil {
		logger.Reason(err).Error("Failed to store ControllerRevision of VirtualMachineFlavorSpec for the Virtualmachine.")
		return err
	}

	preferenceRevision, err := m.storePreferenceRevision(vm)
	if err != nil {
		logger.Reason(err).Error("Failed to store ControllerRevision of VirtualMachinePreferenceSpec for the Virtualmachine.")
		return err
	}

	// Batch any writes to the VirtualMachine into a single Patch() call to avoid races in the controller.
	if flavorRevision != nil || preferenceRevision != nil {

		patch, err := GenerateRevisionNamePatch(flavorRevision, preferenceRevision)
		if err != nil {
			logger.Reason(err).Error("Failed to generate flavor and preference RevisionName patch.")
			return err
		}

		if _, err := m.clientset.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{}); err != nil {
			logger.Reason(err).Error("Failed to update VirtualMachine with flavor and preference ControllerRevision references.")
			return err
		}
	}

	return nil
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

	if len(vm.Spec.Preference.RevisionName) > 0 {
		return m.getPreferenceSpecRevision(vm.Spec.Preference.RevisionName, vm.Namespace)
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

func (m *methods) getPreferenceSpecRevision(revisionName string, namespace string) (*flavorv1alpha1.VirtualMachinePreferenceSpec, error) {

	revision, err := m.clientset.AppsV1().ControllerRevisions(namespace).Get(context.Background(), revisionName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	preferenceSpecRevision := flavorv1alpha1.VirtualMachinePreferenceSpecRevision{}
	err = json.Unmarshal(revision.Data.Raw, &preferenceSpecRevision)
	if err != nil {
		return nil, err
	}

	// For now we only support a single version of VirtualMachinePreferenceSpec but in the future we will need to handle older versions here
	preferenceSpec := flavorv1alpha1.VirtualMachinePreferenceSpec{}
	err = json.Unmarshal(preferenceSpecRevision.Spec, &preferenceSpec)
	if err != nil {
		return nil, err
	}

	return &preferenceSpec, nil
}

func (m *methods) findPreference(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachinePreference, error) {

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	preference, err := m.clientset.VirtualMachinePreference(vm.Namespace).Get(context.Background(), vm.Spec.Preference.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return preference, nil
}

func (m *methods) findClusterPreference(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineClusterPreference, error) {

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	preference, err := m.clientset.VirtualMachineClusterPreference().Get(context.Background(), vm.Spec.Preference.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return preference, nil
}

func (m *methods) FindFlavorSpec(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavorSpec, error) {

	var err error
	var flavor *flavorv1alpha1.VirtualMachineFlavor
	var clusterFlavor *flavorv1alpha1.VirtualMachineClusterFlavor

	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	if len(vm.Spec.Flavor.RevisionName) > 0 {
		return m.getFlavorSpecRevision(vm.Spec.Flavor.RevisionName, vm.Namespace)
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

func (m *methods) getFlavorSpecRevision(revisionName string, namespace string) (*flavorv1alpha1.VirtualMachineFlavorSpec, error) {

	revision, err := m.clientset.AppsV1().ControllerRevisions(namespace).Get(context.Background(), revisionName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	flavorSpecRevision := flavorv1alpha1.VirtualMachineFlavorSpecRevision{}
	err = json.Unmarshal(revision.Data.Raw, &flavorSpecRevision)
	if err != nil {
		return nil, err
	}

	// For now we only support a single version of VirtualMachineFlavorSpec but in the future we will need to handle older versions here
	flavorSpec := flavorv1alpha1.VirtualMachineFlavorSpec{}
	err = json.Unmarshal(flavorSpecRevision.Spec, &flavorSpec)
	if err != nil {
		return nil, err
	}

	return &flavorSpec, nil
}

func (m *methods) findFlavor(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineFlavor, error) {

	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	flavor, err := m.clientset.VirtualMachineFlavor(vm.Namespace).Get(context.Background(), vm.Spec.Flavor.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return flavor, nil
}

func (m *methods) findClusterFlavor(vm *virtv1.VirtualMachine) (*flavorv1alpha1.VirtualMachineClusterFlavor, error) {

	if vm.Spec.Flavor == nil {
		return nil, nil
	}

	flavor, err := m.clientset.VirtualMachineClusterFlavor().Get(context.Background(), vm.Spec.Flavor.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

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

	// Default to PreferSockets when a PreferredCPUTopology isn't provided
	preferredTopology := flavorv1alpha1.PreferSockets
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
