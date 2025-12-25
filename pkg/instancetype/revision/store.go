/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */
//nolint:dupl
package revision

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	virtv1 "kubevirt.io/api/core/v1"
	api "kubevirt.io/api/instancetype"
	"kubevirt.io/api/instancetype/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/instancetype/apply"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	preferenceFind "kubevirt.io/kubevirt/pkg/instancetype/preference/find"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util"
)

func (h *revisionHandler) Store(vm *virtv1.VirtualMachine) error {
	instancetypeStatusRef, err := h.storeInstancetypeRevision(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Failed to store ControllerRevision of VirtualMachineInstancetypeSpec for the Virtualmachine.")
		return err
	}

	preferenceStatusRef, err := h.storePreferenceRevision(vm)
	if err != nil {
		log.Log.Object(vm).Reason(err).Error("Failed to store ControllerRevision of VirtualMachinePreferenceSpec for the Virtualmachine.")
		return err
	}

	return h.patchVM(instancetypeStatusRef, preferenceStatusRef, vm)
}

func (h *revisionHandler) StoreSnapshot(snapshot *snapshotv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) error {
	obj, err := util.GenerateKubeVirtGroupVersionKind(snapshot)
	if err != nil {
		log.Log.Errorf("failed to generate GVK for snapshot: %v", err)
		return err
	}
	snapshot, _ = obj.(*snapshotv1.VirtualMachineSnapshot)

	ownerRef := metav1.OwnerReference{
		APIVersion:         snapshot.GroupVersionKind().GroupVersion().String(),
		Kind:               snapshot.GroupVersionKind().Kind,
		Name:               snapshot.Name,
		UID:                snapshot.UID,
		BlockOwnerDeletion: ptr.To(true),
	}

	if vm.Spec.Instancetype != nil && vm.Status.InstancetypeRef.ControllerRevisionRef.Name != "" {
		err := h.PatchOwnerRef(vm.Namespace, vm.Status.InstancetypeRef.ControllerRevisionRef.Name, &ownerRef)
		if err != nil {
			log.Log.Errorf("Failed to patch InstanceType ControllerRevision: %v", err)
			return err
		}
	}

	if vm.Spec.Preference != nil && vm.Status.PreferenceRef.ControllerRevisionRef.Name != "" {
		err := h.PatchOwnerRef(vm.Namespace, vm.Status.PreferenceRef.ControllerRevisionRef.Name, &ownerRef)
		if err != nil {
			log.Log.Errorf("Failed to patch InstanceType ControllerRevision: %v", err)
			return err
		}
	}

	return h.patchSnapshotContent(snapshot, vm)
}

func (h *revisionHandler) syncStatusWithMatcher(
	vm *virtv1.VirtualMachine,
	matcher virtv1.Matcher,
	statusRef *virtv1.InstancetypeStatusRef,
) error {
	var clearControllerRevisionRef bool
	matcherName := matcher.GetName()
	if matcherName != "" && matcherName != statusRef.Name {
		statusRef.Name = matcherName
		clearControllerRevisionRef = true
	}

	matcherKind := matcher.GetKind()
	if matcherKind != "" && matcherKind != statusRef.Kind {
		statusRef.Kind = matcherKind
		clearControllerRevisionRef = true
	}

	matcherInferFromVolume := matcher.GetInferFromVolume()
	if matcherInferFromVolume != "" && matcherInferFromVolume != statusRef.InferFromVolume {
		statusRef.InferFromVolume = matcherInferFromVolume
		clearControllerRevisionRef = true
	}

	// If the name, kind or inferFromVolume matcher values have changed we need to clear ControllerRevisionRef to either use RevisionName
	// from the matcher or to store a copy of the new resource the matcher is pointing at.
	if clearControllerRevisionRef {
		statusRef.ControllerRevisionRef = nil
	}

	syncInferFromVolumeFailurePolicy(matcher, statusRef)

	matcherRevisionName := matcher.GetRevisionName()
	if matcherRevisionName != "" {
		if statusRef.ControllerRevisionRef == nil || statusRef.ControllerRevisionRef.Name != matcherRevisionName {
			statusRef.ControllerRevisionRef = &virtv1.ControllerRevisionRef{
				Name: matcherRevisionName,
			}
		}
		// Add VM as owner of the existing ControllerRevision (e.g., when restored from snapshot)
		ownerReference := GenerateOwnerReference(vm)
		if err := h.PatchOwnerRef(vm.Namespace, matcherRevisionName, &ownerReference); err != nil {
			log.Log.Object(vm).Reason(err).Errorf("Failed to patch owner reference on ControllerRevision %s", matcherRevisionName)
			return err
		}
	}

	return nil
}

