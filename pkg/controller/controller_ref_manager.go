/*
Copyright 2016 The Kubernetes Authors.
Copyright 2017 The KubeVirt Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Taken from https://github.com/kubernetes/kubernetes/blob/b28a83a4cf779189d72a87e847441888e7918e5d/pkg/controller/controller_ref_manager.go
and adapted for KubeVirt.
*/

package controller

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/datavolumecontroller/v1alpha1"
	virtv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
)

type BaseControllerRefManager struct {
	Controller metav1.Object
	Selector   labels.Selector

	canAdoptErr  error
	canAdoptOnce sync.Once
	CanAdoptFunc func() error
}

func (m *BaseControllerRefManager) CanAdopt() error {
	m.canAdoptOnce.Do(func() {
		if m.CanAdoptFunc != nil {
			m.canAdoptErr = m.CanAdoptFunc()
		}
	})
	return m.canAdoptErr
}

// ClaimObject tries to take ownership of an object for this controller.
//
// It will reconcile the following:
//   * Adopt orphans if the match function returns true.
//   * Release owned objects if the match function returns false.
//
// A non-nil error is returned if some form of reconciliation was attempted and
// failed. Usually, controllers should try again later in case reconciliation
// is still needed.
//
// If the error is nil, either the reconciliation succeeded, or no
// reconciliation was necessary. The returned boolean indicates whether you now
// own the object.
//
// No reconciliation will be attempted if the controller is being deleted.
func (m *BaseControllerRefManager) ClaimObject(obj metav1.Object, match func(metav1.Object) bool, adopt, release func(metav1.Object) error) (bool, error) {
	controllerRef := metav1.GetControllerOf(obj)
	if controllerRef != nil {
		if controllerRef.UID != m.Controller.GetUID() {
			// Owned by someone else. Ignore.
			return false, nil
		}
		if match(obj) {
			// We already own it and the selector matches.
			// Return true (successfully claimed) before checking deletion timestamp.
			// We're still allowed to claim things we already own while being deleted
			// because doing so requires taking no actions.
			return true, nil
		}
		// Owned by us but selector doesn't match.
		// Try to release, unless we're being deleted.
		if m.Controller.GetDeletionTimestamp() != nil {
			return false, nil
		}
		if err := release(obj); err != nil {
			// If the object no longer exists, ignore the error.
			if errors.IsNotFound(err) {
				return false, nil
			}
			// Either someone else released it, or there was a transient error.
			// The controller should requeue and try again if it's still stale.
			return false, err
		}
		// Successfully released.
		return false, nil
	}

	// It's an orphan.
	if m.Controller.GetDeletionTimestamp() != nil || !match(obj) {
		// Ignore if we're being deleted or selector doesn't match.
		return false, nil
	}
	if obj.GetDeletionTimestamp() != nil {
		// Ignore if the object is being deleted
		return false, nil
	}
	// Selector matches. Try to adopt.
	if err := adopt(obj); err != nil {
		// If the object no longer exists, ignore the error.
		if errors.IsNotFound(err) {
			return false, nil
		}
		// Either someone else claimed it first, or there was a transient error.
		// The controller should requeue and try again if it's still orphaned.
		return false, err
	}
	// Successfully adopted.
	return true, nil
}

type VirtualMachineControllerRefManager struct {
	BaseControllerRefManager
	controllerKind        schema.GroupVersionKind
	virtualMachineControl VirtualMachineControlInterface
}

// NewVirtualMachineControllerRefManager returns a VirtualMachineControllerRefManager that exposes
// methods to manage the controllerRef of virtual machines.
//
// The CanAdopt() function can be used to perform a potentially expensive check
// (such as a live GET from the API server) prior to the first adoption.
// It will only be called (at most once) if an adoption is actually attempted.
// If CanAdopt() returns a non-nil error, all adoptions will fail.
//
// NOTE: Once CanAdopt() is called, it will not be called again by the same
//       VirtualMachineControllerRefManager instance. Create a new instance if it makes
//       sense to check CanAdopt() again (e.g. in a different sync pass).
func NewVirtualMachineControllerRefManager(
	virtualMachineControl VirtualMachineControlInterface,
	controller metav1.Object,
	selector labels.Selector,
	controllerKind schema.GroupVersionKind,
	canAdopt func() error,
) *VirtualMachineControllerRefManager {
	return &VirtualMachineControllerRefManager{
		BaseControllerRefManager: BaseControllerRefManager{
			Controller:   controller,
			Selector:     selector,
			CanAdoptFunc: canAdopt,
		},
		controllerKind:        controllerKind,
		virtualMachineControl: virtualMachineControl,
	}
}

