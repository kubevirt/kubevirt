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
	"k8s.io/apimachinery/pkg/api/equality"
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

func (u *upgrader) upgradeInstancetypeCR(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Instancetype == nil || !revision.HasControllerRevisionRef(vm.Status.InstancetypeRef) {
		return nil, nil
	}
	return u.upgradeControllerRevision(vm, vm.Status.InstancetypeRef.ControllerRevisionRef.Name)
}

func (u *upgrader) upgradePreferenceCR(vm *virtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	if vm.Spec.Preference == nil || !revision.HasControllerRevisionRef(vm.Status.PreferenceRef) {
		return nil, nil
	}
	return u.upgradeControllerRevision(vm, vm.Status.PreferenceRef.ControllerRevisionRef.Name)
}

func (u *upgrader) upgradeControllerRevision(
	vm *virtv1.VirtualMachine,
	crName string,
) (*appsv1.ControllerRevision, error) {
	original, err := u.controllerRevisionFinder.Find(types.NamespacedName{Namespace: vm.Namespace, Name: crName})
	if err != nil {
		return nil, err
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

	createdCR, err := u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), newCR, metav1.CreateOptions{})
	if err == nil {
		return createdCR, nil
	}
	if !k8serrors.IsAlreadyExists(err) {
		return nil, err
	}

	// A previous reconcile may have created the replacement without persisting
	// its status reference. Use the deterministic name derived from the original
	// snapshot, and verify the existing revision before adopting it.
	existingCR, err := u.virtClient.AppsV1().ControllerRevisions(vm.Namespace).Get(
		context.Background(), newCR.Name, metav1.GetOptions{},
	)
	if err != nil {
		return nil, err
	}
	if !metav1.IsControlledBy(existingCR, vm) || !equality.Semantic.DeepEqual(newCR.Labels, existingCR.Labels) {
		return nil, fmt.Errorf("existing ControllerRevision %s does not match the expected upgrade", newCR.Name)
	}
	equal, err := revision.Compare(newCR, existingCR)
	if err != nil {
		return nil, err
	}
	if !equal {
		return nil, fmt.Errorf("existing ControllerRevision %s contains unexpected data", newCR.Name)
	}

	return existingCR, nil
}

func IsObjectLatestVersion(cr *appsv1.ControllerRevision) bool {
	if version, ok := cr.GetLabels()[instancetypeapi.ControllerRevisionObjectVersionLabel]; ok {
		return version == instancetypeapi.LatestVersion
	}
	return false
}