func syncInferFromVolumeFailurePolicy(matcher virtv1.Matcher, statusRef *virtv1.InstancetypeStatusRef) {
	matcherInferFromVolumeFailurePolicy := matcher.GetInferFromVolumeFailurePolicy()
	if matcherInferFromVolumeFailurePolicy != nil {
		if statusRef.InferFromVolumeFailurePolicy == nil || (statusRef.InferFromVolumeFailurePolicy != nil &&
			*matcherInferFromVolumeFailurePolicy != *statusRef.InferFromVolumeFailurePolicy) {
			statusRef.InferFromVolumeFailurePolicy = pointer.P(*matcherInferFromVolumeFailurePolicy)
		}
	}
}

func (h *revisionHandler) storeInstancetypeRevision(vm *virtv1.VirtualMachine) (*virtv1.InstancetypeStatusRef, error) {
	if vm.Spec.Instancetype == nil {
		return nil, nil
	}

	if vm.Status.InstancetypeRef == nil {
		vm.Status.InstancetypeRef = &virtv1.InstancetypeStatusRef{}
	}
	statusRef := vm.Status.InstancetypeRef.DeepCopy()

	if err := h.syncStatusWithMatcher(vm, vm.Spec.Instancetype, statusRef); err != nil {
		return nil, err
	}

	if statusRef.ControllerRevisionRef == nil {
		storedRevision, err := h.createInstancetypeRevision(vm)
		if err != nil {
			return nil, err
		}

		statusRef.ControllerRevisionRef = &virtv1.ControllerRevisionRef{
			Name: storedRevision.Name,
		}
	}

	if equality.Semantic.DeepEqual(vm.Status.InstancetypeRef, statusRef) {
		return nil, nil
	}

	vm.Status.InstancetypeRef = statusRef
	return vm.Status.InstancetypeRef, nil
}

func (h *revisionHandler) createInstancetypeRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	switch strings.ToLower(vm.Spec.Instancetype.Kind) {
	case api.SingularResourceName, api.PluralResourceName:
		instancetype, err := find.NewInstancetypeFinder(h.instancetypeStore, h.virtClient).Find(vm)
		if err != nil {
			return nil, err
		}

		// There is still a window where the instancetype can be updated between the VirtualMachine validation webhook accepting
		// the VirtualMachine and the VirtualMachine controller creating a ControllerRevison. As such we need to check one final
		// time that there are no conflicts when applying the instancetype to the VirtualMachine before continuing.
		if err := h.checkForInstancetypeConflicts(&instancetype.Spec, &vm.Spec.Template.Spec, &vm.Spec.Template.ObjectMeta); err != nil {
			return nil, err
		}
		return h.storeControllerRevision(vm, instancetype)
	case api.ClusterSingularResourceName, api.ClusterPluralResourceName, "":
		clusterInstancetype, err := find.NewClusterInstancetypeFinder(h.clusterInstancetypeStore, h.virtClient).Find(vm)
		if err != nil {
			return nil, err
		}

		// There is still a window where the instancetype can be updated between the VirtualMachine validation webhook accepting
		// the VirtualMachine and the VirtualMachine controller creating a ControllerRevison. As such we need to check one final
		// time that there are no conflicts when applying the instancetype to the VirtualMachine before continuing.
		if err := h.checkForInstancetypeConflicts(
			&clusterInstancetype.Spec,
			&vm.Spec.Template.Spec,
			&vm.Spec.Template.ObjectMeta,
		); err != nil {
			return nil, err
		}
		return h.storeControllerRevision(vm, clusterInstancetype)
	default:
		return nil, fmt.Errorf("got unexpected kind in InstancetypeMatcher: %s", vm.Spec.Instancetype.Kind)
	}
}

func (h *revisionHandler) checkForInstancetypeConflicts(
	instancetypeSpec *v1beta1.VirtualMachineInstancetypeSpec,
	vmiSpec *virtv1.VirtualMachineInstanceSpec,
	vmiMetadata *metav1.ObjectMeta,
) error {
	// Apply the instancetype to a copy of the VMISpec as we don't want to persist any changes here in the VM being passed around
	vmiSpecCopy := vmiSpec.DeepCopy()
	conflicts := apply.NewVMIApplier().ApplyToVMI(field.NewPath("spec", "template", "spec"), instancetypeSpec, nil, vmiSpecCopy, vmiMetadata)
	if len(conflicts) > 0 {
		return conflicts
	}
	return nil
}