// ClaimVirtualMachines tries to take ownership of a list of VirtualMachines.
//
// It will reconcile the following:
//   * Adopt orphans if the selector matches.
//   * Release owned objects if the selector no longer matches.
//
// Optional: If one or more filters are specified, a VirtualMachineInstance will only be claimed if
// all filters return true.
//
// A non-nil error is returned if some form of reconciliation was attempted and
// failed. Usually, controllers should try again later in case reconciliation
// is still needed.
//
// If the error is nil, either the reconciliation succeeded, or no
// reconciliation was necessary. The list of VirtualMachines that you now own is returned.
func (m *VirtualMachineControllerRefManager) ClaimVirtualMachines(vmis []*virtv1.VirtualMachineInstance, filters ...func(machine *virtv1.VirtualMachineInstance) bool) ([]*virtv1.VirtualMachineInstance, error) {
	var claimed []*virtv1.VirtualMachineInstance
	var errlist []error

	match := func(obj metav1.Object) bool {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		// Check selector first so filters only run on potentially matching VirtualMachines.
		if !m.Selector.Matches(labels.Set(vmi.Labels)) {
			return false
		}
		for _, filter := range filters {
			if !filter(vmi) {
				return false
			}
		}
		return true
	}
	adopt := func(obj metav1.Object) error {
		return m.AdoptVirtualMachine(obj.(*virtv1.VirtualMachineInstance))
	}
	release := func(obj metav1.Object) error {
		return m.ReleaseVirtualMachine(obj.(*virtv1.VirtualMachineInstance))
	}

	for _, vmi := range vmis {
		ok, err := m.ClaimObject(vmi, match, adopt, release)
		if err != nil {
			errlist = append(errlist, err)
			continue
		}
		if ok {
			claimed = append(claimed, vmi)
		}
	}
	return claimed, utilerrors.NewAggregate(errlist)
}

// ClaimDataVolume tries to take ownership of a list of DataVolumes.
//
// It will reconcile the following:
//   * Adopt orphans if the selector matches.
//   * Release owned objects if the selector no longer matches.
//
// Optional: If one or more filters are specified, a DataVolume will only be claimed if
// all filters return true.
//
// A non-nil error is returned if some form of reconciliation was attempted and
// failed. Usually, controllers should try again later in case reconciliation
// is still needed.
//
// If the error is nil, either the reconciliation succeeded, or no
// reconciliation was necessary. The list of DataVolumes that you now own is returned.
func (m *VirtualMachineControllerRefManager) ClaimMatchedDataVolumes(dataVolumes []*cdiv1.DataVolume, filters ...func(machine *cdiv1.DataVolume) bool) ([]*cdiv1.DataVolume, error) {
	var claimed []*cdiv1.DataVolume
	var errlist []error

	match := func(obj metav1.Object) bool {
		return true

	}
	adopt := func(obj metav1.Object) error {
		return m.AdoptDataVolume(obj.(*cdiv1.DataVolume))
	}
	release := func(obj metav1.Object) error {
		return m.ReleaseDataVolume(obj.(*cdiv1.DataVolume))
	}

	for _, dataVolume := range dataVolumes {
		ok, err := m.ClaimObject(dataVolume, match, adopt, release)
		if err != nil {
			errlist = append(errlist, err)
			continue
		}
		if ok {
			claimed = append(claimed, dataVolume)
		}
	}
	return claimed, utilerrors.NewAggregate(errlist)
}

