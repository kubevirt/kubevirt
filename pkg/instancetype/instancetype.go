package instancetype

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfield "k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	apiinstancetype "kubevirt.io/api/instancetype"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	utils "kubevirt.io/kubevirt/pkg/util"
)

type Methods interface {
	FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error)
	ApplyToVmi(field *k8sfield.Path, instancetypespec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, prefernceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts
	FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error)
	StoreControllerRevisions(vm *virtv1.VirtualMachine) error
	InferDefaultInstancetype(vm *virtv1.VirtualMachine) (*virtv1.InstancetypeMatcher, error)
	InferDefaultPreference(vm *virtv1.VirtualMachine) (*virtv1.PreferenceMatcher, error)
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
	instancetypeStore        cache.Store
	clusterInstancetypeStore cache.Store
	preferenceStore          cache.Store
	clusterPreferenceStore   cache.Store
	controllerRevisionStore  cache.Store
	clientset                kubecli.KubevirtClient
}

var _ Methods = &methods{}

func NewMethods(instancetypeStore, clusterInstancetypeStore, preferenceStore, clusterPreferenceStore, controllerRevisionStore cache.Store, clientset kubecli.KubevirtClient) Methods {
	return &methods{
		instancetypeStore:        instancetypeStore,
		clusterInstancetypeStore: clusterInstancetypeStore,
		preferenceStore:          preferenceStore,
		clusterPreferenceStore:   clusterPreferenceStore,
		controllerRevisionStore:  controllerRevisionStore,
		clientset:                clientset,
	}
}

func GetRevisionName(vmName, resourceName string, resourceUID types.UID, resourceGeneration int64) string {
	return fmt.Sprintf("%s-%s-%s-%d", vmName, resourceName, resourceUID, resourceGeneration)
}

func CreateControllerRevision(vm *virtv1.VirtualMachine, object runtime.Object) (*appsv1.ControllerRevision, error) {
	obj, err := utils.GenerateKubeVirtGroupVersionKind(object)
	if err != nil {
		return nil, err
	}
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("Unexpected object format returned from GenerateKubeVirtGroupVersionKind")
	}

	revisionName := GetRevisionName(vm.Name, metaObj.GetName(), metaObj.GetUID(), metaObj.GetGeneration())

	// Removing unnecessary metadata
	metaObj.SetLabels(nil)
	metaObj.SetAnnotations(nil)
	metaObj.SetFinalizers(nil)
	metaObj.SetOwnerReferences(nil)
	metaObj.SetManagedFields(nil)

	return &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Name:            revisionName,
			Namespace:       vm.Namespace,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(vm, virtv1.VirtualMachineGroupVersionKind)},
		},
		Data: runtime.RawExtension{
			Object: obj,
		},
	}, nil
}

func (m *methods) checkForInstancetypeConflicts(instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) error {
	// Apply the instancetype to a copy of the VMISpec as we don't want to persist any changes here in the VM being passed around
	vmiSpecCopy := vmiSpec.DeepCopy()
	conflicts := m.ApplyToVmi(k8sfield.NewPath("spec", "template", "spec"), instancetypeSpec, nil, vmiSpecCopy)
	if len(conflicts) > 0 {
		return fmt.Errorf("VM field conflicts with selected Instancetype: %v", conflicts.String())
	}
	return nil
}

func (m *methods) createInstancetypeRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case apiinstancetype.SingularResourceName, apiinstancetype.PluralResourceName:
		instancetype, err := m.findInstancetype(vm)
		if err != nil {
			return nil, err
		}

		// There is still a window where the instancetype can be updated between the VirtualMachine validation webhook accepting
		// the VirtualMachine and the VirtualMachine controller creating a ControllerRevison. As such we need to check one final
		// time that there are no conflicts when applying the instancetype to the VirtualMachine before continuing.
		if err = m.checkForInstancetypeConflicts(&instancetype.Spec, &vm.Spec.Template.Spec); err != nil {
			return nil, err
		}
		return CreateControllerRevision(vm, instancetype)

	case apiinstancetype.ClusterSingularResourceName, apiinstancetype.ClusterPluralResourceName:
		clusterInstancetype, err := m.findClusterInstancetype(vm)
		if err != nil {
			return nil, err
		}

		// There is still a window where the instancetype can be updated between the VirtualMachine validation webhook accepting
		// the VirtualMachine and the VirtualMachine controller creating a ControllerRevison. As such we need to check one final
		// time that there are no conflicts when applying the instancetype to the VirtualMachine before continuing.
		if err = m.checkForInstancetypeConflicts(&clusterInstancetype.Spec, &vm.Spec.Template.Spec); err != nil {
			return nil, err
		}
		return CreateControllerRevision(vm, clusterInstancetype)

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

	storedRevision, err := storeRevision(instancetypeRevision, m.clientset, false)
	if err != nil {
		return nil, err
	}

	vm.Spec.Instancetype.RevisionName = storedRevision.Name
	return storedRevision, nil
}

