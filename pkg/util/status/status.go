package status

import (
	"context"
	"sync"

	clonev1alpha1 "kubevirt.io/api/clone/v1alpha1"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1beta1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
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
	if !errors.IsNotFound(patchStatusErr) {
		return patchStatusErr
	}
	oldResourceVersion, newResourceVersions, err := u.patchUnstructured(obj, patchType, data, patchOptions)
	if err != nil {
		return err
	}
	if oldResourceVersion == newResourceVersions {
		return patchStatusErr
	}
	u.setSubresource(false)
	return nil
}

func (u *updater) patchUnstructured(obj runtime.Object, patchType types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (oldResourceVersion, newResourceVerions string, err error) {
	a, err := meta.Accessor(obj)
	if err != nil {
		return "", "", err
	}
	switch obj.(type) {
	case *v1.VirtualMachine:
		oldObj := obj.(*v1.VirtualMachine)
		newObj, err := u.cli.VirtualMachine(a.GetNamespace()).Patch(context.Background(), a.GetName(), patchType, data, *patchOptions)
		if err != nil {
			return "", "", err
		}
		return oldObj.ResourceVersion, newObj.ResourceVersion, nil
	case *v1.KubeVirt:
		oldObj := obj.(*v1.KubeVirt)
		newObj, err := u.cli.KubeVirt(a.GetNamespace()).Patch(context.Background(), a.GetName(), patchType, data, *patchOptions)
		if err != nil {
			return "", "", err
		}
		return oldObj.ResourceVersion, newObj.ResourceVersion, nil
	case *poolv1.VirtualMachinePool:
		oldObj := obj.(*poolv1.VirtualMachinePool)
		newObj, err := u.cli.VirtualMachinePool(a.GetNamespace()).Patch(context.Background(), a.GetName(), patchType, data, *patchOptions)
		if err != nil {
			return "", "", err
		}
		return oldObj.ResourceVersion, newObj.ResourceVersion, nil
	default:
		panic(unknownObj)
	}
}

func (u *updater) patchStatusUnstructured(obj runtime.Object, patchType types.PatchType, data []byte, patchOptions *metav1.PatchOptions) (err error) {
	a, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	switch obj.(type) {
	case *v1.VirtualMachine:
		_, err = u.cli.VirtualMachine(a.GetNamespace()).PatchStatus(context.Background(), a.GetName(), patchType, data, *patchOptions)
		return err
	case *v1.KubeVirt:
		_, err = u.cli.KubeVirt(a.GetNamespace()).PatchStatus(context.Background(), a.GetName(), patchType, data, *patchOptions)
		return err
	default:
		panic(unknownObj)
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
		newObj, err := u.cli.VirtualMachine(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *v1.VirtualMachineInstanceReplicaSet:
		oldObj := obj.(*v1.VirtualMachineInstanceReplicaSet)
		newObj, err := u.cli.ReplicaSet(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *v1.VirtualMachineInstanceMigration:
		oldObj := obj.(*v1.VirtualMachineInstanceMigration)
		newObj, err := u.cli.VirtualMachineInstanceMigration(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *v1.KubeVirt:
		oldObj := obj.(*v1.KubeVirt)
		newObj, err := u.cli.KubeVirt(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *poolv1.VirtualMachinePool:
		oldObj := obj.(*poolv1.VirtualMachinePool)
		newObj, err := u.cli.VirtualMachinePool(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	case *exportv1.VirtualMachineExport:
		oldObj := obj.(*exportv1.VirtualMachineExport)
		newObj, err := u.cli.VirtualMachineExport(a.GetNamespace()).Update(context.Background(), oldObj, metav1.UpdateOptions{})
		if err != nil {
			return nil, nil, err
		}
		return oldObj.Status, newObj.Status, nil
	default:
		panic(unknownObj)
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
		_, err = u.cli.VirtualMachine(a.GetNamespace()).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *v1.VirtualMachineInstanceReplicaSet:
		oldObj := obj.(*v1.VirtualMachineInstanceReplicaSet)
		_, err = u.cli.ReplicaSet(a.GetNamespace()).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *v1.VirtualMachineInstanceMigration:
		oldObj := obj.(*v1.VirtualMachineInstanceMigration)
		_, err = u.cli.VirtualMachineInstanceMigration(a.GetNamespace()).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *v1.KubeVirt:
		oldObj := obj.(*v1.KubeVirt)
		_, err = u.cli.KubeVirt(a.GetNamespace()).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *clonev1alpha1.VirtualMachineClone:
		oldObj := obj.(*clonev1alpha1.VirtualMachineClone)
		_, err = u.cli.VirtualMachineClone(oldObj.Namespace).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *poolv1.VirtualMachinePool:
		oldObj := obj.(*poolv1.VirtualMachinePool)
		_, err = u.cli.VirtualMachinePool(oldObj.Namespace).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
	case *exportv1.VirtualMachineExport:
		oldObj := obj.(*exportv1.VirtualMachineExport)
		_, err = u.cli.VirtualMachineExport(oldObj.Namespace).UpdateStatus(context.Background(), oldObj, metav1.UpdateOptions{})
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