// ClaimVirtualMachineByName tries to take ownership of a VirtualMachineInstance.
//
// It will reconcile the following:
//   * Adopt orphans if the selector matches.
//   * Release owned objects if the selector no longer matches.
//
// Optional: If one or more filters are specified, a VirtualMachineInstance will only be claimed if
// all filters return true.
//
// A non-nil error is returned if some form of reconciliation was attempted and
// failed. Usually, controllers should try again later in case reconciliation
// is still needed.
//
// If the error is nil, either the reconciliation succeeded, or no
// reconciliation was necessary. The list of VirtualMachines that you now own is returned.
func (m *VirtualMachineControllerRefManager) ClaimVirtualMachineByName(vmi *virtv1.VirtualMachineInstance, filters ...func(machine *virtv1.VirtualMachineInstance) bool) (*virtv1.VirtualMachineInstance, error) {
	match := func(obj metav1.Object) bool {
		vmi := obj.(*virtv1.VirtualMachineInstance)
		// Check selector first so filters only run on potentially matching VirtualMachines.
		if m.Controller.GetName() != vmi.Name {
			return false
		}
		for _, filter := range filters {
			if !filter(vmi) {
				return false
			}
		}
		return true
	}
	adopt := func(obj metav1.Object) error {
		return m.AdoptVirtualMachine(obj.(*virtv1.VirtualMachineInstance))
	}
	release := func(obj metav1.Object) error {
		return m.ReleaseVirtualMachine(obj.(*virtv1.VirtualMachineInstance))
	}

	ok, err := m.ClaimObject(vmi, match, adopt, release)
	if err != nil {
		return nil, err
	}
	if ok {
		return vmi, nil
	}
	return nil, nil
}

// AdoptVirtualMachine sends a patch to take control of the vmi. It returns the error if
// the patching fails.
func (m *VirtualMachineControllerRefManager) AdoptVirtualMachine(vmi *virtv1.VirtualMachineInstance) error {
	if err := m.CanAdopt(); err != nil {
		return fmt.Errorf("can't adopt VirtualMachineInstance %v/%v (%v): %v", vmi.Namespace, vmi.Name, vmi.UID, err)
	}
	// Note that ValidateOwnerReferences() will reject this patch if another
	// OwnerReference exists with controller=true.
	addControllerPatch := fmt.Sprintf(
		`{"metadata":{"ownerReferences":[{"apiVersion":"%s","kind":"%s","name":"%s","uid":"%s","controller":true,"blockOwnerDeletion":true}],"uid":"%s"}}`,
		m.controllerKind.GroupVersion(), m.controllerKind.Kind,
		m.Controller.GetName(), m.Controller.GetUID(), vmi.UID)
	return m.virtualMachineControl.PatchVirtualMachine(vmi.Namespace, vmi.Name, []byte(addControllerPatch))
}

// ReleaseVirtualMachine sends a patch to free the virtual machine from the control of the controller.
// It returns the error if the patching fails. 404 and 422 errors are ignored.
func (m *VirtualMachineControllerRefManager) ReleaseVirtualMachine(vmi *virtv1.VirtualMachineInstance) error {
	log.Log.V(2).Object(vmi).Infof("patching vmi to remove its controllerRef to %s/%s:%s",
		m.controllerKind.GroupVersion(), m.controllerKind.Kind, m.Controller.GetName())
	// TODO CRDs don't support strategic merge, therefore replace the onwerReferences list with a merge patch
	deleteOwnerRefPatch := fmt.Sprint(`{"metadata":{"ownerReferences":[]}}`)
	err := m.virtualMachineControl.PatchVirtualMachine(vmi.Namespace, vmi.Name, []byte(deleteOwnerRefPatch))
	if err != nil {
		if errors.IsNotFound(err) {
			// If the vmi no longer exists, ignore it.
			return nil
		}
		if errors.IsInvalid(err) {
			// Invalid error will be returned in two cases: 1. the vmi
			// has no owner reference, 2. the uid of the vmi doesn't
			// match, which means the vmi is deleted and then recreated.
			// In both cases, the error can be ignored.

			// TODO: If the vmi has owner references, but none of them
			// has the owner.UID, server will silently ignore the patch.
			// Investigate why.
			return nil
		}
	}
	return err
}