func (m *methods) createPreferenceRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiinstancetype.SingularPreferenceResourceName, apiinstancetype.PluralPreferenceResourceName:
		preference, err := m.findPreference(vm)
		if err != nil {
			return nil, err
		}
		return CreateControllerRevision(vm, preference)
	case apiinstancetype.ClusterSingularPreferenceResourceName, apiinstancetype.ClusterPluralPreferenceResourceName:
		clusterPreference, err := m.findClusterPreference(vm)
		if err != nil {
			return nil, err
		}
		return CreateControllerRevision(vm, clusterPreference)
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

	storedRevision, err := storeRevision(preferenceRevision, m.clientset, true)
	if err != nil {
		return nil, err
	}

	vm.Spec.Preference.RevisionName = storedRevision.Name
	return storedRevision, nil
}

func GenerateRevisionNamePatch(instancetypeRevision, preferenceRevision *appsv1.ControllerRevision) ([]byte, error) {
	var patches []patch.PatchOperation

	if instancetypeRevision != nil {
		patches = append(patches,
			patch.PatchOperation{
				Op:    patch.PatchTestOp,
				Path:  "/spec/instancetype/revisionName",
				Value: nil,
			},
			patch.PatchOperation{
				Op:    patch.PatchAddOp,
				Path:  "/spec/instancetype/revisionName",
				Value: instancetypeRevision.Name,
			},
		)
	}

	if preferenceRevision != nil {
		patches = append(patches,
			patch.PatchOperation{
				Op:    patch.PatchTestOp,
				Path:  "/spec/preference/revisionName",
				Value: nil,
			},
			patch.PatchOperation{
				Op:    patch.PatchAddOp,
				Path:  "/spec/preference/revisionName",
				Value: preferenceRevision.Name,
			},
		)
	}

	if len(patches) == 0 {
		return nil, nil
	}

	payload, err := patch.GeneratePatchPayload(patches...)
	if err != nil {
		// This is a programmer's error and should not happen
		return nil, fmt.Errorf("failed to generate patch payload: %w", err)
	}

	return payload, nil
}

func (m *methods) StoreControllerRevisions(vm *virtv1.VirtualMachine) error {

	// Lazy logger construction
	logger := func() *log.FilteredLogger { return log.Log.Object(vm) }
	instancetypeRevision, err := m.storeInstancetypeRevision(vm)
	if err != nil {
		logger().Reason(err).Error("Failed to store ControllerRevision of VirtualMachineInstancetypeSpec for the Virtualmachine.")
		return err
	}

	preferenceRevision, err := m.storePreferenceRevision(vm)
	if err != nil {
		logger().Reason(err).Error("Failed to store ControllerRevision of VirtualMachinePreferenceSpec for the Virtualmachine.")
		return err
	}

	// Batch any writes to the VirtualMachine into a single Patch() call to avoid races in the controller.
	patch, err := GenerateRevisionNamePatch(instancetypeRevision, preferenceRevision)
	if err != nil {
		return err
	}
	if len(patch) > 0 {
		if _, err := m.clientset.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patch, &metav1.PatchOptions{}); err != nil {
			logger().Reason(err).Error("Failed to update VirtualMachine with instancetype and preference ControllerRevision references.")
			return err
		}
	}

	return nil
}

func CompareRevisions(revisionA *appsv1.ControllerRevision, revisionB *appsv1.ControllerRevision, isPreference bool) (bool, error) {
	if err := decodeRevisionObject(revisionA, isPreference); err != nil {
		return false, err
	}

	if err := decodeRevisionObject(revisionB, isPreference); err != nil {
		return false, err
	}

	revisionASpec, err := getInstancetypeApiSpec(revisionA.Data.Object)
	if err != nil {
		return false, err
	}

	revisionBSpec, err := getInstancetypeApiSpec(revisionB.Data.Object)
	if err != nil {
		return false, err
	}

	return equality.Semantic.DeepEqual(revisionASpec, revisionBSpec), nil
}

