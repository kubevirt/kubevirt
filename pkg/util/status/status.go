package status

import (
	"reflect"
	"sync"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

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

func (u *updater) patch(obj runtime.Object, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (err error) {
	if u.getSubresource() {
		return u.patchWithSubresource(obj, pt, data, patchOptions)
	} else {
		return u.patchWithoutSubresource(obj, pt, data, patchOptions)
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
	if !reflect.DeepEqual(oldStatus, newStatus) {
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
	if updateStatusErr != nil && !errors.IsNotFound(updateStatusErr) {
		return updateStatusErr
	}
	if errors.IsNotFound(updateStatusErr) {
		oldStatus, newStatus, err := u.updateUnstructured(obj)
		if errors.IsNotFound(err) {
			// object does not exist
			return err
		}
		if err == nil && reflect.DeepEqual(oldStatus, newStatus) {
			u.setSubresource(false)
		} else if err == nil {
			return updateStatusErr
		}
		return err
	} else {
		return updateStatusErr
	}
}

// patchWithoutSubresource will try to update the  status via PATCH sent to the main REST endpoint.
// If the resource version of the returned object did not change, it knows that it should have used the /status subresource
// and will switch the updater itself over to permanently use the /status subresource.
func (u *updater) patchWithoutSubresource(obj runtime.Object, patchType types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (err error) {
	oldResourceVersion, newResourceVersion, err := u.patchUnstructured(obj, patchType, data, patchOptions)
	if err != nil {
		return err
	}
	if oldResourceVersion == newResourceVersion {
		u.setSubresource(true)
		return u.patchStatusUnstructured(obj, patchType, data, patchOptions)
	}
	return nil
}

// patchWithSubresource will try to update the  status via PATCH sent to the /status subresource.
// If a 404 error is returned, it will try the main rest entrypoint instead. In case that this
// call succeeds, it will switch the updater to permanently use the main entrypoint.
func (u *updater) patchWithSubresource(obj runtime.Object, patchType types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (patchStatusErr error) {
	patchStatusErr = u.patchStatusUnstructured(obj, patchType, data, patchOptions)
	if patchStatusErr != nil && !errors.IsNotFound(patchStatusErr) {
		return patchStatusErr
	}
	if errors.IsNotFound(patchStatusErr) {
		oldResourceVersion, newResourceVersions, err := u.patchUnstructured(obj, patchType, data, patchOptions)
		if errors.IsNotFound(err) {
			// object does not exist
			return err
		}
		if err == nil && oldResourceVersion != newResourceVersions {
			u.setSubresource(false)
		} else if err == nil {
			return patchStatusErr
		}
		return err
	} else {
		return patchStatusErr
	}
}

func (u *updater) patchUnstructured(obj runtime.Object, patchType types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (oldResourceVersion, newResourceVerions string, err error) {
	a, err := meta.Accessor(obj)
	if err != nil {
		return "", "", err
	}
	switch obj.(type) {
	case *v1.VirtualMachine:
		oldObj := obj.(*v1.VirtualMachine)
		newObj, err := u.cli.VirtualMachine(a.GetNamespace()).Patch(a.GetName(), patchType, data, patchOptions)
		if err != nil {
			return "", "", err
		}
		return oldObj.ResourceVersion, newObj.ResourceVersion, nil
	case *v1.KubeVirt:
		oldObj := obj.(*v1.KubeVirt)
		newObj, err := u.cli.KubeVirt(a.GetNamespace()).Patch(a.GetName(), patchType, data, patchOptions)
		if err != nil {
			return "", "", err
		}
		return oldObj.ResourceVersion, newObj.ResourceVersion, nil
	default:
		panic("Unknown object")
	}
}

func (u *updater) patchStatusUnstructured(obj runtime.Object, patchType types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (err error) {
	a, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	switch obj.(type) {
	case *v1.VirtualMachine:
		_, err = u.cli.VirtualMachine(a.GetNamespace()).PatchStatus(a.GetName(), patchType, data, patchOptions)
		return err
	case *v1.KubeVirt:
		_, err = u.cli.KubeVirt(a.GetNamespace()).PatchStatus(a.GetName(), patchType, data, patchOptions)
		return err
	default:
		panic("Unknown object")
	}
}

func (u *updater) updateUnstructured(obj runtime.Object) (oldStatus interface{}, newStatus interface{}, err error) {
	a, err := meta.Accessor(obj)
	if err != nil {
		return nil, nil, err
	}
	switch obj.(type) {
	case *v1.VirtualMachine:
		oldObj := obj.(*v1.VirtualMachine)
		newObj, err := u.cli.VirtualMachine(a.GetNamespace()).Update(oldObj)
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *v1.VirtualMachineInstanceReplicaSet:
		oldObj := obj.(*v1.VirtualMachineInstanceReplicaSet)
		newObj, err := u.cli.ReplicaSet(a.GetNamespace()).Update(oldObj)
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *v1.VirtualMachineInstanceMigration:
		oldObj := obj.(*v1.VirtualMachineInstanceMigration)
		newObj, err := u.cli.VirtualMachineInstanceMigration(a.GetNamespace()).Update(oldObj)
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *v1.KubeVirt:
		oldObj := obj.(*v1.KubeVirt)
		newObj, err := u.cli.KubeVirt(a.GetNamespace()).Update(oldObj)
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	default:
		panic("Unknown object")
	}
}

func (u *updater) updateStatusUnstructured(obj runtime.Object) (err error) {
	a, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	switch obj.(type) {
	case *v1.VirtualMachine:
		oldObj := obj.(*v1.VirtualMachine)
		_, err = u.cli.VirtualMachine(a.GetNamespace()).UpdateStatus(oldObj)
		return err
	case *v1.VirtualMachineInstanceReplicaSet:
		oldObj := obj.(*v1.VirtualMachineInstanceReplicaSet)
		_, err = u.cli.ReplicaSet(a.GetNamespace()).UpdateStatus(oldObj)
		return err
	case *v1.VirtualMachineInstanceMigration:
		oldObj := obj.(*v1.VirtualMachineInstanceMigration)
		_, err = u.cli.VirtualMachineInstanceMigration(a.GetNamespace()).UpdateStatus(oldObj)
		return err
	case *v1.KubeVirt:
		oldObj := obj.(*v1.KubeVirt)
		_, err = u.cli.KubeVirt(a.GetNamespace()).UpdateStatus(oldObj)
		return err
	default:
		panic("Unknown object")
	}
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

type VMStatusUpdater struct {
	updater updater
}

func (v *VMStatusUpdater) UpdateStatus(vm *v1.VirtualMachine) error {
	return v.updater.update(vm)
}

func (v *VMStatusUpdater) PatchStatus(vm *v1.VirtualMachine, pt types.PatchType, data []byte, patchOptions *metav1.PatchOptions) error {
	return v.updater.patch(vm, pt, data, patchOptions)
}

func NewVMStatusUpdater(cli kubecli.KubevirtClient) *VMStatusUpdater {
	return &VMStatusUpdater{
		updater: updater{
			lock:        sync.Mutex{},
			subresource: true,
			cli:         cli,
		},
	}
}

type VMIRSStatusUpdater struct {
	updater updater
}

func (v *VMIRSStatusUpdater) UpdateStatus(vmirs *v1.VirtualMachineInstanceReplicaSet) error {
	return v.updater.update(vmirs)
}

func NewVMIRSStatusUpdater(cli kubecli.KubevirtClient) *VMIRSStatusUpdater {
	return &VMIRSStatusUpdater{
		updater: updater{
			lock:        sync.Mutex{},
			subresource: true,
			cli:         cli,
		},
	}
}

type KVStatusUpdater struct {
	updater updater
}

func (v *KVStatusUpdater) UpdateStatus(kv *v1.KubeVirt) error {
	return v.updater.update(kv)
}

func (v *KVStatusUpdater) PatchStatus(kv *v1.KubeVirt, pt types.PatchType, data []byte) error {
	return v.updater.patch(kv, pt, data, &metav1.PatchOptions{})
}

func NewKubeVirtStatusUpdater(cli kubecli.KubevirtClient) *KVStatusUpdater {
	return &KVStatusUpdater{
		updater: updater{
			lock:        sync.Mutex{},
			subresource: true,
			cli:         cli,
		},
	}
}

type MigrationStatusUpdater struct {
	updater updater
}

func (v *MigrationStatusUpdater) UpdateStatus(migration *v1.VirtualMachineInstanceMigration) error {
	return v.updater.update(migration)
}

func NewMigrationStatusUpdater(cli kubecli.KubevirtClient) *MigrationStatusUpdater {
	return &MigrationStatusUpdater{
		updater: updater{
			lock:        sync.Mutex{},
			subresource: true,
			cli:         cli,
		},
	}
}
