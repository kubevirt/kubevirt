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

package snapshot

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	pvcRestoreAnnotation = "restore.kubevirt.io/name"

	populatedForPVCAnnotation = "cdi.kubevirt.io/storage.populatedFor"

	lastRestoreAnnotation = "restore.kubevirt.io/lastRestoreUID"

	restoreCompleteEvent = "VirtualMachineRestoreComplete"

	restoreErrorEvent = "VirtualMachineRestoreError"
)

type restoreTarget interface {
	UID() types.UID
	Ready() (bool, error)
	Reconcile() (bool, error)
	Cleanup() error
	Own(obj metav1.Object)
	UpdateDoneRestore() (bool, error)
	UpdateRestoreInProgress() error
}

type vmRestoreTarget struct {
	controller *VMRestoreController
	vmRestore  *snapshotv1.VirtualMachineRestore
	vm         *kubevirtv1.VirtualMachine
}

var restoreAnnotationsToDelete = []string{
	"pv.kubernetes.io",
	"volume.beta.kubernetes.io",
	"cdi.kubevirt.io",
}

func restorePVCName(vmRestore *snapshotv1.VirtualMachineRestore, name string) string {
	return fmt.Sprintf("restore-%s-%s", vmRestore.UID, name)
}

func restoreDVName(vmRestore *snapshotv1.VirtualMachineRestore, name string) string {
	return restorePVCName(vmRestore, name)
}

func vmRestoreProgressing(vmRestore *snapshotv1.VirtualMachineRestore) bool {
	return vmRestore.Status == nil || vmRestore.Status.Complete == nil || !*vmRestore.Status.Complete
}