func storeRevision(revision *appsv1.ControllerRevision, clientset kubecli.KubevirtClient, isPreference bool) (*appsv1.ControllerRevision, error) {
	createdRevision, err := clientset.AppsV1().ControllerRevisions(revision.Namespace).Create(context.Background(), revision, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create ControllerRevision: %w", err)
		}

		// Grab the existing revision to check the data it contains
		existingRevision, err := clientset.AppsV1().ControllerRevisions(revision.Namespace).Get(context.Background(), revision.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get ControllerRevision: %w", err)
		}

		equal, err := CompareRevisions(revision, existingRevision, isPreference)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, fmt.Errorf("found existing ControllerRevision with unexpected data: %s", revision.Name)
		}
		return existingRevision, nil
	}
	return createdRevision, nil
}

func decodeRevisionObject(revision *appsv1.ControllerRevision, isPreference bool) error {
	if len(revision.Data.Raw) == 0 {
		return nil
	}

	// Backward compatibility check. Try to decode ControllerRevision from v1alpha1 version.
	oldObject, err := decodeOldObject(revision.Data.Raw, isPreference)
	if err != nil {
		return fmt.Errorf("failed to decode old ControllerRevision: %w", err)
	}
	if oldObject != nil {
		revision.Data.Object = oldObject
		return nil
	}

	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), revision.Data.Raw)
	if err != nil {
		return fmt.Errorf("failed to decode object in ControllerRevision: %w", err)
	}
	revision.Data.Object = decodedObj
	return nil
}

func getInstancetypeApiSpec(obj runtime.Object) (interface{}, error) {
	switch o := obj.(type) {
	case *instancetypev1alpha2.VirtualMachineInstancetype:
		return &o.Spec, nil
	case *instancetypev1alpha2.VirtualMachineClusterInstancetype:
		return &o.Spec, nil
	case *instancetypev1alpha2.VirtualMachinePreference:
		return &o.Spec, nil
	case *instancetypev1alpha2.VirtualMachineClusterPreference:
		return &o.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type: %T", obj)
	}
}

func (m *methods) ApplyToVmi(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
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
		ApplyDevicePreferences(preferenceSpec, vmiSpec)
		applyFeaturePreferences(preferenceSpec, vmiSpec)
		applyFirmwarePreferences(preferenceSpec, vmiSpec)
		applyMachinePreferences(preferenceSpec, vmiSpec)
		applyClockPreferences(preferenceSpec, vmiSpec)
	}

	return conflicts
}

func (m *methods) FindPreferenceSpec(vm *virtv1.VirtualMachine) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}

	if len(vm.Spec.Preference.RevisionName) > 0 {
		return m.getPreferenceSpecRevision(types.NamespacedName{
			Namespace: vm.Namespace,
			Name:      vm.Spec.Preference.RevisionName,
		})
	}

	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case apiinstancetype.SingularPreferenceResourceName, apiinstancetype.PluralPreferenceResourceName:
		preference, err := m.findPreference(vm)
		if err != nil {
			return nil, err
		}
		return &preference.Spec, nil

	case apiinstancetype.ClusterSingularPreferenceResourceName, apiinstancetype.ClusterPluralPreferenceResourceName:
		clusterPreference, err := m.findClusterPreference(vm)
		if err != nil {
			return nil, err
		}
		return &clusterPreference.Spec, nil

	default:
		return nil, fmt.Errorf("got unexpected kind in PreferenceMatcher: %s", vm.Spec.Preference.Kind)
	}
}

func (m *methods) getPreferenceSpecRevision(namespacedName types.NamespacedName) (*instancetypev1alpha2.VirtualMachinePreferenceSpec, error) {
	var (
		err      error
		revision *appsv1.ControllerRevision
	)

	if m.controllerRevisionStore != nil {
		revision, err = m.getControllerRevisionByInformer(namespacedName)
	} else {
		revision, err = m.getControllerRevisionByClient(namespacedName)
	}

	if err != nil {
		return nil, err
	}

	if err = decodeRevisionObject(revision, true); err != nil {
		return nil, err
	}

	switch obj := revision.Data.Object.(type) {
	case *instancetypev1alpha2.VirtualMachinePreference:
		return &obj.Spec, nil
	case *instancetypev1alpha2.VirtualMachineClusterPreference:
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}

func (m *methods) getControllerRevisionByInformer(namespacedName types.NamespacedName) (*appsv1.ControllerRevision, error) {
	obj, exists, err := m.controllerRevisionStore.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		return m.getControllerRevisionByClient(namespacedName)
	}
	revision, ok := obj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in ControllerRevision informer")
	}
	return revision, nil
}

