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
 */

package status

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	exportv1 "kubevirt.io/api/export/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
)

const unknownObj = "Unknown object"

// updater transparently switches for status updates between /status and the main entrypoint for resource,
// allowing CRDs to enable or disable the status subresource support anytime.
type updater struct {
	lock        sync.Mutex
	subresource bool
	cli         kubecli.KubevirtClient
}

func (u *updater) update(obj runtime.Object) (err error) {
	if u.getSubresource() {
		return u.updateWithSubresource(obj)
	} else {
		return u.updateWithoutSubresource(obj)
	}
}

// updateWithoutSubresource will try to update the  status via PUT sent to the main REST endpoint.
// If status of the returned object did not change, it knows that it should have used the /status subresource
// and will switch the updater itself over to permanently use the /status subresource.
func (u *updater) updateWithoutSubresource(obj runtime.Object) (err error) {
	oldStatus, newStatus, err := u.updateUnstructured(obj)
	if err != nil {
		return err
	}
	if !equality.Semantic.DeepEqual(oldStatus, newStatus) {
		u.setSubresource(true)
		return u.updateStatusUnstructured(obj)
	}
	return nil
}

// updateWithSubresource will try to update the  status via PUT sent to the /status subresource.
// If a 404 error is returned, it will try the main rest entrypoint instead. In case that this
// call succeeds, it will switch the updater to permanently use the main entrypoint.
func (u *updater) updateWithSubresource(obj runtime.Object) (updateStatusErr error) {
	updateStatusErr = u.updateStatusUnstructured(obj)
	if !errors.IsNotFound(updateStatusErr) {
		return updateStatusErr
	}
	oldStatus, newStatus, err := u.updateUnstructured(obj)
	if err != nil {
		return err
	}
	if !equality.Semantic.DeepEqual(oldStatus, newStatus) {
		return updateStatusErr
	}
	u.setSubresource(false)
	return nil
}

func (u *updater) updateUnstructured(obj runtime.Object) (oldStatus interface{}, newStatus interface{}, err error) {
	a, err := meta.Accessor(obj)
	if err != nil {
		return nil, nil, err
	}
	switch obj.(type) {
	case *exportv1.VirtualMachineExport:
		oldObj := obj.(*exportv1.VirtualMachineExport)
		newObj, err := u.cli.VirtualMachineExport(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *snapshotv1.VirtualMachineSnapshot:
		oldObj := obj.(*snapshotv1.VirtualMachineSnapshot)
		newObj, err := u.cli.VirtualMachineSnapshot(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *snapshotv1.VirtualMachineSnapshotContent:
		oldObj := obj.(*snapshotv1.VirtualMachineSnapshotContent)
		newObj, err := u.cli.VirtualMachineSnapshotContent(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *snapshotv1.VirtualMachineRestore:
		oldObj := obj.(*snapshotv1.VirtualMachineRestore)
		newObj, err := u.cli.VirtualMachineRestore(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	default:
		panic(unknownObj)
	}
}

func (u *updater) updateStatusUnstructured(obj runtime.Object) (err error) {
	switch obj.(type) {
	case *exportv1.VirtualMachineExport:
		oldObj := obj.(*exportv1.VirtualMachineExport)
		_, err = u.cli.VirtualMachineExport(oldObj.Namespace).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *snapshotv1.VirtualMachineSnapshot:
		oldObj := obj.(*snapshotv1.VirtualMachineSnapshot)
		_, err = u.cli.VirtualMachineSnapshot(oldObj.Namespace).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *snapshotv1.VirtualMachineSnapshotContent:
		oldObj := obj.(*snapshotv1.VirtualMachineSnapshotContent)
		_, err = u.cli.VirtualMachineSnapshotContent(oldObj.Namespace).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *snapshotv1.VirtualMachineRestore:
		oldObj := obj.(*snapshotv1.VirtualMachineRestore)
		_, err = u.cli.VirtualMachineRestore(oldObj.Namespace).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	default:
		panic(unknownObj)
	}

	return err
}

func (u *updater) setSubresource(exists bool) {
	u.lock.Lock()
	defer u.lock.Unlock()
	u.subresource = exists
}

func (u *updater) getSubresource() bool {
	u.lock.Lock()
	defer u.lock.Unlock()
	return u.subresource
}

type VMExportStatusUpdater struct {
	updater
}

func (v *VMExportStatusUpdater) UpdateStatus(vmExport *exportv1.VirtualMachineExport) error {
	return v.update(vmExport)
}

func NewVMExportStatusUpdater(cli kubecli.KubevirtClient) *VMExportStatusUpdater {
	return &VMExportStatusUpdater{
		updater: updater{
			lock:        sync.Mutex{},
			subresource: true,
			cli:         cli,
		},
	}
}

type VMSnapshotStatusUpdater struct {
	updater
}

func (v *VMSnapshotStatusUpdater) UpdateStatus(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	return v.update(vmSnapshot)
}

func NewVMSnapshotStatusUpdater(cli kubecli.KubevirtClient) *VMSnapshotStatusUpdater {
	return &VMSnapshotStatusUpdater{
		updater: updater{
			lock:        sync.Mutex{},
			subresource: true,
			cli:         cli,
		},
	}
}

type VMSnapshotContentStatusUpdater struct {
	updater
}

func (v *VMSnapshotContentStatusUpdater) UpdateStatus(vmSnapshotContent *snapshotv1.VirtualMachineSnapshotContent) error {
	return v.update(vmSnapshotContent)
}

func NewVMSnapshotContentStatusUpdater(cli kubecli.KubevirtClient) *VMSnapshotContentStatusUpdater {
	return &VMSnapshotContentStatusUpdater{
		updater: updater{
			lock:        sync.Mutex{},
			subresource: true,
			cli:         cli,
		},
	}
}

type VMRestoreStatusUpdater struct {
	updater
}

func (v *VMRestoreStatusUpdater) UpdateStatus(vmRestore *snapshotv1.VirtualMachineRestore) error {
	return v.update(vmRestore)
}

func NewVMRestoreStatusUpdater(cli kubecli.KubevirtClient) *VMRestoreStatusUpdater {
	return &VMRestoreStatusUpdater{
		updater: updater{
			lock:        sync.Mutex{},
			subresource: true,
			cli:         cli,
		},
	}
}
