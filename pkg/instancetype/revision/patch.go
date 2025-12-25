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
package revision

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	virtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

func (h *revisionHandler) patchVM(
	instancetypeStatusRef, preferenceStatusRef *virtv1.InstancetypeStatusRef,
	vm *virtv1.VirtualMachine,
) error {
	// Batch any writes to the VirtualMachine into a single PatchStatus() call to avoid races in the controller.
	logger := func() *log.FilteredLogger { return log.Log.Object(vm) }
	revisionPatch, err := GeneratePatch(instancetypeStatusRef, preferenceStatusRef)
	if err != nil || len(revisionPatch) == 0 {
		return err
	}
	if _, err := h.virtClient.VirtualMachine(vm.Namespace).PatchStatus(
		context.Background(), vm.Name, types.JSONPatchType, revisionPatch, metav1.PatchOptions{},
	); err != nil {
		logger().Reason(err).Error("Failed to update VirtualMachine with instancetype and preference ControllerRevision references.")
		return err
	}
	return nil
}

func (h *revisionHandler) PatchOwnerRef(namespace, revisionName string, ownerRef *metav1.OwnerReference) error {
	log.Log.Infof("Adding owner ref %+v to revision: %s", *ownerRef, revisionName)

	patchOps := []map[string]interface{}{
		{
			"op":    "add",
			"path":  "/metadata/ownerReferences/-",
			"value": ownerRef,
		},
	}

	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		log.Log.Errorf("failed to marshal patch for owner ref %+v to revision: %s", *ownerRef, revisionName)
		return err
	}

	_, err = h.virtClient.AppsV1().
		ControllerRevisions(namespace).
		Patch(context.Background(), revisionName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		log.Log.Errorf("failed to patch owner ref %+v to revision: %s", *ownerRef, revisionName)
		return err
	}

	return nil
}

func GeneratePatch(instancetypeStatusRef, preferenceStatusRef *virtv1.InstancetypeStatusRef) ([]byte, error) {
	patchSet := patch.New()
	if instancetypeStatusRef != nil {
		patchSet.AddOption(
			patch.WithAdd("/status/instancetypeRef", instancetypeStatusRef),
		)
	}

	if preferenceStatusRef != nil {
		patchSet.AddOption(
			patch.WithAdd("/status/preferenceRef", preferenceStatusRef),
		)
	}

	if patchSet.IsEmpty() {
		return nil, nil
	}

	return patchSet.GeneratePayload()
}

func (h *revisionHandler) patchSnapshotContent(snapshot *snapshotv1.VirtualMachineSnapshot, vm *virtv1.VirtualMachine) error {
	if snapshot.Status == nil || snapshot.Status.VirtualMachineSnapshotContentName == nil {
		log.Log.Infof("Snapshot %s has no content yet, skipping content patch", snapshot.Name)
		return nil
	}

	contentName := *snapshot.Status.VirtualMachineSnapshotContentName
	var patchOps []map[string]interface{}

	// Patch instancetype revision name if present
	if vm.Spec.Instancetype != nil && vm.Status.InstancetypeRef != nil && vm.Status.InstancetypeRef.ControllerRevisionRef != nil {
		revisionName := vm.Status.InstancetypeRef.ControllerRevisionRef.Name
		if revisionName != "" {
			patchOps = append(patchOps, map[string]interface{}{
				"op":    "add",
				"path":  "/spec/source/virtualMachine/spec/instancetype/revisionName",
				"value": revisionName,
			})
		}
	}

	// Patch preference revision name if present
	if vm.Spec.Preference != nil && vm.Status.PreferenceRef != nil && vm.Status.PreferenceRef.ControllerRevisionRef != nil {
		revisionName := vm.Status.PreferenceRef.ControllerRevisionRef.Name
		if revisionName != "" {
			patchOps = append(patchOps, map[string]interface{}{
				"op":    "add",
				"path":  "/spec/source/virtualMachine/spec/preference/revisionName",
				"value": revisionName,
			})
		}
	}

	// If no patches are needed, return early
	if len(patchOps) == 0 {
		return nil
	}

	patchBytes, err := json.Marshal(patchOps)
	if err != nil {
		log.Log.Errorf("failed to marshal patch for snapshot content %s: %v", contentName, err)
		return err
	}

	log.Log.Infof("Patching snapshot content %s with revision names", contentName)
	_, err = h.virtClient.VirtualMachineSnapshotContent(snapshot.Namespace).
		Patch(context.Background(), contentName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		log.Log.Errorf("failed to patch snapshot content %s: %v", contentName, err)
		return err
	}

	return nil
}