func (m *methods) getControllerRevisionByClient(namespacedName types.NamespacedName) (*appsv1.ControllerRevision, error) {
	revision, err := m.clientset.AppsV1().ControllerRevisions(namespacedName.Namespace).Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return revision, nil
}

func (m *methods) findPreference(vm *virtv1.VirtualMachine) (*instancetypev1alpha2.VirtualMachinePreference, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}
	namespacedName := types.NamespacedName{
		Namespace: vm.Namespace,
		Name:      vm.Spec.Preference.Name,
	}
	if m.preferenceStore != nil {
		return m.findPreferenceByInformer(namespacedName)
	}
	return m.findPreferenceByClient(namespacedName)
}

func (m *methods) findPreferenceByInformer(namespacedName types.NamespacedName) (*instancetypev1alpha2.VirtualMachinePreference, error) {
	obj, exists, err := m.preferenceStore.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		return m.findPreferenceByClient(namespacedName)
	}
	preference, ok := obj.(*instancetypev1alpha2.VirtualMachinePreference)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachinePreference informer")
	}
	return preference, nil
}

func (m *methods) findPreferenceByClient(namespacedName types.NamespacedName) (*instancetypev1alpha2.VirtualMachinePreference, error) {
	preference, err := m.clientset.VirtualMachinePreference(namespacedName.Namespace).Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return preference, nil
}

func (m *methods) findClusterPreference(vm *virtv1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineClusterPreference, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}
	if m.clusterPreferenceStore != nil {
		return m.findClusterPreferenceByInformer(vm.Spec.Preference.Name)
	}
	return m.findClusterPreferenceByClient(vm.Spec.Preference.Name)
}

func (m *methods) findClusterPreferenceByInformer(resourceName string) (*instancetypev1alpha2.VirtualMachineClusterPreference, error) {
	obj, exists, err := m.preferenceStore.GetByKey(resourceName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return m.findClusterPreferenceByClient(resourceName)
	}
	preference, ok := obj.(*instancetypev1alpha2.VirtualMachineClusterPreference)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachineClusterPreference informer")
	}
	return preference, nil
}

func (m *methods) findClusterPreferenceByClient(resourceName string) (*instancetypev1alpha2.VirtualMachineClusterPreference, error) {
	preference, err := m.clientset.VirtualMachineClusterPreference().Get(context.Background(), resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return preference, nil
}

func (m *methods) FindInstancetypeSpec(vm *virtv1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}

	if len(vm.Spec.Instancetype.RevisionName) > 0 {
		return m.getInstancetypeSpecRevision(types.NamespacedName{
			Namespace: vm.Namespace,
			Name:      vm.Spec.Instancetype.RevisionName,
		})
	}

	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case apiinstancetype.SingularResourceName, apiinstancetype.PluralResourceName:
		instancetype, err := m.findInstancetype(vm)
		if err != nil {
			return nil, err
		}
		return &instancetype.Spec, nil

	case apiinstancetype.ClusterSingularResourceName, apiinstancetype.ClusterPluralResourceName, "":
		clusterInstancetype, err := m.findClusterInstancetype(vm)
		if err != nil {
			return nil, err
		}
		return &clusterInstancetype.Spec, nil

	default:
		return nil, fmt.Errorf("got unexpected kind in InstancetypeMatcher: %s", vm.Spec.Instancetype.Kind)
	}
}

func (m *methods) getInstancetypeSpecRevision(namespacedName types.NamespacedName) (*instancetypev1alpha2.VirtualMachineInstancetypeSpec, error) {
	var (
		err      error
		revision *appsv1.ControllerRevision
	)

	if m.controllerRevisionStore != nil {
		revision, err = m.getControllerRevisionByInformer(namespacedName)
	} else {
		revision, err = m.getControllerRevisionByClient(namespacedName)
	}

	if err != nil {
		return nil, err
	}

	if err := decodeRevisionObject(revision, false); err != nil {
		return nil, err
	}

	switch obj := revision.Data.Object.(type) {
	case *instancetypev1alpha2.VirtualMachineInstancetype:
		return &obj.Spec, nil
	case *instancetypev1alpha2.VirtualMachineClusterInstancetype:
		return &obj.Spec, nil
	default:
		return nil, fmt.Errorf("unexpected type in ControllerRevision: %T", obj)
	}
}