func (h *revisionHandler) storePreferenceRevision(vm *virtv1.VirtualMachine) (*virtv1.InstancetypeStatusRef, error) {
	if vm.Spec.Preference == nil {
		return nil, nil
	}

	if vm.Status.PreferenceRef == nil {
		vm.Status.PreferenceRef = &virtv1.InstancetypeStatusRef{}
	}
	statusRef := vm.Status.PreferenceRef.DeepCopy()

	if err := h.syncStatusWithMatcher(vm, vm.Spec.Preference, statusRef); err != nil {
		return nil, err
	}

	if statusRef.ControllerRevisionRef == nil {
		storedRevision, err := h.createPreferenceRevision(vm)
		if err != nil {
			return nil, err
		}

		statusRef.ControllerRevisionRef = &virtv1.ControllerRevisionRef{
			Name: storedRevision.Name,
		}
	}

	if equality.Semantic.DeepEqual(vm.Status.PreferenceRef, statusRef) {
		return nil, nil
	}

	vm.Status.PreferenceRef = statusRef
	return vm.Status.PreferenceRef, nil
}

func (h *revisionHandler) createPreferenceRevision(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	switch strings.ToLower(vm.Spec.Preference.Kind) {
	case api.SingularPreferenceResourceName, api.PluralPreferenceResourceName:
		preference, err := preferenceFind.NewPreferenceFinder(h.preferenceStore, h.virtClient).FindPreference(vm)
		if err != nil {
			return nil, err
		}
		return h.storeControllerRevision(vm, preference)
	case api.ClusterSingularPreferenceResourceName, api.ClusterPluralPreferenceResourceName, "":
		clusterPreference, err := preferenceFind.NewClusterPreferenceFinder(h.clusterPreferenceStore, h.virtClient).FindPreference(vm)
		if err != nil {
			return nil, err
		}
		return h.storeControllerRevision(vm, clusterPreference)
	default:
		return nil, fmt.Errorf("got unexpected kind in PreferenceMatcher: %s", vm.Spec.Preference.Kind)
	}
}

func GenerateName(namespace, resourceName, resourceVersion string, resourceUID types.UID, resourceGeneration int64) string {
	return fmt.Sprintf("%s-%s-%s-%s-%d", namespace, resourceName, resourceVersion, resourceUID, resourceGeneration)
}

func GenerateOwnerReference(vm *virtv1.VirtualMachine) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion:         virtv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               virtv1.VirtualMachineGroupVersionKind.Kind,
		Name:               vm.Name,
		UID:                vm.UID,
		BlockOwnerDeletion: ptr.To(true),
	}
}

func CreateControllerRevision(vm *virtv1.VirtualMachine, object runtime.Object) (*appsv1.ControllerRevision, error) {
	obj, err := util.GenerateKubeVirtGroupVersionKind(object)
	if err != nil {
		return nil, err
	}
	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("unexpected object format returned from GenerateKubeVirtGroupVersionKind")
	}

	revisionName := GenerateName(
		vm.Namespace,
		metaObj.GetName(),
		obj.GetObjectKind().GroupVersionKind().Version,
		metaObj.GetUID(),
		metaObj.GetGeneration(),
	)

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
			OwnerReferences: []metav1.OwnerReference{GenerateOwnerReference(vm)},
			Labels: map[string]string{
				api.ControllerRevisionObjectGenerationLabel: fmt.Sprintf("%d", metaObj.GetGeneration()),
				api.ControllerRevisionObjectKindLabel:       obj.GetObjectKind().GroupVersionKind().Kind,
				api.ControllerRevisionObjectNameLabel:       metaObj.GetName(),
				api.ControllerRevisionObjectUIDLabel:        string(metaObj.GetUID()),
				api.ControllerRevisionObjectVersionLabel:    obj.GetObjectKind().GroupVersionKind().Version,
			},
		},
		Data: runtime.RawExtension{
			Object: obj,
		},
	}, nil
}

func (h *revisionHandler) storeControllerRevision(vm *virtv1.VirtualMachine, object runtime.Object) (*appsv1.ControllerRevision, error) {
	revision, err := CreateControllerRevision(vm, object)
	if err != nil {
		return nil, err
	}

	createdRevision, err := h.virtClient.AppsV1().ControllerRevisions(revision.Namespace).Create(
		context.Background(), revision, metav1.CreateOptions{})
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, fmt.Errorf("failed to create ControllerRevision: %w", err)
		}

		// Grab the existing revision to check the data it contains
		existingRevision, err := h.virtClient.AppsV1().ControllerRevisions(revision.Namespace).Get(
			context.Background(), revision.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get ControllerRevision: %w", err)
		}

		equal, err := Compare(revision, existingRevision)
		if err != nil {
			return nil, err
		}
		if !equal {
			return nil, fmt.Errorf("found existing ControllerRevision with unexpected data: %s", revision.Name)
		}

		ownerReference := GenerateOwnerReference(vm)
		err = h.PatchOwnerRef(existingRevision.Namespace, existingRevision.Name, &ownerReference)
		if err != nil {
			return nil, fmt.Errorf("failed to patch ControllerRevision: %w", err)
		}

		return existingRevision, nil
	}
	return createdRevision, nil
}
