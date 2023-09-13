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
 * Copyright 2023 Red Hat, Inc.
 *
 */
package instancetype

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
	instancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"
	instancetypev1alpha2 "kubevirt.io/api/instancetype/v1alpha2"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	generatedscheme "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/scheme"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
)

type UpgraderInterface interface {
	Upgrade(*appsv1.ControllerRevision) (*appsv1.ControllerRevision, error)
}

var _ UpgraderInterface = &Upgrader{}

type Upgrader struct {
	client     kubecli.KubevirtClient
	vmInformer cache.SharedIndexInformer
}

func NewUpgrader(client kubecli.KubevirtClient, vmInformer cache.SharedIndexInformer) *Upgrader {
	return &Upgrader{
		client:     client,
		vmInformer: vmInformer,
	}
}

func (u *Upgrader) Upgrade(original *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	alreadyLatest, err := isAlreadyLatestVersion(original)
	if err != nil {
		return nil, err
	}
	if alreadyLatest {
		return original, nil
	}

	// Find the VM owner of the CR
	vm, err := u.discoverOwner(original)
	if err != nil {
		return nil, err
	}

	// Upgrade the CR object and create a new CR
	newCR, err := u.upgradeControllerRevision(vm, original)
	if err != nil {
		return nil, err
	}

	// Update the VM Owner to reference the new CR
	if err := u.patchVirtualMachine(vm, original.Name, newCR.Name); err != nil {
		return nil, err
	}

	// Delete the original CR
	if err := u.client.AppsV1().ControllerRevisions(original.Namespace).Delete(context.Background(), original.Name, metav1.DeleteOptions{}); err != nil {
		return nil, err
	}

	return newCR, nil
}

func (u *Upgrader) upgradeControllerRevision(vm *virtv1.VirtualMachine, original *appsv1.ControllerRevision) (*appsv1.ControllerRevision, error) {
	// Upgrade the stashed object to the latest version
	if err := decodeControllerRevisionObject(original); err != nil {
		return nil, err
	}

	// Recreate the CR with the now upgraded runtime.Object
	newCR, err := CreateControllerRevision(vm, original.Data.Object)
	if err != nil {
		return nil, err
	}

	newCR, err = u.client.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), newCR, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	return newCR, nil
}

func (u *Upgrader) patchVirtualMachine(vm *virtv1.VirtualMachine, originalCRName, newName string) error {
	var (
		payload []byte
		err     error
	)

	if vm.Spec.Instancetype != nil && vm.Spec.Instancetype.RevisionName == originalCRName {
		payload, err = patch.GenerateTestReplacePatch("/spec/instancetype/revisionName", originalCRName, newName)
		if err != nil {
			return fmt.Errorf("failed to generate instancetype revisionName patch payload: %w", err)
		}
	}

	if vm.Spec.Preference != nil && vm.Spec.Preference.RevisionName == originalCRName {
		payload, err = patch.GenerateTestReplacePatch("/spec/preference/revisionName", originalCRName, newName)
		if err != nil {
			return fmt.Errorf("failed to generate preference revisionName patch payload: %w", err)
		}
	}

	// Ultimately we only care about upgrading the object within the CR, so if
	// the VM somehow no longer references the CR then skip patching it
	if payload != nil {
		if _, err := u.client.VirtualMachine(vm.Namespace).Patch(context.Background(), vm.Name, types.JSONPatchType, payload, &metav1.PatchOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (u *Upgrader) discoverOwner(cr *appsv1.ControllerRevision) (*virtv1.VirtualMachine, error) {
	if len(cr.OwnerReferences) < 1 || cr.OwnerReferences[0].Kind != "VirtualMachine" {
		return nil, fmt.Errorf("unable to determine VirtualMachine owner of ControllerRevision")
	}
	vmKey := fmt.Sprintf("%s/%s", cr.Namespace, cr.OwnerReferences[0].Name)
	obj, exists, err := u.vmInformer.GetStore().GetByKey(vmKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("unable to find ControllerRevision %s owner VirtualMachine %s", cr.Name, vmKey)
	}
	vm, ok := obj.(*virtv1.VirtualMachine)
	if !ok {
		return nil, fmt.Errorf("unknown object found in ControllerRevision informer")
	}
	return vm, nil
}

func isAlreadyLatestVersion(cr *appsv1.ControllerRevision) (bool, error) {
	originalObjectVersion, err := discoverObjectVersion(cr)
	if err != nil {
		return false, err
	}
	return originalObjectVersion == instancetypeapi.LatestVersion, nil
}

func DiscoverObjectVersionByLabel(cr *appsv1.ControllerRevision) (string, error) {
	if version, ok := cr.GetLabels()[instancetypeapi.ControllerRevisionObjectVersionLabel]; ok {
		return version, nil
	}
	return "", fmt.Errorf("unable to find object version label %s", instancetypeapi.ControllerRevisionObjectVersionLabel)
}

func DiscoverObjectVersionByDecode(cr *appsv1.ControllerRevision) (string, error) {
	// TODO(lyarwood) We need to handle VirtualMachine{Instancetype,Preference}SpecRevision but this requires
	// isPreference to be dropped from the compat code as we can't tell what we are dealing with here ahead of time

	// FIXME(lyarwood) This can be removed eventually once all
	// ControllerRevisions are labeled sometime after >=v1.2.0?
	decodedObj, err := runtime.Decode(generatedscheme.Codecs.UniversalDeserializer(), cr.Data.Raw)
	if err != nil {
		return "", fmt.Errorf("failed to decode object in ControllerRevision: %w", err)
	}
	switch decodedObj.(type) {
	case *instancetypev1beta1.VirtualMachineInstancetype, *instancetypev1beta1.VirtualMachineClusterInstancetype, *instancetypev1beta1.VirtualMachinePreference, *instancetypev1beta1.VirtualMachineClusterPreference:
		return instancetypev1beta1.SchemeGroupVersion.Version, nil
	case *instancetypev1alpha2.VirtualMachineInstancetype, *instancetypev1alpha2.VirtualMachineClusterInstancetype, *instancetypev1alpha2.VirtualMachinePreference, *instancetypev1alpha2.VirtualMachineClusterPreference:
		return instancetypev1alpha2.SchemeGroupVersion.Version, nil
	case *instancetypev1alpha1.VirtualMachineInstancetype, *instancetypev1alpha1.VirtualMachineClusterInstancetype, *instancetypev1alpha1.VirtualMachinePreference, *instancetypev1alpha1.VirtualMachineClusterPreference:
		return instancetypev1alpha1.SchemeGroupVersion.Version, nil
	}
	return "", fmt.Errorf("unexpected object type in ControllerRevision")
}

func discoverObjectVersion(cr *appsv1.ControllerRevision) (string, error) {
	// We started labeling instance type and preference ControllerRevisions in
	// KubeVirt v1.0.0 but can't expect them to always be labeled yet
	version, err := DiscoverObjectVersionByLabel(cr)
	if err == nil {
		return version, nil
	}
	return DiscoverObjectVersionByDecode(cr)
}