func (m *methods) findInstancetype(vm *virtv1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineInstancetype, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}
	namespacedName := types.NamespacedName{
		Namespace: vm.Namespace,
		Name:      vm.Spec.Instancetype.Name,
	}
	if m.instancetypeStore != nil {
		return m.findInstancetypeByInformer(namespacedName)
	}
	return m.findInstancetypeByClient(namespacedName)
}

func (m *methods) findInstancetypeByInformer(namespacedName types.NamespacedName) (*instancetypev1alpha2.VirtualMachineInstancetype, error) {
	obj, exists, err := m.instancetypeStore.GetByKey(namespacedName.String())
	if err != nil {
		return nil, err
	}
	if !exists {
		return m.findInstancetypeByClient(namespacedName)
	}
	instancetype, ok := obj.(*instancetypev1alpha2.VirtualMachineInstancetype)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachineInstancetype informer")
	}
	return instancetype, nil
}

func (m *methods) findInstancetypeByClient(namespacedName types.NamespacedName) (*instancetypev1alpha2.VirtualMachineInstancetype, error) {
	instancetype, err := m.clientset.VirtualMachineInstancetype(namespacedName.Namespace).Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return instancetype, nil
}

func (m *methods) findClusterInstancetype(vm *virtv1.VirtualMachine) (*instancetypev1alpha2.VirtualMachineClusterInstancetype, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}
	if m.clusterInstancetypeStore != nil {
		return m.findClusterInstancetypeByInformer(vm.Spec.Instancetype.Name)
	}
	return m.findClusterInstancetypeByClient(vm.Spec.Instancetype.Name)
}

func (m *methods) findClusterInstancetypeByInformer(resourceName string) (*instancetypev1alpha2.VirtualMachineClusterInstancetype, error) {
	obj, exists, err := m.clusterInstancetypeStore.GetByKey(resourceName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return m.findClusterInstancetypeByClient(resourceName)
	}
	instancetype, ok := obj.(*instancetypev1alpha2.VirtualMachineClusterInstancetype)
	if !ok {
		return nil, fmt.Errorf("unknown object type found in VirtualMachineClusterInstancetype informer")
	}
	return instancetype, nil
}

func (m *methods) findClusterInstancetypeByClient(resourceName string) (*instancetypev1alpha2.VirtualMachineClusterInstancetype, error) {
	instancetype, err := m.clientset.VirtualMachineClusterInstancetype().Get(context.Background(), resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return instancetype, nil
}

func (m *methods) InferDefaultInstancetype(vm *virtv1.VirtualMachine) (*virtv1.InstancetypeMatcher, error) {
	if vm.Spec.Instancetype == nil || vm.Spec.Instancetype.InferFromVolume == "" {
		return nil, nil
	}
	defaultName, defaultKind, err := m.inferDefaultsFromVolumes(vm, vm.Spec.Instancetype.InferFromVolume, apiinstancetype.DefaultInstancetypeLabel, apiinstancetype.DefaultInstancetypeKindLabel)
	if err != nil {
		return nil, err
	}
	return &v1.InstancetypeMatcher{
		Name: defaultName,
		Kind: defaultKind,
	}, nil
}

func (m *methods) InferDefaultPreference(vm *virtv1.VirtualMachine) (*virtv1.PreferenceMatcher, error) {
	if vm.Spec.Preference == nil || vm.Spec.Preference.InferFromVolume == "" {
		return nil, nil
	}
	defaultName, defaultKind, err := m.inferDefaultsFromVolumes(vm, vm.Spec.Preference.InferFromVolume, apiinstancetype.DefaultPreferenceLabel, apiinstancetype.DefaultPreferenceKindLabel)
	if err != nil {
		return nil, err
	}
	return &v1.PreferenceMatcher{
		Name: defaultName,
		Kind: defaultKind,
	}, nil
}

/*
Defaults will be inferred from the following combinations of DataVolumeSources, DataVolumeTemplates, DataSources and PVCs:

Volume -> PersistentVolumeClaimVolumeSource -> PersistentVolumeClaim
Volume -> DataVolumeSource -> DataVolume
Volume -> DataVolumeSource -> DataVolumeSourcePVC -> PersistentVolumeClaim
Volume -> DataVolumeSource -> DataVolumeSourceRef -> DataSource
Volume -> DataVolumeSource -> DataVolumeSourceRef -> DataSource -> PersistentVolumeClaim
Volume -> DataVolumeSource -> DataVolumeTemplate -> DataVolumeSourcePVC -> PersistentVolumeClaim
Volume -> DataVolumeSource -> DataVolumeTemplate -> DataVolumeSourceRef -> DataSource
Volume -> DataVolumeSource -> DataVolumeTemplate -> DataVolumeSourceRef -> DataSource -> PersistentVolumeClaim
*/
func (m *methods) inferDefaultsFromVolumes(vm *virtv1.VirtualMachine, inferFromVolumeName, defaultNameLabel, defaultKindLabel string) (string, string, error) {
	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.Name != inferFromVolumeName {
			continue
		}
		if volume.PersistentVolumeClaim != nil {
			return m.inferDefaultsFromPVC(volume.PersistentVolumeClaim.ClaimName, vm.Namespace, defaultNameLabel, defaultKindLabel)
		}
		if volume.DataVolume != nil {
			return m.inferDefaultsFromDataVolume(vm, volume.DataVolume.Name, defaultNameLabel, defaultKindLabel)
		}
		return "", "", fmt.Errorf("unable to infer defaults from volume %s as type is not supported", inferFromVolumeName)
	}
	return "", "", fmt.Errorf("unable to find volume %s to infer defaults", inferFromVolumeName)
}

