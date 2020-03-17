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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package watch

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	vmsnapshotv1alpha1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	vmFinalizer = "snapshot.kubevirt.io/vm-as-source-protection"

	vmSnapshotFinalizer = "snapshot.kubevirt.io/vmsnapshot-as-source-protection"

	vmSnapshotBoundFinalizer = "snapshot.kubevirt.io/vmsnapshot-bound-protection"

	vmSnapshotContentFinalizer = "snapshot.kubevirt.io/vmsnapshotcontent-bound-protection"
)

func cacheKeyFunc(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func vmSnapshotError(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) *vmsnapshotv1alpha1.VirtualMachineSnapshotError {
	if vmSnapshot.Status != nil && vmSnapshot.Status.Error != nil {
		return vmSnapshot.Status.Error
	}
	return nil
}

func vmSnapshotProceeding(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) bool {
	return vmSnapshotError(vmSnapshot) == nil &&
		(vmSnapshot.Status == nil || vmSnapshot.Status.ReadyToUse == nil || *vmSnapshot.Status.ReadyToUse == false)
}

func getVMSnapshotContentName(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) string {
	if vmSnapshot.Status != nil && vmSnapshot.Status.VirtualMachineSnapshotContentName != nil {
		return *vmSnapshot.Status.VirtualMachineSnapshotContentName
	}

	return fmt.Sprintf("%s-%s", "vmsnapshot-content", vmSnapshot.UID)
}

func (ctrl *SnapshotController) updateVMSnapshot(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) error {
	proceed, err := ctrl.checkSource(vmSnapshot)
	if !proceed || err != nil {
		return err
	}

	if vmSnapshot.DeletionTimestamp != nil {
		return ctrl.handleDeletedVMSnapshot(vmSnapshot)
	}

	if vmSnapshot.Status == nil || vmSnapshot.Status.VirtualMachineSnapshotContentName == nil {
		proceed, err = ctrl.checkFinalizers(vmSnapshot)
		if !proceed || err != nil {
			return err
		}

		if err = ctrl.createContent(vmSnapshot); err != nil {
			return err
		}
	}

	if err = ctrl.updateSnapshotStatus(vmSnapshot); err != nil {
		return err
	}

	return nil
}

func (ctrl *SnapshotController) updateVMSnapshotContent(content *vmsnapshotv1alpha1.VirtualMachineSnapshotContent) error {
	//TODO - check snapshots
	ready := true
	now := metav1.Now()

	contentCpy := content.DeepCopy()
	contentCpy.Status.ReadyToUse = &ready
	if contentCpy.Status.CreationTime == nil {
		contentCpy.Status.CreationTime = &now
	}

	if !reflect.DeepEqual(content, contentCpy) {
		if _, err := ctrl.client.VirtualMachineSnapshotContent(contentCpy.Namespace).Update(contentCpy); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *SnapshotController) checkSource(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (bool, error) {
	switch vmSnapshot.Spec.Source.Kind {
	case "VirtualMachine":
		return ctrl.updateVM(vmSnapshot)
	}

	return false, fmt.Errorf("unknown datasource kind %s", vmSnapshot.Spec.Source.Kind)
}

func (ctrl *SnapshotController) checkFinalizers(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (bool, error) {
	if vmSnapshot.DeletionTimestamp != nil {
		return true, nil
	}

	vmSnapshotCpy := vmSnapshot.DeepCopy()
	controller.AddFinalizer(vmSnapshotCpy, vmSnapshotFinalizer)
	controller.AddFinalizer(vmSnapshotCpy, vmSnapshotBoundFinalizer)

	if !reflect.DeepEqual(vmSnapshot, vmSnapshotCpy) {
		if _, err := ctrl.client.VirtualMachineSnapshot(vmSnapshot.Namespace).Update(vmSnapshotCpy); err != nil {
			return false, err
		}

		return false, nil
	}

	return true, nil
}

func (ctrl *SnapshotController) handleDeletedVMSnapshot(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) error {
	// TODO check restore in progress

	if vmSnapshotProceeding(vmSnapshot) {
		log.Log.Infof("VirtualMachineSnapshot %s proceeding, so not removing finalizers", vmSnapshot.Name)
		return nil
	}

	content, err := ctrl.getContent(vmSnapshot)
	if err != nil {
		return err
	}

	if content != nil && controller.HasFinalizer(content, vmSnapshotContentFinalizer) {
		cpy := content.DeepCopy()
		controller.RemoveFinalizer(cpy, vmSnapshotContentFinalizer)

		_, err := ctrl.client.VirtualMachineSnapshotContent(cpy.Namespace).Update(cpy)
		if err != nil {
			return err
		}

		// TODO revisit with retention policy
		err = ctrl.client.VirtualMachineSnapshotContent(cpy.Namespace).Delete(cpy.Name, &metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
	}

	vmSnapshotCpy := vmSnapshot.DeepCopy()
	controller.RemoveFinalizer(vmSnapshotCpy, vmSnapshotFinalizer)
	controller.RemoveFinalizer(vmSnapshotCpy, vmSnapshotBoundFinalizer)

	if !reflect.DeepEqual(vmSnapshot, vmSnapshotCpy) {
		_, err := ctrl.client.VirtualMachineSnapshot(vmSnapshotCpy.Namespace).Update(vmSnapshotCpy)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *SnapshotController) createContent(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) error {
	content, err := ctrl.getContent(vmSnapshot)
	if err != nil {
		return err
	}

	if content != nil {
		return nil
	}

	resourceSpec, err := ctrl.getResourceSpec(vmSnapshot)
	if err != nil {
		return err
	}

	ready := false
	content = &vmsnapshotv1alpha1.VirtualMachineSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       getVMSnapshotContentName(vmSnapshot),
			Namespace:  vmSnapshot.Namespace,
			Finalizers: []string{vmSnapshotContentFinalizer},
		},
		Spec: vmsnapshotv1alpha1.VirtualMachineSnapshotContentSpec{
			VirtualMachineSnapshotName: &vmSnapshot.Name,
			Source:                     *resourceSpec,
		},
		Status: &vmsnapshotv1alpha1.VirtualMachineSnapshotContentStatus{
			ReadyToUse: &ready,
		},
	}

	_, err = ctrl.client.VirtualMachineSnapshotContent(content.Namespace).Create(content)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (ctrl *SnapshotController) updateSnapshotStatus(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) error {
	content, err := ctrl.getContent(vmSnapshot)
	if err != nil {
		return err
	}

	if content == nil {
		return nil
	}

	vmSnapshotCpy := vmSnapshot.DeepCopy()
	if vmSnapshotCpy.Status == nil {
		vmSnapshotCpy.Status = &vmsnapshotv1alpha1.VirtualMachineSnapshotStatus{}
	}

	vmSnapshotCpy.Status.VirtualMachineSnapshotContentName = &content.Name
	vmSnapshotCpy.Status.CreationTime = content.Status.CreationTime
	vmSnapshotCpy.Status.ReadyToUse = content.Status.ReadyToUse

	if !reflect.DeepEqual(vmSnapshot, vmSnapshotCpy) {
		if _, err = ctrl.client.VirtualMachineSnapshot(vmSnapshotCpy.Namespace).Update(vmSnapshotCpy); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *SnapshotController) updateVM(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (bool, error) {
	vm, err := ctrl.getVM(vmSnapshot)
	if err != nil {
		return false, err
	}

	if vm == nil {
		return true, nil
	}

	if vm.Spec.Running == nil || *vm.Spec.Running {
		log.Log.V(3).Infof("Snapshottting a running VM is not supported yet")
		return false, nil
	}

	if vm.Status.SnapshotInProgress != nil && *vm.Status.SnapshotInProgress != vmSnapshot.Name {
		log.Log.V(3).Infof("Snapshot %s in progress", *vm.Status.SnapshotInProgress)
		return false, nil
	}

	if vmSnapshotProceeding(vmSnapshot) {
		if vmSnapshot.DeletionTimestamp == nil && vm.Status.SnapshotInProgress == nil {
			log.Log.Infof("Adding VM snapshot finalizer to %s", vm.Name)

			vmCopy := vm.DeepCopy()
			vmCopy.Status.SnapshotInProgress = &vmSnapshot.Name
			controller.AddFinalizer(vmCopy, vmFinalizer)

			_, err := ctrl.client.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
			// exit and resume when we get VM update callback
			return false, err
		}

		// deleted or this snapshot proceeding
		return true, nil
	}

	// done or error
	if vm.Status.SnapshotInProgress != nil && *vm.Status.SnapshotInProgress == vmSnapshot.Name {
		vmCopy := vm.DeepCopy()
		vmCopy.Status.SnapshotInProgress = nil
		controller.RemoveFinalizer(vmCopy, vmFinalizer)

		_, err := ctrl.client.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (ctrl *SnapshotController) getResourceSpec(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (*vmsnapshotv1alpha1.SourceSpec, error) {
	switch vmSnapshot.Spec.Source.Kind {
	case "VirtualMachine":
		vm, err := ctrl.getVM(vmSnapshot)
		if err != nil {
			return nil, err
		}
		return &vmsnapshotv1alpha1.SourceSpec{VirtualMachineSpec: &vm.Spec}, nil
	}

	return nil, nil
}

func (ctrl *SnapshotController) getVM(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (*kubevirtv1.VirtualMachine, error) {
	vmName := vmSnapshot.Spec.Source.Name
	obj, exists, err := ctrl.vmInformer.GetStore().GetByKey(cacheKeyFunc(vmSnapshot.Namespace, vmName))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	return obj.(*kubevirtv1.VirtualMachine), nil
}

func (ctrl *SnapshotController) getContent(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (*vmsnapshotv1alpha1.VirtualMachineSnapshotContent, error) {
	contentName := getVMSnapshotContentName(vmSnapshot)
	obj, exists, err := ctrl.vmSnapshotContentInformer.GetStore().GetByKey(cacheKeyFunc(vmSnapshot.Namespace, contentName))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	return obj.(*vmsnapshotv1alpha1.VirtualMachineSnapshotContent), nil
}
