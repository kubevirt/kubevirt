//nolint:lll
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
 * Copyright 2024 Red Hat, Inc.
 *
 */
package instancetype

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

type Upgrader interface {
	Upgrade(vm *virtv1.VirtualMachine) error
}

func (m *InstancetypeMethods) Upgrade(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return nil
	}

	vmPatchSet := patch.New()

	newInstancetypeCR, err := m.upgradeInstancetypeCR(vm, vmPatchSet)
	if err != nil {
		return err
	}

	newPreferenceCR, err := m.upgradePreferenceCR(vm, vmPatchSet)
	if err != nil {
		return err
	}

	if vmPatchSet.IsEmpty() {
		return nil
	}

	patchPayload, err := vmPatchSet.GeneratePayload()
	if err != nil {
		return err
	}

	if _, err := m.Clientset.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{}); err != nil {
		return err
	}

	if newInstancetypeCR != nil {
		if err := m.Clientset.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), vm.Spec.Instancetype.RevisionName, metav1.DeleteOptions{}); err != nil {
			log.Log.Object(vm).Reason(err).Error("ignoring failure to delete ControllerRevision during stashed instance type object upgrade")
		}
		vm.Spec.Instancetype.RevisionName = newInstancetypeCR.Name
	}

	if newPreferenceCR != nil {
		if err := m.Clientset.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), vm.Spec.Preference.RevisionName, metav1.DeleteOptions{}); err != nil {
			log.Log.Object(vm).Reason(err).Error("ignoring failure to delete ControllerRevision during stashed preference object upgrade")
		}
		vm.Spec.Preference.RevisionName = newPreferenceCR.Name
	}

	log.Log.Object(vm).Info("instancetype.kubevirt.io ControllerRevisions upgrade successful")

	return nil
}

func (m *InstancetypeMethods) upgradeInstancetypeCR(vm *virtv1.VirtualMachine, vmPatchSet *patch.PatchSet) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Instancetype == nil || vm.Spec.Instancetype.RevisionName == "" {
		return nil, nil
	}
	return m.upgradeControllerRevision(vm, vm.Spec.Instancetype.RevisionName, "/spec/instancetype/revisionName", vmPatchSet)
}

func (m *InstancetypeMethods) upgradePreferenceCR(vm *virtv1.VirtualMachine, vmPatchSet *patch.PatchSet) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Preference == nil || vm.Spec.Preference.RevisionName == "" {
		return nil, nil
	}
	return m.upgradeControllerRevision(vm, vm.Spec.Preference.RevisionName, "/spec/preference/revisionName", vmPatchSet)
}

func (m *InstancetypeMethods) upgradeControllerRevision(vm *virtv1.VirtualMachine, crName, jsonPath string, vmPatchSet *patch.PatchSet) (*appsv1.ControllerRevision, error) {
	// We always have an informer in this codepath so use getControllerRevisionByInformer
	original, err := m.getControllerRevisionByInformer(types.NamespacedName{Namespace: vm.Namespace, Name: crName})
	if err != nil {
		return nil, err
	}

	// If the CR is already labeled with the latest version then skip
	if IsObjectLatestVersion(original) {
		return nil, nil
	}

	log.Log.Object(vm).Infof("upgrading instancetype.kubevirt.io ControllerRevision %s (%s)", crName, jsonPath)

	// Upgrade the stashed object to the latest version
	err = decodeControllerRevision(original)
	if err != nil {
		return nil, err
	}

	newCR, err := CreateControllerRevision(vm, original.Data.Object)
	if err != nil {
		return nil, err
	}

	// Recreate the CR with the now upgraded runtime.Object
	newCR, err = m.Clientset.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), newCR, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// Add the patches to the VM patchset
	vmPatchSet.AddOption(
		patch.WithTest(jsonPath, original.Name),
		patch.WithReplace(jsonPath, newCR.Name),
	)

	return newCR, nil
}

func IsObjectLatestVersion(cr *appsv1.ControllerRevision) bool {
	if version, ok := cr.GetLabels()[instancetypeapi.ControllerRevisionObjectVersionLabel]; ok {
		return version == instancetypeapi.LatestVersion
	}
	return false
}