// AdoptDataVolume sends a patch to take control of the dataVolume. It returns the error if
// the patching fails.
func (m *VirtualMachineControllerRefManager) AdoptDataVolume(dataVolume *cdiv1.DataVolume) error {
	if err := m.CanAdopt(); err != nil {
		return fmt.Errorf("can't adopt DataVolume %v/%v (%v): %v", dataVolume.Namespace, dataVolume.Name, dataVolume.UID, err)
	}
	// Note that ValidateOwnerReferences() will reject this patch if another
	// OwnerReference exists with controller=true.
	addControllerPatch := fmt.Sprintf(
		`{"metadata":{"ownerReferences":[{"apiVersion":"%s","kind":"%s","name":"%s","uid":"%s","controller":true,"blockOwnerDeletion":true}],"uid":"%s"}}`,
		m.controllerKind.GroupVersion(), m.controllerKind.Kind,
		m.Controller.GetName(), m.Controller.GetUID(), dataVolume.UID)
	return m.virtualMachineControl.PatchVirtualMachine(dataVolume.Namespace, dataVolume.Name, []byte(addControllerPatch))
}

// ReleaseDataVolume sends a patch to free the dataVolume from the control of the controller.
// It returns the error if the patching fails. 404 and 422 errors are ignored.
func (m *VirtualMachineControllerRefManager) ReleaseDataVolume(dataVolume *cdiv1.DataVolume) error {
	log.Log.V(2).Object(dataVolume).Infof("patching dataVolume to remove its controllerRef to %s/%s:%s",
		m.controllerKind.GroupVersion(), m.controllerKind.Kind, m.Controller.GetName())
	// TODO CRDs don't support strategic merge, therefore replace the onwerReferences list with a merge patch
	deleteOwnerRefPatch := fmt.Sprint(`{"metadata":{"ownerReferences":[]}}`)
	err := m.virtualMachineControl.PatchDataVolume(dataVolume.Namespace, dataVolume.Name, []byte(deleteOwnerRefPatch))
	if err != nil {
		if errors.IsNotFound(err) {
			// If no longer exists, ignore it.
			return nil
		}
		if errors.IsInvalid(err) {
			// Invalid error will be returned in two cases: 1. the dataVolume
			// has no owner reference, 2. the uid of the dataVolume doesn't
			// match, which means the dataVolume is deleted and then recreated.
			// In both cases, the error can be ignored.

			// TODO: If the dataVolume has owner references, but none of them
			// has the owner.UID, server will silently ignore the patch.
			// Investigate why.
			return nil
		}
	}
	return err
}

type VirtualMachineControlInterface interface {
	PatchVirtualMachine(namespace, name string, data []byte) error
	PatchDataVolume(namespace, name string, data []byte) error
}

type RealVirtualMachineControl struct {
	Clientset kubecli.KubevirtClient
}

func (r RealVirtualMachineControl) PatchVirtualMachine(namespace, name string, data []byte) error {
	// TODO should be a strategic merge patch, but not possible until https://github.com/kubernetes/kubernetes/issues/56348 is resolved
	_, err := r.Clientset.VirtualMachineInstance(namespace).Patch(name, types.MergePatchType, data)
	return err
}

func (r RealVirtualMachineControl) PatchDataVolume(namespace, name string, data []byte) error {
	// TODO should be a strategic merge patch, but not possible until https://github.com/kubernetes/kubernetes/issues/56348 is resolved
	_, err := r.Clientset.CdiClient().CdiV1alpha1().DataVolumes(namespace).Patch(name, types.MergePatchType, data)
	return err
}

// RecheckDeletionTimestamp returns a CanAdopt() function to recheck deletion.
//
// The CanAdopt() function calls getObject() to fetch the latest value,
// and denies adoption attempts if that object has a non-nil DeletionTimestamp.
func RecheckDeletionTimestamp(getObject func() (metav1.Object, error)) func() error {
	return func() error {
		obj, err := getObject()
		if err != nil {
			return fmt.Errorf("can't recheck DeletionTimestamp: %v", err)
		}
		if obj.GetDeletionTimestamp() != nil {
			return fmt.Errorf("%v/%v has just been deleted at %v", obj.GetNamespace(), obj.GetName(), obj.GetDeletionTimestamp())
		}
		return nil
	}
}
