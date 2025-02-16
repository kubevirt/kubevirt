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
 * Copyright The KubeVirt Authors
 *
 */

package upgrade

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/instancetype/compatibility"
	"kubevirt.io/kubevirt/pkg/instancetype/find"
	"kubevirt.io/kubevirt/pkg/instancetype/revision"
)

type controllerRevisionFinder interface {
	Find(types.NamespacedName) (*appsv1.ControllerRevision, error)
}

type upgrader struct {
	controllerRevisionFinder controllerRevisionFinder
	virtClient               kubecli.KubevirtClient
}

func New(store cache.Store, virtClient kubecli.KubevirtClient) *upgrader {
	return &upgrader{
		controllerRevisionFinder: find.NewControllerRevisionFinder(store, virtClient),
		virtClient:               virtClient,
	}
}

func (u *upgrader) Upgrade(vm *virtv1.VirtualMachine) error {
	if vm.Spec.Instancetype == nil && vm.Spec.Preference == nil {
		return nil
	}

	vmPatchSet := patch.New()

	newInstancetypeCR, err := u.upgradeInstancetypeCR(vm, vmPatchSet)
	if err != nil {
		return err
	}

	newPreferenceCR, err := u.upgradePreferenceCR(vm, vmPatchSet)
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

	if _, err := u.virtClient.VirtualMachine(vm.Namespace).PatchStatus(
		context.Background(), vm.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{}); err != nil {
		return err
	}

	if newInstancetypeCR != nil {
		if err := u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Delete(
			context.Background(), vm.Status.InstancetypeRef.ControllerRevisionRef.Name, metav1.DeleteOptions{}); err != nil {
			log.Log.Object(vm).Reason(err).Error("ignoring failure to delete ControllerRevision during stashed instance type object upgrade")
		}
		vm.Status.InstancetypeRef.ControllerRevisionRef.Name = newInstancetypeCR.Name
	}

	if newPreferenceCR != nil {
		if err := u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Delete(
			context.Background(), vm.Status.PreferenceRef.ControllerRevisionRef.Name, metav1.DeleteOptions{}); err != nil {
			log.Log.Object(vm).Reason(err).Error("ignoring failure to delete ControllerRevision during stashed preference object upgrade")
		}
		vm.Status.PreferenceRef.ControllerRevisionRef.Name = newPreferenceCR.Name
	}

	log.Log.Object(vm).Info("instancetype.kubevirt.io ControllerRevisions upgrade successful")

	return nil
}

func (u *upgrader) upgradeInstancetypeCR(vm *virtv1.VirtualMachine, vmPatchSet *patch.PatchSet) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Instancetype == nil || !revision.HasControllerRevisionRef(vm.Status.InstancetypeRef) {
		return nil, nil
	}
	return u.upgradeControllerRevision(
		vm, vm.Status.InstancetypeRef.ControllerRevisionRef.Name, "/status/instancetypeRef/controllerRevisionRef/name", vmPatchSet)
}

func (u *upgrader) upgradePreferenceCR(vm *virtv1.VirtualMachine, vmPatchSet *patch.PatchSet) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Preference == nil || !revision.HasControllerRevisionRef(vm.Status.PreferenceRef) {
		return nil, nil
	}
	return u.upgradeControllerRevision(
		vm, vm.Status.PreferenceRef.ControllerRevisionRef.Name, "/status/preferenceRef/controllerRevisionRef/name", vmPatchSet)
}

func (u *upgrader) upgradeControllerRevision(
	vm *virtv1.VirtualMachine,
	crName, jsonPath string,
	vmPatchSet *patch.PatchSet,
) (*appsv1.ControllerRevision, error) {
	original, err := u.controllerRevisionFinder.Find(types.NamespacedName{Namespace: vm.Namespace, Name: crName})
	if err != nil {
		return nil, err
	}

	// If the CR is already labeled with the latest version then skip
	if IsObjectLatestVersion(original) {
		return nil, nil
	}

	log.Log.Object(vm).Infof("upgrading instancetype.kubevirt.io ControllerRevision %s (%s)", crName, jsonPath)

	upgradedCR := original.DeepCopy()
	// Upgrade the stashed object to the latest version
	err = compatibility.Decode(upgradedCR)
	if err != nil {
		return nil, err
	}

	newCR, err := revision.CreateControllerRevision(vm, upgradedCR.Data.Object)
	if err != nil {
		return nil, err
	}

	// Recreate the CR with the now upgraded runtime.Object
	newCR, err = u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), newCR, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// Add the patches to the VM patchset
	vmPatchSet.AddOption(
		patch.WithTest(jsonPath, upgradedCR.Name),
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