func (ctrl *VMRestoreController) updateVMRestore(vmRestoreIn *snapshotv1.VirtualMachineRestore) (time.Duration, error) {
	logger := log.Log.Object(vmRestoreIn)

	logger.V(1).Infof("Updating VirtualMachineRestore")

	vmRestoreOut := vmRestoreIn.DeepCopy()

	if vmRestoreOut.Status == nil {
		f := false
		vmRestoreOut.Status = &snapshotv1.VirtualMachineRestoreStatus{
			Complete: &f,
		}
	}

	target, err := ctrl.getTarget(vmRestoreOut)
	if err != nil {
		logger.Reason(err).Error("Error getting restore target")
		return 0, ctrl.doUpdateError(vmRestoreOut, err)
	}

	if !vmRestoreProgressing(vmRestoreIn) && target != nil {
		//update the vm if Done restore
		if updated, err := target.UpdateDoneRestore(); updated || err != nil {
			return 0, err
		}

		return 0, nil
	}

	if len(vmRestoreOut.OwnerReferences) == 0 {
		target.Own(vmRestoreOut)
		updateRestoreCondition(vmRestoreOut, newProgressingCondition(corev1.ConditionTrue, "Initializing VirtualMachineRestore"))
		updateRestoreCondition(vmRestoreOut, newReadyCondition(corev1.ConditionFalse, "Initializing VirtualMachineRestore"))
	}

	err = target.UpdateRestoreInProgress()
	if err != nil {
		return 0, err
	}

	// let's make sure everything is initialized properly before continuing
	if !reflect.DeepEqual(vmRestoreIn, vmRestoreOut) {
		return 0, ctrl.doUpdate(vmRestoreIn, vmRestoreOut)
	}

	var updated bool
	updated, err = ctrl.reconcileVolumeRestores(vmRestoreOut, target)
	if err != nil {
		logger.Reason(err).Error("Error reconciling VolumeRestores")
		return 0, ctrl.doUpdateError(vmRestoreIn, err)
	}

	if !updated {
		var ready bool
		ready, err = target.Ready()
		if err != nil {
			logger.Reason(err).Error("Error checking target ready")
			return 0, ctrl.doUpdateError(vmRestoreIn, err)
		}

		if ready {
			updated, err = target.Reconcile()
			if err != nil {
				logger.Reason(err).Error("Error reconciling target")
				return 0, ctrl.doUpdateError(vmRestoreIn, err)
			}

			if !updated {
				if err = target.Cleanup(); err != nil {
					logger.Reason(err).Error("Error cleaning up")
					return 0, ctrl.doUpdateError(vmRestoreIn, err)
				}

				ctrl.Recorder.Eventf(
					vmRestoreOut,
					corev1.EventTypeNormal,
					restoreCompleteEvent,
					"Successfully completed VirtualMachineRestore %s",
					vmRestoreOut.Name,
				)

				t := true
				vmRestoreOut.Status.Complete = &t
				vmRestoreOut.Status.RestoreTime = currentTime()
				updateRestoreCondition(vmRestoreOut, newProgressingCondition(corev1.ConditionFalse, "Operation complete"))
				updateRestoreCondition(vmRestoreOut, newReadyCondition(corev1.ConditionTrue, "Operation complete"))
			} else {
				updateRestoreCondition(vmRestoreOut, newProgressingCondition(corev1.ConditionTrue, "Updating target spec"))
				updateRestoreCondition(vmRestoreOut, newReadyCondition(corev1.ConditionFalse, "Waiting for target update"))
			}
		} else {
			reason := "Waiting for target to be ready"
			updateRestoreCondition(vmRestoreOut, newProgressingCondition(corev1.ConditionFalse, reason))
			updateRestoreCondition(vmRestoreOut, newReadyCondition(corev1.ConditionFalse, reason))
			// try again in 5 secs
			return 5 * time.Second, ctrl.doUpdate(vmRestoreIn, vmRestoreOut)
		}
	} else {
		updateRestoreCondition(vmRestoreOut, newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"))
		updateRestoreCondition(vmRestoreOut, newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"))
	}

	return 0, ctrl.doUpdate(vmRestoreIn, vmRestoreOut)
}

func (ctrl *VMRestoreController) doUpdateError(restore *snapshotv1.VirtualMachineRestore, err error) error {
	ctrl.Recorder.Eventf(
		restore,
		corev1.EventTypeWarning,
		restoreErrorEvent,
		"VirtualMachineRestore encountered error %s",
		err.Error(),
	)

	updated := restore.DeepCopy()

	updateRestoreCondition(updated, newProgressingCondition(corev1.ConditionFalse, err.Error()))
	updateRestoreCondition(updated, newReadyCondition(corev1.ConditionFalse, err.Error()))
	if err2 := ctrl.doUpdate(restore, updated); err2 != nil {
		return err2
	}

	return err
}

func (ctrl *VMRestoreController) doUpdate(original, updated *snapshotv1.VirtualMachineRestore) error {
	if !reflect.DeepEqual(original, updated) {
		if _, err := ctrl.Client.VirtualMachineRestore(updated.Namespace).Update(context.Background(), updated, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *VMRestoreController) reconcileVolumeRestores(vmRestore *snapshotv1.VirtualMachineRestore, target restoreTarget) (bool, error) {
	content, err := ctrl.getSnapshotContent(vmRestore, target.UID())
	if err != nil {
		return false, err
	}

	var restores []snapshotv1.VolumeRestore
	for _, vb := range content.Spec.VolumeBackups {
		found := false
		for _, vr := range vmRestore.Status.Restores {
			if vb.VolumeName == vr.VolumeName {
				restores = append(restores, vr)
				found = true
				break
			}
		}

		if !found {
			if vb.VolumeSnapshotName == nil {
				return false, fmt.Errorf("VolumeSnapshotName missing %+v", vb)
			}

			vr := snapshotv1.VolumeRestore{
				VolumeName:                vb.VolumeName,
				PersistentVolumeClaimName: restorePVCName(vmRestore, vb.VolumeName),
				VolumeSnapshotName:        *vb.VolumeSnapshotName,
			}
			restores = append(restores, vr)
		}
	}

	if !reflect.DeepEqual(vmRestore.Status.Restores, restores) {
		if len(vmRestore.Status.Restores) > 0 {
			log.Log.Object(vmRestore).Warning("VMRestore in strange state")
		}

		vmRestore.Status.Restores = restores
		return true, nil
	}

	createdPVC := false
	waitingPVC := false
	for i, restore := range restores {
		pvc, err := ctrl.getPVC(vmRestore.Namespace, restore.PersistentVolumeClaimName)
		if err != nil {
			return false, err
		}

		if pvc == nil {
			backup := content.Spec.VolumeBackups[i]
			if err = ctrl.createRestorePVC(vmRestore, target, backup, restore); err != nil {
				return false, err
			}
			createdPVC = true
		} else if pvc.Status.Phase == corev1.ClaimPending {
			bindingMode, err := ctrl.getBindingMode(pvc)
			if err != nil {
				return false, err
			}

			if bindingMode == nil || *bindingMode == storagev1.VolumeBindingImmediate {
				waitingPVC = true
			}
		} else if pvc.Status.Phase != corev1.ClaimBound {
			return false, fmt.Errorf("PVC %s/%s in status %q", pvc.Namespace, pvc.Name, pvc.Status.Phase)
		}
	}

	return createdPVC || waitingPVC, nil
}

func (ctrl *VMRestoreController) getBindingMode(pvc *corev1.PersistentVolumeClaim) (*storagev1.VolumeBindingMode, error) {
	if pvc.Spec.StorageClassName == nil {
		return nil, nil
	}

	obj, exists, err := ctrl.StorageClassInformer.GetStore().GetByKey(*pvc.Spec.StorageClassName)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("StorageClass %s does not exist", *pvc.Spec.StorageClassName)
	}

	sc := obj.(*storagev1.StorageClass).DeepCopy()

	return sc.VolumeBindingMode, nil
}

func (t *vmRestoreTarget) UID() types.UID {
	return t.vm.UID
}

func (t *vmRestoreTarget) UpdateDoneRestore() (bool, error) {
	if t.vm.Status.RestoreInProgress == nil || *t.vm.Status.RestoreInProgress != t.vmRestore.Name {
		return false, nil
	}

	vmCopy := t.vm.DeepCopy()

	vmCopy.Status.RestoreInProgress = nil
	return true, t.controller.vmStatusUpdater.UpdateStatus(vmCopy)
}

func (t *vmRestoreTarget) UpdateRestoreInProgress() error {
	if t.vm.Status.RestoreInProgress != nil && *t.vm.Status.RestoreInProgress != t.vmRestore.Name {
		return fmt.Errorf("vm restore %s in progress", *t.vm.Status.RestoreInProgress)
	}

	vmCopy := t.vm.DeepCopy()

	if vmCopy.Status.RestoreInProgress == nil {
		vmCopy.Status.RestoreInProgress = &t.vmRestore.Name

		// unfortunately, status Updater does not return the updated resource
		// but the controller is watching VMs so will get notified
		return t.controller.vmStatusUpdater.UpdateStatus(vmCopy)
	}

	return nil
}

func (t *vmRestoreTarget) Ready() (bool, error) {
	log.Log.Object(t.vmRestore).V(3).Info("Checking VM ready")

	rs, err := t.vm.RunStrategy()
	if err != nil {
		return false, err
	}

	if rs != kubevirtv1.RunStrategyHalted {
		return false, fmt.Errorf("invalid RunStrategy %q", rs)
	}

	vmiKey, err := controller.KeyFunc(t.vm)
	if err != nil {
		return false, err
	}

	_, exists, err := t.controller.VMIInformer.GetStore().GetByKey(vmiKey)
	if err != nil {
		return false, err
	}

	return !exists, nil
}

func (t *vmRestoreTarget) Reconcile() (bool, error) {
	log.Log.Object(t.vmRestore).V(3).Info("Reconciling VM")

	restoreID := fmt.Sprintf("%s-%s", t.vmRestore.Name, t.vmRestore.UID)

	if lastRestoreID, ok := t.vm.Annotations[lastRestoreAnnotation]; ok && lastRestoreID == restoreID {
		return false, nil
	}

	content, err := t.controller.getSnapshotContent(t.vmRestore, t.UID())
	if err != nil {
		return false, err
	}

	snapshotVM := content.Spec.Source.VirtualMachine
	if snapshotVM == nil {
		return false, fmt.Errorf("unexpected snapshot source")
	}

	var newTemplates = make([]kubevirtv1.DataVolumeTemplateSpec, len(snapshotVM.Spec.DataVolumeTemplates))
	var newVolumes = make([]kubevirtv1.Volume, len(snapshotVM.Spec.Template.Spec.Volumes))
	var deletedDataVolumes []string
	updatedStatus := false

	for i, t := range snapshotVM.Spec.DataVolumeTemplates {
		t.DeepCopyInto(&newTemplates[i])
	}

	for i, v := range snapshotVM.Spec.Template.Spec.Volumes {
		v.DeepCopyInto(&newVolumes[i])
	}

	for j, v := range snapshotVM.Spec.Template.Spec.Volumes {
		if v.DataVolume != nil || v.PersistentVolumeClaim != nil {
			for k := range t.vmRestore.Status.Restores {
				vr := &t.vmRestore.Status.Restores[k]
				if vr.VolumeName != v.Name {
					continue
				}

				pvc, err := t.controller.getPVC(t.vmRestore.Namespace, vr.PersistentVolumeClaimName)
				if err != nil {
					return false, err
				}

				if pvc == nil {
					return false, fmt.Errorf("pvc %s/%s does not exist and should", t.vmRestore.Namespace, vr.PersistentVolumeClaimName)
				}

				if v.DataVolume != nil {
					templateIndex := -1
					for i, dvt := range snapshotVM.Spec.DataVolumeTemplates {
						if v.DataVolume.Name == dvt.Name {
							templateIndex = i
							break
						}
					}

					if templateIndex >= 0 {
						if vr.DataVolumeName == nil {
							updatePVC := pvc.DeepCopy()
							dvName := restoreDVName(t.vmRestore, vr.VolumeName)

							if updatePVC.Annotations[populatedForPVCAnnotation] != dvName {
								if updatePVC.Annotations == nil {
									updatePVC.Annotations = make(map[string]string)
								}
								updatePVC.Annotations[populatedForPVCAnnotation] = dvName
								// datavolume will take ownership
								updatePVC.OwnerReferences = nil
								_, err = t.controller.Client.CoreV1().PersistentVolumeClaims(updatePVC.Namespace).Update(context.Background(), updatePVC, metav1.UpdateOptions{})
								if err != nil {
									return false, err
								}
							}

							vr.DataVolumeName = &dvName
							updatedStatus = true
						}

						dv := snapshotVM.Spec.DataVolumeTemplates[templateIndex].DeepCopy()
						dv.Name = *vr.DataVolumeName
						newTemplates[templateIndex] = *dv

						nv := v.DeepCopy()
						nv.DataVolume.Name = *vr.DataVolumeName
						newVolumes[j] = *nv
					} else {
						// convert to PersistentVolumeClaim volume
						nv := kubevirtv1.Volume{
							Name: v.Name,
							VolumeSource: kubevirtv1.VolumeSource{
								PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
									PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
										ClaimName: vr.PersistentVolumeClaimName,
									},
								},
							},
						}
						newVolumes[j] = nv
					}
				} else {
					nv := v.DeepCopy()
					nv.PersistentVolumeClaim.ClaimName = vr.PersistentVolumeClaimName
					newVolumes[j] = *nv
				}
			}
		}
	}

	if updatedStatus {
		// find DataVolumes that will no longer exist
		for _, cdv := range t.vm.Spec.DataVolumeTemplates {
			found := false
			for _, ndv := range newTemplates {
				if cdv.Name == ndv.Name {
					found = true
					break
				}
			}
			if !found {
				deletedDataVolumes = append(deletedDataVolumes, cdv.Name)
			}
		}
		t.vmRestore.Status.DeletedDataVolumes = deletedDataVolumes

		return true, nil
	}

	newVM := t.vm.DeepCopy()
	newVM.Spec = snapshotVM.Spec
	// update Running state in case snapshot was on online VM
	if newVM.Spec.RunStrategy != nil {
		runStrategyHalted := kubevirtv1.RunStrategyHalted
		newVM.Spec.RunStrategy = &runStrategyHalted
	} else if newVM.Spec.Running != nil {
		running := false
		newVM.Spec.Running = &running
	}
	newVM.Spec.DataVolumeTemplates = newTemplates
	newVM.Spec.Template.Spec.Volumes = newVolumes
	if newVM.Annotations == nil {
		newVM.Annotations = make(map[string]string)
	}
	newVM.Annotations[lastRestoreAnnotation] = restoreID

	_, err = t.controller.Client.VirtualMachine(newVM.Namespace).Update(newVM)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (t *vmRestoreTarget) Own(obj metav1.Object) {
	b := true
	obj.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         kubevirtv1.GroupVersion.String(),
			Kind:               "VirtualMachine",
			Name:               t.vm.Name,
			UID:                t.vm.UID,
			Controller:         &b,
			BlockOwnerDeletion: &b,
		},
	})
}

