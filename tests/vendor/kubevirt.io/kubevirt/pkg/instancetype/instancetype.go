package instancetype

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	utiltypes "kubevirt.io/kubevirt/pkg/util/types"
)

type Methods interface {
	FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1alpha1.VirtualMachineInstancetypeSpec, error)
	ApplyToVmi(field *k8sfield.Path, instancetypespec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, prefernceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts
	FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1alpha1.VirtualMachinePreferenceSpec, error)
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

func CreateInstancetypeControllerRevision(vm *virtv1.VirtualMachine, revisionName string, instancetypeApiVersion string, instancetypeSpec *instancetypev1alpha1.VirtualMachineInstancetypeSpec) (*appsv1.ControllerRevision, error) {

	instancetypeSpecPatch, err := json.Marshal(*instancetypeSpec)
	if err != nil {
		return nil, err
	}

	specRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{
		APIVersion: instancetypeApiVersion,
		Spec:       instancetypeSpecPatch,
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

func (m *methods) createInstancetypeRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {

	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case apiinstancetype.SingularResourceName, apiinstancetype.PluralResourceName:
		instancetype, err := m.findInstancetype(vm)
		if err != nil {
			return nil, err
		}
		revision, err := CreateInstancetypeControllerRevision(vm, GetRevisionName(vm.Name, instancetype.Name, instancetype.UID, instancetype.Generation), instancetype.APIVersion, &instancetype.Spec)
		if err != nil {
			return nil, err
		}
		return revision, nil

	case apiinstancetype.ClusterSingularResourceName, apiinstancetype.ClusterPluralResourceName:
		clusterInstancetype, err := m.findClusterInstancetype(vm)
		if err != nil {
			return nil, err
		}

		revision, err := CreateInstancetypeControllerRevision(vm, GetRevisionName(vm.Name, clusterInstancetype.Name, clusterInstancetype.UID, clusterInstancetype.Generation), clusterInstancetype.APIVersion, &clusterInstancetype.Spec)
		if err != nil {
			return nil, err
		}
		return revision, nil
	default:
		return nil, fmt.Errorf("got unexpected kind in InstancetypeMatcher: %s", vm.Spec.Instancetype.Kind)
	}
}

func (m *methods) storeInstancetypeRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {

	if vm.Spec.Instancetype == nil || len(vm.Spec.Instancetype.RevisionName) > 0 {
		return nil, nil
	}

	instancetypeRevision, err := m.createInstancetypeRevision(vm)
	if err != nil {
		return nil, err
	}

	_, err = m.clientset.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), instancetypeRevision, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Grab the existing revision to check the data it contains
			existingRevision, err := m.clientset.AppsV1().ControllerRevisions(vm.Namespace).Get(context.Background(), instancetypeRevision.Name, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			// If the data between the two differs return an error, otherwise continue and store the name below.
			if bytes.Compare(existingRevision.Data.Raw, instancetypeRevision.Data.Raw) != 0 {
				return nil, fmt.Errorf("found existing ControllerRevision with unexpected data: %s", instancetypeRevision.Name)
			}
		} else {
			return nil, err
		}
	}

	vm.Spec.Instancetype.RevisionName = instancetypeRevision.Name

	return instancetypeRevision, nil

}

