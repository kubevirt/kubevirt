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

package upgrade

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

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

	oldInstancetypeCRName := ""
	if revision.HasControllerRevisionRef(vm.Status.InstancetypeRef) {
		oldInstancetypeCRName = vm.Status.InstancetypeRef.ControllerRevisionRef.Name
	}

	oldPreferenceCRName := ""
	if revision.HasControllerRevisionRef(vm.Status.PreferenceRef) {
		oldPreferenceCRName = vm.Status.PreferenceRef.ControllerRevisionRef.Name
	}

	newInstancetypeCR, err := u.upgradeInstancetypeCR(vm)
	if err != nil {
		return err
	}

	newPreferenceCR, err := u.upgradePreferenceCR(vm)
	if err != nil {
		return err
	}

	if newInstancetypeCR == nil && newPreferenceCR == nil {
		return nil
	}

	u.updateStatusRefs(vm, newInstancetypeCR, newPreferenceCR)

	// Delete old ControllerRevisions only after status refs have been updated
	// locally to point to the new CRs. If the caller fails to persist the
	// status update and Upgrade is called again, upgradeControllerRevision
	// handles the retry by tolerating AlreadyExists on Create and NotFound
	// on the old CR cleanup.
	u.cleanupOldControllerRevisions(vm, newInstancetypeCR, oldInstancetypeCRName, newPreferenceCR, oldPreferenceCRName)

	log.Log.Object(vm).Info("instancetype.kubevirt.io ControllerRevisions upgrade successful")

	return nil
}

func (u *upgrader) updateStatusRefs(vm *virtv1.VirtualMachine, newInstancetypeCR, newPreferenceCR *appsv1.ControllerRevision) {
	if newInstancetypeCR != nil {
		vm.Status.InstancetypeRef.ControllerRevisionRef.Name = newInstancetypeCR.Name
	}

	if newPreferenceCR != nil {
		vm.Status.PreferenceRef.ControllerRevisionRef.Name = newPreferenceCR.Name
	}
}

func (u *upgrader) cleanupOldControllerRevisions(
	vm *virtv1.VirtualMachine,
	newInstancetypeCR *appsv1.ControllerRevision, oldInstancetypeCRName string,
	newPreferenceCR *appsv1.ControllerRevision, oldPreferenceCRName string,
) {
	if newInstancetypeCR != nil && oldInstancetypeCRName != "" {
		if err := u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Delete(
			context.Background(), oldInstancetypeCRName, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
			log.Log.Object(vm).Reason(err).Error("ignoring failure to delete ControllerRevision during stashed instance type object upgrade")
		}
	}

	if newPreferenceCR != nil && oldPreferenceCRName != "" {
		if err := u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Delete(
			context.Background(), oldPreferenceCRName, metav1.DeleteOptions{}); err != nil && !k8serrors.IsNotFound(err) {
			log.Log.Object(vm).Reason(err).Error("ignoring failure to delete ControllerRevision during stashed preference object upgrade")
		}
	}
}

var instancetypeKinds = map[string]bool{
	"VirtualMachineInstancetype":        true,
	"VirtualMachineClusterInstancetype": true,
}

var preferenceKinds = map[string]bool{
	"VirtualMachinePreference":        true,
	"VirtualMachineClusterPreference": true,
}

func (u *upgrader) upgradeInstancetypeCR(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Instancetype == nil || !revision.HasControllerRevisionRef(vm.Status.InstancetypeRef) {
		return nil, nil
	}
	return u.upgradeControllerRevision(vm, vm.Status.InstancetypeRef.ControllerRevisionRef.Name, instancetypeKinds)
}

func (u *upgrader) upgradePreferenceCR(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Preference == nil || !revision.HasControllerRevisionRef(vm.Status.PreferenceRef) {
		return nil, nil
	}
	return u.upgradeControllerRevision(vm, vm.Status.PreferenceRef.ControllerRevisionRef.Name, preferenceKinds)
}

func (u *upgrader) upgradeControllerRevision(
	vm *virtv1.VirtualMachine,
	crName string,
	validKinds map[string]bool,
) (*appsv1.ControllerRevision, error) {
	original, err := u.controllerRevisionFinder.Find(types.NamespacedName{Namespace: vm.Namespace, Name: crName})
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
		// The old CR was already deleted by a previous upgrade attempt
		// whose status update was not persisted. Look up the new CR that
		// was already created so the status refs can be updated.
		return u.findUpgradedControllerRevision(vm, validKinds)
	}

	if IsObjectLatestVersion(original) {
		return nil, nil
	}

	log.Log.Object(vm).Infof("upgrading instancetype.kubevirt.io ControllerRevision %s", crName)

	upgradedCR := original.DeepCopy()
	err = compatibility.Decode(upgradedCR)
	if err != nil {
		return nil, err
	}

	newCR, err := revision.CreateControllerRevision(vm, upgradedCR.Data.Object)
	if err != nil {
		return nil, err
	}

	newCR, err = u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), newCR, metav1.CreateOptions{})
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			return nil, err
		}
		newCR, err = u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
			context.Background(), newCR.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	return newCR, nil
}

func (u *upgrader) findUpgradedControllerRevision(
	vm *virtv1.VirtualMachine, validKinds map[string]bool,
) (*appsv1.ControllerRevision, error) {
	log.Log.Object(vm).Info("looking for previously upgraded ControllerRevision")

	crList, err := u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).List(
		context.Background(), metav1.ListOptions{
			LabelSelector: instancetypeapi.ControllerRevisionObjectVersionLabel + "=" + instancetypeapi.LatestVersion,
		})
	if err != nil {
		return nil, err
	}

	for i := range crList.Items {
		cr := &crList.Items[i]
		if !metav1.IsControlledBy(cr, vm) {
			continue
		}
		if kind, ok := cr.Labels[instancetypeapi.ControllerRevisionObjectKindLabel]; ok && validKinds[kind] {
			return cr, nil
		}
	}

	return nil, fmt.Errorf("ControllerRevision not found and no upgraded replacement exists for VM %s/%s", vm.Namespace, vm.Name)
}

func IsObjectLatestVersion(cr *appsv1.ControllerRevision) bool {
	if version, ok := cr.GetLabels()[instancetypeapi.ControllerRevisionObjectVersionLabel]; ok {
		return version == instancetypeapi.LatestVersion
	}
	return false
}