func (t *vmRestoreTarget) Cleanup() error {
	for _, dvName := range t.vmRestore.Status.DeletedDataVolumes {
		objKey := cacheKeyFunc(t.vmRestore.Namespace, dvName)
		_, exists, err := t.controller.DataVolumeInformer.GetStore().GetByKey(objKey)
		if err != nil {
			return err
		}

		if exists {
			err = t.controller.Client.CdiClient().CdiV1beta1().DataVolumes(t.vmRestore.Namespace).
				Delete(context.Background(), dvName, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (ctrl *VMRestoreController) getSnapshotContent(vmRestore *snapshotv1.VirtualMachineRestore, targetUID types.UID) (*snapshotv1.VirtualMachineSnapshotContent, error) {
	objKey := cacheKeyFunc(vmRestore.Namespace, vmRestore.Spec.VirtualMachineSnapshotName)
	obj, exists, err := ctrl.VMSnapshotInformer.GetStore().GetByKey(objKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("VMSnapshot %s does not exist", objKey)
	}

	vms := obj.(*snapshotv1.VirtualMachineSnapshot).DeepCopy()
	if !vmSnapshotReady(vms) {
		return nil, fmt.Errorf("VMSnapshot %s not ready", objKey)
	}

	if vms.Status.SourceUID == nil || *vms.Status.SourceUID != targetUID {
		return nil, fmt.Errorf("VMSnapshot source and restore target differ")
	}

	if vms.Status.VirtualMachineSnapshotContentName == nil {
		return nil, fmt.Errorf("no snapshot content name in %s", objKey)
	}

	objKey = cacheKeyFunc(vmRestore.Namespace, *vms.Status.VirtualMachineSnapshotContentName)
	obj, exists, err = ctrl.VMSnapshotContentInformer.GetStore().GetByKey(objKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("VMSnapshotContent %s does not exist", objKey)
	}

	vmss := obj.(*snapshotv1.VirtualMachineSnapshotContent).DeepCopy()
	if !vmSnapshotContentReady(vmss) {
		return nil, fmt.Errorf("VMSnapshotContent %s not ready", objKey)
	}

	return vmss, nil
}

func (ctrl *VMRestoreController) getVM(namespace, name string) (*kubevirtv1.VirtualMachine, error) {
	objKey := cacheKeyFunc(namespace, name)
	obj, exists, err := ctrl.VMInformer.GetStore().GetByKey(objKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("VirtualMachine %s/%s does not exist", namespace, name)
	}

	return obj.(*kubevirtv1.VirtualMachine).DeepCopy(), nil
}

func (ctrl *VMRestoreController) getPVC(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	objKey := cacheKeyFunc(namespace, name)
	obj, exists, err := ctrl.PVCInformer.GetStore().GetByKey(objKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	return obj.(*corev1.PersistentVolumeClaim).DeepCopy(), nil
}

func (ctrl *VMRestoreController) getTarget(vmRestore *snapshotv1.VirtualMachineRestore) (restoreTarget, error) {
	vmRestore.Spec.Target.DeepCopy()
	switch vmRestore.Spec.Target.Kind {
	case "VirtualMachine":
		vm, err := ctrl.getVM(vmRestore.Namespace, vmRestore.Spec.Target.Name)
		if err != nil {
			return nil, err
		}

		return &vmRestoreTarget{
			controller: ctrl,
			vmRestore:  vmRestore,
			vm:         vm,
		}, nil
	}

	return nil, fmt.Errorf("unknown source %+v", vmRestore.Spec.Target)
}

func (ctrl *VMRestoreController) createRestorePVC(
	vmRestore *snapshotv1.VirtualMachineRestore,
	target restoreTarget,
	volumeBackup snapshotv1.VolumeBackup,
	volumeRestore snapshotv1.VolumeRestore,
) error {
	sourcePVC := volumeBackup.PersistentVolumeClaim.DeepCopy()
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        volumeRestore.PersistentVolumeClaimName,
			Labels:      sourcePVC.Labels,
			Annotations: sourcePVC.Annotations,
		},
		Spec: sourcePVC.Spec,
	}

	if volumeBackup.VolumeSnapshotName == nil {
		log.Log.Errorf("VolumeSnapshot name missing %+v", volumeBackup)
		return fmt.Errorf("missing VolumeSnapshot name")
	}

	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}

	for _, prefix := range restoreAnnotationsToDelete {
		for anno := range pvc.Annotations {
			if strings.HasPrefix(anno, prefix) {
				delete(pvc.Annotations, anno)
			}
		}
	}
	pvc.Annotations[pvcRestoreAnnotation] = vmRestore.Name

	apiGroup := vsv1beta1.GroupName
	pvc.Spec.DataSource = &corev1.TypedLocalObjectReference{
		APIGroup: &apiGroup,
		Kind:     "VolumeSnapshot",
		Name:     *volumeBackup.VolumeSnapshotName,
	}
	pvc.Spec.VolumeName = ""

	target.Own(pvc)

	_, err := ctrl.Client.CoreV1().PersistentVolumeClaims(vmRestore.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func updateRestoreCondition(r *snapshotv1.VirtualMachineRestore, c snapshotv1.Condition) {
	r.Status.Conditions = updateCondition(r.Status.Conditions, c, true)
}