func inferDefaultsFromLabels(labels map[string]string, defaultNameLabel, defaultKindLabel string) (string, string, error) {
	defaultName, hasLabel := labels[defaultNameLabel]
	if !hasLabel {
		return "", "", fmt.Errorf("unable to find required %s label on the volume", defaultNameLabel)
	}
	return defaultName, labels[defaultKindLabel], nil
}

func (m *methods) inferDefaultsFromPVC(pvcName, pvcNamespace, defaultNameLabel, defaultKindLabel string) (string, string, error) {
	pvc, err := m.clientset.CoreV1().PersistentVolumeClaims(pvcNamespace).Get(context.Background(), pvcName, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	return inferDefaultsFromLabels(pvc.Labels, defaultNameLabel, defaultKindLabel)
}

func (m *methods) inferDefaultsFromDataVolume(vm *virtv1.VirtualMachine, dvName, defaultNameLabel, defaultKindLabel string) (string, string, error) {
	if len(vm.Spec.DataVolumeTemplates) > 0 {
		for _, dvt := range vm.Spec.DataVolumeTemplates {
			if dvt.Name != dvName {
				continue
			}
			return m.inferDefaultsFromDataVolumeSpec(&dvt.Spec, defaultNameLabel, defaultKindLabel)
		}
	}
	dv, err := m.clientset.CdiClient().CdiV1beta1().DataVolumes(vm.Namespace).Get(context.Background(), dvName, metav1.GetOptions{})
	if err != nil {
		// Handle garbage collected DataVolumes by attempting to lookup the PVC using the name of the DataVolume in the VM namespace
		if errors.IsNotFound(err) {
			return m.inferDefaultsFromPVC(dvName, vm.Namespace, defaultNameLabel, defaultKindLabel)
		}
		return "", "", err
	}
	// Check the DataVolume for any labels before checking the underlying PVC
	defaultName, defaultKind, err := inferDefaultsFromLabels(dv.Labels, defaultNameLabel, defaultKindLabel)
	if err == nil {
		return defaultName, defaultKind, nil
	}
	return m.inferDefaultsFromDataVolumeSpec(&dv.Spec, defaultNameLabel, defaultKindLabel)
}

func (m *methods) inferDefaultsFromDataVolumeSpec(dataVolumeSpec *v1beta1.DataVolumeSpec, defaultNameLabel, defaultKindLabel string) (string, string, error) {
	if dataVolumeSpec != nil && dataVolumeSpec.Source != nil && dataVolumeSpec.Source.PVC != nil {
		return m.inferDefaultsFromPVC(dataVolumeSpec.Source.PVC.Name, dataVolumeSpec.Source.PVC.Namespace, defaultNameLabel, defaultKindLabel)
	}
	if dataVolumeSpec != nil && dataVolumeSpec.SourceRef != nil {
		return m.inferDefaultsFromDataVolumeSourceRef(dataVolumeSpec.SourceRef, defaultNameLabel, defaultKindLabel)
	}
	return "", "", fmt.Errorf("unable to infer defaults from DataVolumeSpec as DataVolumeSource is not supported")
}

func (m *methods) inferDefaultsFromDataSource(dataSourceName, dataSourceNamespace, defaultNameLabel, defaultKindLabel string) (string, string, error) {
	ds, err := m.clientset.CdiClient().CdiV1beta1().DataSources(dataSourceNamespace).Get(context.Background(), dataSourceName, metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}
	// Check the DataSource for any labels before checking the underlying PVC
	defaultName, defaultKind, err := inferDefaultsFromLabels(ds.Labels, defaultNameLabel, defaultKindLabel)
	if err == nil {
		return defaultName, defaultKind, nil
	}
	if ds.Spec.Source.PVC != nil {
		return m.inferDefaultsFromPVC(ds.Spec.Source.PVC.Name, ds.Spec.Source.PVC.Namespace, defaultNameLabel, defaultKindLabel)
	}
	return "", "", fmt.Errorf("unable to infer defaults from DataSource that doesn't provide DataVolumeSourcePVC")
}

func (m *methods) inferDefaultsFromDataVolumeSourceRef(sourceRef *v1beta1.DataVolumeSourceRef, defaultNameLabel, defaultKindLabel string) (string, string, error) {
	switch sourceRef.Kind {
	case "DataSource":
		return m.inferDefaultsFromDataSource(sourceRef.Name, *sourceRef.Namespace, defaultNameLabel, defaultKindLabel)
	}
	return "", "", fmt.Errorf("unable to infer defaults from DataVolumeSourceRef as Kind %s is not supported", sourceRef.Kind)
}

func applyCpu(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {
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
	preferredTopology := instancetypev1alpha2.PreferSockets
	if preferenceSpec != nil && preferenceSpec.CPU != nil && preferenceSpec.CPU.PreferredCPUTopology != "" {
		preferredTopology = preferenceSpec.CPU.PreferredCPUTopology
	}

	switch preferredTopology {
	case instancetypev1alpha2.PreferCores:
		vmiSpec.Domain.CPU.Cores = instancetypeSpec.CPU.Guest
	case instancetypev1alpha2.PreferSockets:
		vmiSpec.Domain.CPU.Sockets = instancetypeSpec.CPU.Guest
	case instancetypev1alpha2.PreferThreads:
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

func applyMemory(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

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

func applyIOThreadPolicy(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

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

func applyLaunchSecurity(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

	if instancetypeSpec.LaunchSecurity == nil {
		return nil
	}

	if vmiSpec.Domain.LaunchSecurity != nil {
		return Conflicts{field.Child("domain", "launchSecurity")}
	}

	vmiSpec.Domain.LaunchSecurity = instancetypeSpec.LaunchSecurity.DeepCopy()

	return nil
}

func applyGPUs(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

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

func applyHostDevices(field *k8sfield.Path, instancetypeSpec *instancetypev1alpha2.VirtualMachineInstancetypeSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) Conflicts {

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

func ApplyDevicePreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

	if preferenceSpec.Devices == nil {
		return
	}

	// We only want to apply a preference bool when...
	//
	// 1. A preference has actually been provided
	// 2. The user hasn't defined the corresponding attribute already within the VMI
	//
	if preferenceSpec.Devices.PreferredAutoattachGraphicsDevice != nil && vmiSpec.Domain.Devices.AutoattachGraphicsDevice == nil {
		vmiSpec.Domain.Devices.AutoattachGraphicsDevice = pointer.Bool(*preferenceSpec.Devices.PreferredAutoattachGraphicsDevice)
	}

	if preferenceSpec.Devices.PreferredAutoattachMemBalloon != nil && vmiSpec.Domain.Devices.AutoattachMemBalloon == nil {
		vmiSpec.Domain.Devices.AutoattachMemBalloon = pointer.Bool(*preferenceSpec.Devices.PreferredAutoattachMemBalloon)
	}

	if preferenceSpec.Devices.PreferredAutoattachPodInterface != nil && vmiSpec.Domain.Devices.AutoattachPodInterface == nil {
		vmiSpec.Domain.Devices.AutoattachPodInterface = pointer.Bool(*preferenceSpec.Devices.PreferredAutoattachPodInterface)
	}

	if preferenceSpec.Devices.PreferredAutoattachSerialConsole != nil && vmiSpec.Domain.Devices.AutoattachSerialConsole == nil {
		vmiSpec.Domain.Devices.AutoattachSerialConsole = pointer.Bool(*preferenceSpec.Devices.PreferredAutoattachSerialConsole)
	}

	if preferenceSpec.Devices.PreferredUseVirtioTransitional != nil && vmiSpec.Domain.Devices.UseVirtioTransitional == nil {
		vmiSpec.Domain.Devices.UseVirtioTransitional = pointer.Bool(*preferenceSpec.Devices.PreferredUseVirtioTransitional)
	}

	if preferenceSpec.Devices.PreferredBlockMultiQueue != nil && vmiSpec.Domain.Devices.BlockMultiQueue == nil {
		vmiSpec.Domain.Devices.BlockMultiQueue = pointer.Bool(*preferenceSpec.Devices.PreferredBlockMultiQueue)
	}

	if preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue != nil && vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue == nil {
		vmiSpec.Domain.Devices.NetworkInterfaceMultiQueue = pointer.Bool(*preferenceSpec.Devices.PreferredNetworkInterfaceMultiQueue)
	}

	if preferenceSpec.Devices.PreferredAutoattachInputDevice != nil && vmiSpec.Domain.Devices.AutoattachInputDevice == nil {
		vmiSpec.Domain.Devices.AutoattachInputDevice = pointer.Bool(*preferenceSpec.Devices.PreferredAutoattachInputDevice)
	}

	// FIXME DisableHotplug isn't a pointer bool so we don't have a way to tell if a user has actually set it, for now override.
	if preferenceSpec.Devices.PreferredDisableHotplug != nil {
		vmiSpec.Domain.Devices.DisableHotplug = *preferenceSpec.Devices.PreferredDisableHotplug
	}

	if preferenceSpec.Devices.PreferredSoundModel != "" && vmiSpec.Domain.Devices.Sound != nil && vmiSpec.Domain.Devices.Sound.Model == "" {
		vmiSpec.Domain.Devices.Sound.Model = preferenceSpec.Devices.PreferredSoundModel
	}

	if preferenceSpec.Devices.PreferredRng != nil && vmiSpec.Domain.Devices.Rng == nil {
		vmiSpec.Domain.Devices.Rng = preferenceSpec.Devices.PreferredRng.DeepCopy()
	}

	if preferenceSpec.Devices.PreferredTPM != nil && vmiSpec.Domain.Devices.TPM == nil {
		vmiSpec.Domain.Devices.TPM = preferenceSpec.Devices.PreferredTPM.DeepCopy()
	}

	applyDiskPreferences(preferenceSpec, vmiSpec)
	applyInterfacePreferences(preferenceSpec, vmiSpec)
	applyInputPreferences(preferenceSpec, vmiSpec)

}

func applyDiskPreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
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

			if preferenceSpec.Devices.PreferredDiskDedicatedIoThread != nil && vmiDisk.DedicatedIOThread == nil {
				vmiDisk.DedicatedIOThread = pointer.Bool(*preferenceSpec.Devices.PreferredDiskDedicatedIoThread)
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

func applyInterfacePreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
	for ifaceIndex := range vmiSpec.Domain.Devices.Interfaces {
		vmiIface := &vmiSpec.Domain.Devices.Interfaces[ifaceIndex]
		if preferenceSpec.Devices.PreferredInterfaceModel != "" && vmiIface.Model == "" {
			vmiIface.Model = preferenceSpec.Devices.PreferredInterfaceModel
		}
	}
}

func applyInputPreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
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

func applyFeaturePreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

func applyHyperVFeaturePreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

func applyFirmwarePreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

	if preferenceSpec.Firmware.PreferredUseBiosSerial != nil && vmiSpec.Domain.Firmware.Bootloader.BIOS != nil {
		vmiSpec.Domain.Firmware.Bootloader.BIOS.UseSerial = pointer.Bool(*preferenceSpec.Firmware.PreferredUseBiosSerial)
	}

	if preferenceSpec.Firmware.PreferredUseEfi != nil && *preferenceSpec.Firmware.PreferredUseEfi && vmiSpec.Domain.Firmware.Bootloader.EFI == nil && vmiSpec.Domain.Firmware.Bootloader.BIOS == nil {
		vmiSpec.Domain.Firmware.Bootloader.EFI = &v1.EFI{}
	}

	if preferenceSpec.Firmware.PreferredUseSecureBoot != nil && vmiSpec.Domain.Firmware.Bootloader.EFI != nil {
		vmiSpec.Domain.Firmware.Bootloader.EFI.SecureBoot = pointer.Bool(*preferenceSpec.Firmware.PreferredUseSecureBoot)
	}
}

func applyMachinePreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {

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

func applyClockPreferences(preferenceSpec *instancetypev1alpha2.VirtualMachinePreferenceSpec, vmiSpec *virtv1.VirtualMachineInstanceSpec) {
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