func CreatePreferenceControllerRevision(vm *virtv1.VirtualMachine, revisionName string, preferenceApiVersion string, preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec) (*appsv1.ControllerRevision, error) {

	preferenceSpecPatch, err := json.Marshal(*preferenceSpec)
	if err != nil {
		return nil, err
	}

	specRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{
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
	case apiinstancetype.SingularPreferenceResourceName, apiinstancetype.PluralPreferenceResourceName:
		preference, err := m.findPreference(vm)
		if err != nil {
			return nil, err
		}

		revision, err := CreatePreferenceControllerRevision(vm, GetRevisionName(vm.Name, preference.Name, preference.UID, preference.Generation), preference.APIVersion, &preference.Spec)
		if err != nil {
			return nil, err
		}
		return revision, nil
	case apiinstancetype.ClusterSingularPreferenceResourceName, apiinstancetype.ClusterPluralPreferenceResourceName:
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

func GenerateRevisionNamePatch(instancetypeRevision, preferenceRevision *appsv1.ControllerRevision) ([]byte, error) {

	var patches []utiltypes.PatchOperation

	if instancetypeRevision != nil {
		patches = append(patches,
			utiltypes.PatchOperation{
				Op:    utiltypes.PatchTestOp,
				Path:  "/spec/instancetype/revisionName",
				Value: nil,
			},
			utiltypes.PatchOperation{
				Op:    utiltypes.PatchAddOp,
				Path:  "/spec/instancetype/revisionName",
				Value: instancetypeRevision.Name,
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
	instancetypeRevision, err := m.storeInstancetypeRevision(vm)
	if err != nil {
		logger.Reason(err).Error("Failed to store ControllerRevision of VirtualMachineInstancetypeSpec for the Virtualmachine.")
		return err
	}

	preferenceRevision, err := m.storePreferenceRevision(vm)
	if err != nil {
		logger.Reason(err).Error("Failed to store ControllerRevision of VirtualMachinePreferenceSpec for the Virtualmachine.")
		return err
	}

	// Batch any writes to the VirtualMachine into a single Patch() call to avoid races in the controller.
	if instancetypeRevision != nil || preferenceRevision != nil {

		patch, err := GenerateRevisionNamePatch(instancetypeRevision, preferenceRevision)
		if err != nil {
			logger.Reason(err).Error("Failed to generate instancetype and preference RevisionName patch.")
			return err
		}

		if _, err := m.clientset.VirtualMachine(vm.Namespace).Patch(vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{}); err != nil {
			logger.Reason(err).Error("Failed to update VirtualMachine with instancetype and preference ControllerRevision references.")
			return err
		}
	}

	return nil
}

func (m *methods) ApplyToVmi(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	var conflicts Conflicts

	if instancetypeSpec != nil {
		conflicts = append(conflicts, applyCpu(field, instancetypeSpec, preferenceSpec, vmiSpec)...)
		conflicts = append(conflicts, applyMemory(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyIOThreadPolicy(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyLaunchSecurity(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyGPUs(field, instancetypeSpec, vmiSpec)...)
		conflicts = append(conflicts, applyHostDevices(field, instancetypeSpec, vmiSpec)...)
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

func (m *methods) FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1alpha1.VirtualMachinePreferenceSpec, error) {

	var err error
	var preference *instancetypev1alpha1.VirtualMachinePreference
	var clusterPreference *instancetypev1alpha1.VirtualMachineClusterPreference

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	if len(vm.Spec.Preference.RevisionName) > 0 {
		return m.getPreferenceSpecRevision(vm.Spec.Preference.RevisionName, vm.Namespace)
	}

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiinstancetype.SingularPreferenceResourceName, apiinstancetype.PluralPreferenceResourceName:
		preference, err = m.findPreference(vm)
	case apiinstancetype.ClusterSingularPreferenceResourceName, apiinstancetype.ClusterPluralPreferenceResourceName:
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

func (m *methods) getPreferenceSpecRevision(revisionName string, namespace string) (*instancetypev1alpha1.VirtualMachinePreferenceSpec, error) {

	revision, err := m.clientset.AppsV1().ControllerRevisions(namespace).Get(context.Background(), revisionName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	preferenceSpecRevision := instancetypev1alpha1.VirtualMachinePreferenceSpecRevision{}
	err = json.Unmarshal(revision.Data.Raw, &preferenceSpecRevision)
	if err != nil {
		return nil, err
	}

	// For now we only support a single version of VirtualMachinePreferenceSpec but in the future we will need to handle older versions here
	preferenceSpec := instancetypev1alpha1.VirtualMachinePreferenceSpec{}
	err = json.Unmarshal(preferenceSpecRevision.Spec, &preferenceSpec)
	if err != nil {
		return nil, err
	}

	return &preferenceSpec, nil
}

func (m *methods) findPreference(vm *virtv1.VirtualMachine) (*instancetypev1alpha1.VirtualMachinePreference, error) {

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	preference, err := m.clientset.VirtualMachinePreference(vm.Namespace).Get(context.Background(), vm.Spec.Preference.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return preference, nil
}

func (m *methods) findClusterPreference(vm *virtv1.VirtualMachine) (*instancetypev1alpha1.VirtualMachineClusterPreference, error) {

	if vm.Spec.Preference == nil {
		return nil, nil
	}

	preference, err := m.clientset.VirtualMachineClusterPreference().Get(context.Background(), vm.Spec.Preference.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return preference, nil
}

func (m *methods) FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1alpha1.VirtualMachineInstancetypeSpec, error) {

	var err error
	var instancetype *instancetypev1alpha1.VirtualMachineInstancetype
	var clusterInstancetype *instancetypev1alpha1.VirtualMachineClusterInstancetype

	if vm.Spec.Instancetype == nil {
		return nil, nil
	}

	if len(vm.Spec.Instancetype.RevisionName) > 0 {
		return m.getInstancetypeSpecRevision(vm.Spec.Instancetype.RevisionName, vm.Namespace)
	}

	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case apiinstancetype.SingularResourceName, apiinstancetype.PluralResourceName:
		instancetype, err = m.findInstancetype(vm)
	case apiinstancetype.ClusterSingularResourceName, apiinstancetype.ClusterPluralResourceName, "":
		clusterInstancetype, err = m.findClusterInstancetype(vm)
	default:
		err = fmt.Errorf("got unexpected kind in InstancetypeMatcher: %s", vm.Spec.Instancetype.Kind)
	}

	if err != nil {
		return nil, err
	}

	if instancetype != nil {
		return &instancetype.Spec, nil
	}

	if clusterInstancetype != nil {
		return &clusterInstancetype.Spec, nil
	}

	return nil, nil
}

func (m *methods) getInstancetypeSpecRevision(revisionName string, namespace string) (*instancetypev1alpha1.VirtualMachineInstancetypeSpec, error) {

	revision, err := m.clientset.AppsV1().ControllerRevisions(namespace).Get(context.Background(), revisionName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	instancetypeSpecRevision := instancetypev1alpha1.VirtualMachineInstancetypeSpecRevision{}
	err = json.Unmarshal(revision.Data.Raw, &instancetypeSpecRevision)
	if err != nil {
		return nil, err
	}

	// For now we only support a single version of VirtualMachineInstancetypeSpec but in the future we will need to handle older versions here
	instancetypeSpec := instancetypev1alpha1.VirtualMachineInstancetypeSpec{}
	err = json.Unmarshal(instancetypeSpecRevision.Spec, &instancetypeSpec)
	if err != nil {
		return nil, err
	}

	return &instancetypeSpec, nil
}

func (m *methods) findInstancetype(vm *virtv1.VirtualMachine) (*instancetypev1alpha1.VirtualMachineInstancetype, error) {

	if vm.Spec.Instancetype == nil {
		return nil, nil
	}

	instancetype, err := m.clientset.VirtualMachineInstancetype(vm.Namespace).Get(context.Background(), vm.Spec.Instancetype.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return instancetype, nil
}

func (m *methods) findClusterInstancetype(vm *virtv1.VirtualMachine) (*instancetypev1alpha1.VirtualMachineClusterInstancetype, error) {

	if vm.Spec.Instancetype == nil {
		return nil, nil
	}

	instancetype, err := m.clientset.VirtualMachineClusterInstancetype().Get(context.Background(), vm.Spec.Instancetype.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return instancetype, nil
}

func applyCpu(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
	if vmiSpec.Domain.CPU != nil {
		return Conflicts{field.Child("domain", "cpu")}
	}

	if _, hasCPURequests := vmiSpec.Domain.Resources.Requests[k8sv1.ResourceCPU]; hasCPURequests {
		return Conflicts{field.Child("domain", "resources", "requests", string(k8sv1.ResourceCPU))}
	}

	if _, hasCPULimits := vmiSpec.Domain.Resources.Limits[k8sv1.ResourceCPU]; hasCPULimits {
		return Conflicts{field.Child("domain", "resources", "limits", string(k8sv1.ResourceCPU))}
	}

	vmiSpec.Domain.CPU = &virtv1.CPU{
		Sockets:               uint32(1),
		Cores:                 uint32(1),
		Threads:               uint32(1),
		Model:                 instancetypeSpec.CPU.Model,
		DedicatedCPUPlacement: instancetypeSpec.CPU.DedicatedCPUPlacement,
		IsolateEmulatorThread: instancetypeSpec.CPU.IsolateEmulatorThread,
		NUMA:                  instancetypeSpec.CPU.NUMA.DeepCopy(),
		Realtime:              instancetypeSpec.CPU.Realtime.DeepCopy(),
	}

	// Default to PreferSockets when a PreferredCPUTopology isn't provided
	preferredTopology := instancetypev1alpha1.PreferSockets
	if preferenceSpec != nil && preferenceSpec.CPU != nil && preferenceSpec.CPU.PreferredCPUTopology != "" {
		preferredTopology = preferenceSpec.CPU.PreferredCPUTopology
	}

	switch preferredTopology {
	case instancetypev1alpha1.PreferCores:
		vmiSpec.Domain.CPU.Cores = instancetypeSpec.CPU.Guest
	case instancetypev1alpha1.PreferSockets:
		vmiSpec.Domain.CPU.Sockets = instancetypeSpec.CPU.Guest
	case instancetypev1alpha1.PreferThreads:
		vmiSpec.Domain.CPU.Threads = instancetypeSpec.CPU.Guest
	}

	return nil
}

func AddInstancetypeNameAnnotations(vm *virtv1.VirtualMachine, target metav1.Object) {
	if vm.Spec.Instancetype == nil {
		return
	}

	if target.GetAnnotations() == nil {
		target.SetAnnotations(make(map[string]string))
	}
	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case apiinstancetype.PluralResourceName, apiinstancetype.SingularResourceName:
		target.GetAnnotations()[virtv1.InstancetypeAnnotation] = vm.Spec.Instancetype.Name
	case "", apiinstancetype.ClusterPluralResourceName, apiinstancetype.ClusterSingularResourceName:
		target.GetAnnotations()[virtv1.ClusterInstancetypeAnnotation] = vm.Spec.Instancetype.Name
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
	case apiinstancetype.PluralPreferenceResourceName, apiinstancetype.SingularPreferenceResourceName:
		target.GetAnnotations()[virtv1.PreferenceAnnotation] = vm.Spec.Preference.Name
	case "", apiinstancetype.ClusterPluralPreferenceResourceName, apiinstancetype.ClusterSingularPreferenceResourceName:
		target.GetAnnotations()[virtv1.ClusterPreferenceAnnotation] = vm.Spec.Preference.Name
	}
}

func applyMemory(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if vmiSpec.Domain.Memory != nil {
		return Conflicts{field.Child("domain", "memory")}
	}

	if _, hasMemoryRequests := vmiSpec.Domain.Resources.Requests[k8sv1.ResourceMemory]; hasMemoryRequests {
		return Conflicts{field.Child("domain", "resources", "requests", string(k8sv1.ResourceMemory))}
	}

	if _, hasMemoryLimits := vmiSpec.Domain.Resources.Limits[k8sv1.ResourceMemory]; hasMemoryLimits {
		return Conflicts{field.Child("domain", "resources", "limits", string(k8sv1.ResourceMemory))}
	}

	instancetypeMemoryGuest := instancetypeSpec.Memory.Guest.DeepCopy()
	vmiSpec.Domain.Memory = &virtv1.Memory{
		Guest: &instancetypeMemoryGuest,
	}

	if instancetypeSpec.Memory.Hugepages != nil {
		vmiSpec.Domain.Memory.Hugepages = instancetypeSpec.Memory.Hugepages.DeepCopy()
	}

	return nil
}

func applyIOThreadPolicy(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if instancetypeSpec.IOThreadsPolicy == nil {
		return nil
	}

	if vmiSpec.Domain.IOThreadsPolicy != nil {
		return Conflicts{field.Child("domain", "ioThreadsPolicy")}
	}

	instancetypeIOThreadPolicy := *instancetypeSpec.IOThreadsPolicy
	vmiSpec.Domain.IOThreadsPolicy = &instancetypeIOThreadPolicy

	return nil
}

func applyLaunchSecurity(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if instancetypeSpec.LaunchSecurity == nil {
		return nil
	}

	if vmiSpec.Domain.LaunchSecurity != nil {
		return Conflicts{field.Child("domain", "launchSecurity")}
	}

	vmiSpec.Domain.LaunchSecurity = instancetypeSpec.LaunchSecurity.DeepCopy()

	return nil
}

func applyGPUs(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if len(instancetypeSpec.GPUs) == 0 {
		return nil
	}

	if len(vmiSpec.Domain.Devices.GPUs) >= 1 {
		return Conflicts{field.Child("domain", "devices", "gpus")}
	}

	vmiSpec.Domain.Devices.GPUs = make([]v1.GPU, len(instancetypeSpec.GPUs))
	copy(vmiSpec.Domain.Devices.GPUs, instancetypeSpec.GPUs)

	return nil
}

func applyHostDevices(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha1.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if len(instancetypeSpec.HostDevices) == 0 {
		return nil
	}

	if len(vmiSpec.Domain.Devices.HostDevices) >= 1 {
		return Conflicts{field.Child("domain", "devices", "hostDevices")}
	}

	vmiSpec.Domain.Devices.HostDevices = make([]v1.HostDevice, len(instancetypeSpec.HostDevices))
	copy(vmiSpec.Domain.Devices.HostDevices, instancetypeSpec.HostDevices)

	return nil
}

func applyDevicePreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

func applyDiskPreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
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

func applyInterfacePreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for ifaceIndex := range vmiSpec.Domain.Devices.Interfaces {
		vmiIface := &vmiSpec.Domain.Devices.Interfaces[ifaceIndex]
		if preferenceSpec.Devices.PreferredInterfaceModel != "" && vmiIface.Model == "" {
			vmiIface.Model = preferenceSpec.Devices.PreferredInterfaceModel
		}
	}
}

func applyInputPreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
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

func applyFeaturePreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

func applyHyperVFeaturePreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

func applyFirmwarePreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

func applyMachinePreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

func applyClockPreferences(preferenceSpec *instancetypev1alpha1.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
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
