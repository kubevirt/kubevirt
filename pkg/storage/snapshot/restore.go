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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	vsv1 "github.com/kubernetes-csi/external-snapshotter/client/v4/apis/volumesnapshot/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/instancetype"
	"kubevirt.io/kubevirt/pkg/pointer"
	typesutil "kubevirt.io/kubevirt/pkg/storage/types"
	storageutils "kubevirt.io/kubevirt/pkg/storage/utils"
)

const (
	RestoreNameAnnotation = "restore.kubevirt.io/name"

	vmRestoreFinalizer = "snapshot.kubevirt.io/vmrestore-protection"

	populatedForPVCAnnotation = "cdi.kubevirt.io/storage.populatedFor"

	lastRestoreAnnotation = "restore.kubevirt.io/lastRestoreUID"

	restoreSourceNameLabel = "restore.kubevirt.io/source-vm-name"

	restoreSourceNamespaceLabel = "restore.kubevirt.io/source-vm-namespace"

	restoreCompleteEvent = "VirtualMachineRestoreComplete"

	restoreErrorEvent = "VirtualMachineRestoreError"

	restoreVMNotReadyEvent = "RestoreTargetNotReady"

	restoreDataVolumeCreateErrorEvent = "RestoreDataVolumeCreateError"
)

var (
	restoreGracePeriodExceededError = fmt.Sprintf("Restore target failed to be ready within %s", snapshotv1.DefaultGracePeriod)
	restoreTargetNotReady           = "Restore target not ready"
	restoredFailed                  = "Operation failed"
	errorRestoreToExistingTarget    = "restore source and restore target are different but restore target already exists"
)

type restoreTarget interface {
	Stop() error
	Ready() (bool, error)
	Reconcile() (bool, error)
	Own(obj metav1.Object)
	UpdateDoneRestore() error
	UpdateRestoreInProgress() error
	UpdateTarget(obj metav1.Object)
	Exists() bool
	UID() types.UID
	TargetRestored() bool
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
	"volume.kubernetes.io",
	"k8s.io/CloneRequest",
	"k8s.io/CloneOf",
}

func restorePVCName(vmRestore *snapshotv1.VirtualMachineRestore, name string) string {
	return fmt.Sprintf("restore-%s-%s", vmRestore.UID, name)
}

func restoreDVName(vmRestore *snapshotv1.VirtualMachineRestore, name string) string {
	return restorePVCName(vmRestore, name)
}

func vmRestoreFailed(vmRestore *snapshotv1.VirtualMachineRestore) bool {
	return vmRestore.Status != nil &&
		hasConditionType(vmRestore.Status.Conditions, snapshotv1.ConditionFailure)
}

func vmRestoreCompleted(vmRestore *snapshotv1.VirtualMachineRestore) bool {
	return vmRestore.Status != nil && vmRestore.Status.Complete != nil && *vmRestore.Status.Complete
}

func VmRestoreProgressing(vmRestore *snapshotv1.VirtualMachineRestore) bool {
	return !vmRestoreCompleted(vmRestore) && !vmRestoreFailed(vmRestore)
}

func vmRestoreDeleting(vmRestore *snapshotv1.VirtualMachineRestore) bool {
	return vmRestore != nil && vmRestore.DeletionTimestamp != nil
}

func (ctrl *VMRestoreController) updateVMRestore(vmRestoreIn *snapshotv1.VirtualMachineRestore) (time.Duration, error) {
	logger := log.Log.Object(vmRestoreIn)
	logger.V(1).Infof("Updating VirtualMachineRestore")

	vmRestoreOut := vmRestoreIn.DeepCopy()

	if vmRestoreOut.Status == nil {
		vmRestoreOut.Status = &snapshotv1.VirtualMachineRestoreStatus{
			Complete: pointer.P(false),
		}
		updateRestoreCondition(vmRestoreOut, newProgressingCondition(corev1.ConditionTrue, "Initializing VirtualMachineRestore"))
		updateRestoreCondition(vmRestoreOut, newReadyCondition(corev1.ConditionFalse, "Initializing VirtualMachineRestore"))
	}

	// let's make sure everything is initialized properly before continuing
	if !equality.Semantic.DeepEqual(vmRestoreIn.Status, vmRestoreOut.Status) {
		return 0, ctrl.doUpdateStatus(vmRestoreIn, vmRestoreOut)
	}

	target, err := ctrl.getTarget(vmRestoreOut)
	if err != nil {
		logger.Reason(err).Error("Error getting restore target")
		return 0, ctrl.doUpdateError(vmRestoreOut, err)
	}

	if vmRestoreDeleting(vmRestoreOut) {
		return 0, ctrl.handleVMRestoreDeletion(vmRestoreOut, target)
	}

	if !VmRestoreProgressing(vmRestoreOut) {
		return 0, nil
	}

	if len(vmRestoreOut.OwnerReferences) == 0 {
		target.Own(vmRestoreOut)
	}
	controller.AddFinalizer(vmRestoreOut, vmRestoreFinalizer)

	if !equality.Semantic.DeepEqual(vmRestoreIn.ObjectMeta, vmRestoreOut.ObjectMeta) {
		vmRestoreOut, err = ctrl.Client.VirtualMachineRestore(vmRestoreOut.Namespace).Update(context.Background(), vmRestoreOut, metav1.UpdateOptions{})
		if err != nil {
			logger.Reason(err).Error("Error updating owner references")
			return 0, err
		}
	}

	ready, err := target.Ready()
	if err != nil {
		logger.Reason(err).Error("Error checking target ready")
		return 0, ctrl.doUpdateError(vmRestoreIn, err)
	}
	if !ready {
		return 0, ctrl.handleVMRestoreTargetNotReady(vmRestoreOut, target)
	}

	vmSnapshot, err := ctrl.getVMSnapshot(vmRestoreOut)
	if err != nil {
		return 0, ctrl.doUpdateError(vmRestoreIn, err)
	}

	// Check if target exists before the restore
	// and that it is not the same as the source
	// We do not allow restoring to an existing
	// target which is not the same as the source
	if target.Exists() && !target.TargetRestored() && sourceAndTargetAreDifferent(target, vmSnapshot) {
		logger.Error(errorRestoreToExistingTarget)
		return 0, ctrl.doUpdateError(vmRestoreIn, fmt.Errorf(errorRestoreToExistingTarget))
	}

	err = target.UpdateRestoreInProgress()
	if err != nil {
		return 0, err
	}

	updated, err := ctrl.reconcileVolumeRestores(vmRestoreOut, target, vmSnapshot)
	if err != nil {
		logger.Reason(err).Error("Error reconciling VolumeRestores")
		return 0, ctrl.doUpdateError(vmRestoreIn, err)
	}
	if updated {
		updateRestoreCondition(vmRestoreOut, newProgressingCondition(corev1.ConditionTrue, "Creating new PVCs"))
		updateRestoreCondition(vmRestoreOut, newReadyCondition(corev1.ConditionFalse, "Waiting for new PVCs"))
		return 0, ctrl.doUpdateStatus(vmRestoreIn, vmRestoreOut)
	}

	updated, err = target.Reconcile()
	if err != nil {
		logger.Reason(err).Error("Error reconciling target")
		return 0, ctrl.doUpdateError(vmRestoreIn, err)
	}
	if updated {
		updateRestoreCondition(vmRestoreOut, newProgressingCondition(corev1.ConditionTrue, "Updating target spec"))
		updateRestoreCondition(vmRestoreOut, newReadyCondition(corev1.ConditionFalse, "Waiting for target update"))
		return 0, ctrl.doUpdateStatus(vmRestoreIn, vmRestoreOut)
	}

	if err = ctrl.deleteObsoleteDataVolumes(vmRestoreOut); err != nil {
		logger.Reason(err).Error("Error cleaning up")
		return 0, ctrl.doUpdateError(vmRestoreIn, err)
	}

	err = target.UpdateDoneRestore()
	if err != nil {
		logger.Reason(err).Error("Error updating done restore")
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

	return 0, ctrl.doUpdateStatus(vmRestoreIn, vmRestoreOut)
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
	if err2 := ctrl.doUpdateStatus(restore, updated); err2 != nil {
		return err2
	}

	return err
}

func (ctrl *VMRestoreController) doUpdateStatus(original, updated *snapshotv1.VirtualMachineRestore) error {
	if !equality.Semantic.DeepEqual(original.Status, updated.Status) {
		if err := ctrl.VMRestoreStatusUpdater.UpdateStatus(updated); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *VMRestoreController) handleVMRestoreDeletion(vmRestore *snapshotv1.VirtualMachineRestore, target restoreTarget) error {
	logger := log.Log.Object(vmRestore)
	logger.V(3).Infof("Handling deleted VirtualMachineRestore")

	if !controller.HasFinalizer(vmRestore, vmRestoreFinalizer) {
		return nil
	}

	vmRestoreCpy := vmRestore.DeepCopy()
	if target.Exists() {
		err := target.UpdateDoneRestore()
		if err != nil {
			logger.Reason(err).Error("Error updating done restore")
			return ctrl.doUpdateError(vmRestoreCpy, err)
		}
	}

	updateRestoreCondition(vmRestoreCpy, newProgressingCondition(corev1.ConditionFalse, "VM restore is deleting"))
	updateRestoreCondition(vmRestoreCpy, newReadyCondition(corev1.ConditionFalse, "VM restore is deleting"))
	if !equality.Semantic.DeepEqual(vmRestore.Status, vmRestoreCpy.Status) {
		return ctrl.doUpdateStatus(vmRestore, vmRestoreCpy)
	}

	controller.RemoveFinalizer(vmRestoreCpy, vmRestoreFinalizer)
	patch, err := generateFinalizerPatch(vmRestore.Finalizers, vmRestoreCpy.Finalizers)
	if err != nil {
		return err
	}
	_, err = ctrl.Client.VirtualMachineRestore(vmRestore.Namespace).Patch(context.Background(), vmRestore.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	return err
}

func (ctrl *VMRestoreController) handleVMRestoreTargetNotReady(vmRestore *snapshotv1.VirtualMachineRestore, target restoreTarget) error {
	vmRestoreCpy := vmRestore.DeepCopy()

	// default policy is having a grace period for the user the make
	// the target ready
	policy := snapshotv1.VirtualMachineRestoreWaitGracePeriodAndFail
	if vmRestore.Spec.TargetReadinessPolicy != nil {
		policy = *vmRestore.Spec.TargetReadinessPolicy
	}

	eventMsg := "Restore target VMI still exists"
	ctrl.Recorder.Event(vmRestoreCpy, corev1.EventTypeNormal, restoreVMNotReadyEvent, eventMsg)
	reason := "Waiting for target to be ready"
	updateRestoreCondition(vmRestoreCpy, newProgressingCondition(corev1.ConditionFalse, reason))
	updateRestoreCondition(vmRestoreCpy, newReadyCondition(corev1.ConditionFalse, reason))

	switch policy {
	case snapshotv1.VirtualMachineRestoreWaitEventually:
	case snapshotv1.VirtualMachineRestoreStopTarget:
		err := target.Stop()
		if err != nil {
			return ctrl.doUpdateError(vmRestore, err)
		}
	case snapshotv1.VirtualMachineRestoreWaitGracePeriodAndFail:
		if vmRestoreTargetReadyGracePeriodExceeded(vmRestore) {
			updateRestoreCondition(vmRestoreCpy, newProgressingCondition(corev1.ConditionFalse, restoredFailed))
			updateRestoreCondition(vmRestoreCpy, newFailureCondition(corev1.ConditionTrue, restoreGracePeriodExceededError))
			updateRestoreCondition(vmRestoreCpy, newReadyCondition(corev1.ConditionFalse, restoredFailed))
		}
	case snapshotv1.VirtualMachineRestoreFailImmediate:
		updateRestoreCondition(vmRestoreCpy, newProgressingCondition(corev1.ConditionFalse, restoredFailed))
		updateRestoreCondition(vmRestoreCpy, newFailureCondition(corev1.ConditionTrue, restoreTargetNotReady))
		updateRestoreCondition(vmRestoreCpy, newReadyCondition(corev1.ConditionFalse, restoredFailed))
	}
	return ctrl.doUpdateStatus(vmRestore, vmRestoreCpy)
}

func vmRestoreTargetReadyGracePeriodExceeded(vmRestore *snapshotv1.VirtualMachineRestore) bool {
	deadline := vmRestore.CreationTimestamp.Add(snapshotv1.DefaultGracePeriod)
	return time.Until(deadline) < 0
}

func (ctrl *VMRestoreController) reconcileVolumeRestores(vmRestore *snapshotv1.VirtualMachineRestore, target restoreTarget, vmSnapshot *snapshotv1.VirtualMachineSnapshot) (bool, error) {
	content, err := ctrl.getSnapshotContent(vmSnapshot)
	if err != nil {
		return false, err
	}

	noRestore, err := ctrl.volumesNotForRestore(content)
	if err != nil {
		return false, err
	}

	var restores []snapshotv1.VolumeRestore
	for _, vb := range content.Spec.VolumeBackups {
		if noRestore.Has(vb.VolumeName) {
			continue
		}

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

	if !equality.Semantic.DeepEqual(vmRestore.Status.Restores, restores) {
		if len(vmRestore.Status.Restores) > 0 {
			log.Log.Object(vmRestore).Warning("VMRestore in strange state")
		}

		vmRestore.Status.Restores = restores
		return true, nil
	}

	createdPVC := false
	waitingPVC := false
	for _, restore := range restores {
		pvc, err := ctrl.getPVC(vmRestore.Namespace, restore.PersistentVolumeClaimName)
		if err != nil {
			return false, err
		}

		if pvc == nil {
			backup, err := getRestoreVolumeBackup(restore.VolumeName, content)
			if err != nil {
				return false, err
			}
			if err = ctrl.createRestorePVC(vmRestore, target, backup, &restore, content.Spec.Source.VirtualMachine.Name, content.Spec.Source.VirtualMachine.Namespace); err != nil {
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

func (t *vmRestoreTarget) UpdateDoneRestore() error {
	if !t.Exists() {
		return fmt.Errorf("At this point target should exist")
	}

	if t.vm.Status.RestoreInProgress == nil || *t.vm.Status.RestoreInProgress != t.vmRestore.Name {
		return nil
	}

	vmCopy := t.vm.DeepCopy()

	vmCopy.Status.RestoreInProgress = nil
	vmCopy.Status.MemoryDumpRequest = nil
	vmCopy, err := t.controller.Client.VirtualMachine(vmCopy.Namespace).UpdateStatus(context.Background(), vmCopy, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	t.vm = vmCopy
	return nil
}

func (t *vmRestoreTarget) UpdateRestoreInProgress() error {
	if !t.Exists() || hasLastRestoreAnnotation(t.vmRestore, t.vm) {
		return nil
	}

	if t.vm.Status.RestoreInProgress != nil && *t.vm.Status.RestoreInProgress != t.vmRestore.Name {
		return fmt.Errorf("vm restore %s in progress", *t.vm.Status.RestoreInProgress)
	}

	vmCopy := t.vm.DeepCopy()

	if vmCopy.Status.RestoreInProgress == nil {
		vmCopy.Status.RestoreInProgress = &t.vmRestore.Name

		var err error
		vmCopy, err = t.controller.Client.VirtualMachine(vmCopy.Namespace).UpdateStatus(context.Background(), vmCopy, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	t.vm = vmCopy

	return nil
}

func (t *vmRestoreTarget) Stop() error {
	if !t.Exists() {
		return nil
	}

	log.Log.Infof("Stopping VM before restore [%s/%s]", t.vm.Namespace, t.vm.Name)
	return t.controller.Client.VirtualMachine(t.vm.Namespace).Stop(context.Background(), t.vm.Name, &kubevirtv1.StopOptions{})
}

func (t *vmRestoreTarget) Ready() (bool, error) {
	if !t.Exists() {
		return true, nil
	}

	log.Log.Object(t.vmRestore).V(3).Info("Checking VM ready")

	vmiKey, err := controller.KeyFunc(t.vm)
	if err != nil {
		return false, err
	}

	_, exists, err := t.controller.VMIInformer.GetStore().GetByKey(vmiKey)

	return !exists, err
}

func (t *vmRestoreTarget) Reconcile() (bool, error) {
	if t.Exists() && hasLastRestoreAnnotation(t.vmRestore, t.vm) {
		return false, nil
	}
	snapshotVM, err := t.getSnapshotVM()
	if err != nil {
		return false, err
	}

	if updated, err := t.updateVMRestoreRestores(snapshotVM); updated || err != nil {
		return updated, err
	}

	restoredVM, err := t.generateRestoredVMSpec(snapshotVM)
	if err != nil {
		return false, err
	}
	if updated, err := t.reconcileDataVolumes(restoredVM); updated || err != nil {
		return updated, err
	}

	return t.reconcileSpec(restoredVM)
}

func (t *vmRestoreTarget) getSnapshotVM() (*snapshotv1.VirtualMachine, error) {
	vmSnapshot, err := t.controller.getVMSnapshot(t.vmRestore)
	if err != nil {
		return nil, err
	}

	content, err := t.controller.getSnapshotContent(vmSnapshot)
	if err != nil {
		return nil, err
	}

	snapshotVM := content.Spec.Source.VirtualMachine
	if snapshotVM == nil {
		return nil, fmt.Errorf("unexpected snapshot source")
	}

	return snapshotVM, nil
}

func (t *vmRestoreTarget) updateVMRestoreRestores(snapshotVM *snapshotv1.VirtualMachine) (bool, error) {
	var restores = make([]snapshotv1.VolumeRestore, len(t.vmRestore.Status.Restores))
	for i, t := range t.vmRestore.Status.Restores {
		t.DeepCopyInto(&restores[i])
	}
	for k := range restores {
		restore := &restores[k]
		volumes, err := storageutils.GetVolumes(snapshotVM, t.controller.Client, storageutils.WithBackendVolume)
		if err != nil {
			return false, err
		}
		for _, volume := range volumes {
			if volume.Name != restore.VolumeName {
				continue
			}
			if volume.DataVolume != nil {
				templateIndex := findDVTemplateIndex(volume.DataVolume.Name, snapshotVM)
				if templateIndex >= 0 {
					dvName := restoreDVName(t.vmRestore, restore.VolumeName)
					pvc, err := t.controller.getPVC(t.vmRestore.Namespace, restore.PersistentVolumeClaimName)
					if err != nil {
						return false, err
					}

					if pvc == nil {
						return false, fmt.Errorf("pvc %s/%s does not exist and should", t.vmRestore.Namespace, restore.PersistentVolumeClaimName)
					}

					if err = t.updatePVCPopulatedForAnnotation(pvc, dvName); err != nil {
						return false, err
					}
					restore.DataVolumeName = &dvName
					break
				}
			}
		}
	}
	if !equality.Semantic.DeepEqual(t.vmRestore.Status.Restores, restores) {
		t.vmRestore.Status.Restores = restores
		return true, nil
	}
	return false, nil
}

func (t *vmRestoreTarget) UpdateTarget(obj metav1.Object) {
	t.vm = obj.(*kubevirtv1.VirtualMachine)
}

func (t *vmRestoreTarget) generateRestoredVMSpec(snapshotVM *snapshotv1.VirtualMachine) (*kubevirtv1.VirtualMachine, error) {
	log.Log.Object(t.vmRestore).V(3).Info("generating restored VM spec")
	var newTemplates = make([]kubevirtv1.DataVolumeTemplateSpec, len(snapshotVM.Spec.DataVolumeTemplates))
	var newVolumes []kubevirtv1.Volume

	for i, t := range snapshotVM.Spec.DataVolumeTemplates {
		t.DeepCopyInto(&newTemplates[i])
	}

	volumes, err := storageutils.GetVolumes(snapshotVM, t.controller.Client, storageutils.WithBackendVolume)
	if err != nil {
		return nil, err
	}

	for _, v := range volumes {
		nv := v.DeepCopy()
		if nv.DataVolume != nil || nv.PersistentVolumeClaim != nil {
			for _, vr := range t.vmRestore.Status.Restores {
				if vr.VolumeName != nv.Name {
					continue
				}

				if nv.DataVolume == nil {
					nv.PersistentVolumeClaim.ClaimName = vr.PersistentVolumeClaimName
					continue
				}

				templateIndex := findDVTemplateIndex(v.DataVolume.Name, snapshotVM)
				if templateIndex >= 0 {
					if vr.DataVolumeName == nil {
						return nil, fmt.Errorf("DataVolumeName for dv %s should have been updated already", v.DataVolume.Name)
					}

					dv := snapshotVM.Spec.DataVolumeTemplates[templateIndex].DeepCopy()
					dv.Name = *vr.DataVolumeName
					newTemplates[templateIndex] = *dv

					nv.DataVolume.Name = *vr.DataVolumeName
				} else {
					// convert to PersistentVolumeClaim volume
					nv = &kubevirtv1.Volume{
						Name: nv.Name,
						VolumeSource: kubevirtv1.VolumeSource{
							PersistentVolumeClaim: &kubevirtv1.PersistentVolumeClaimVolumeSource{
								PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: vr.PersistentVolumeClaimName,
								},
							},
						},
					}
				}
			}
		} else if nv.MemoryDump != nil {
			// don't restore memory dump volume in the new spec
			continue
		}
		newVolumes = append(newVolumes, *nv)
	}

	var newVM *kubevirtv1.VirtualMachine
	if !t.Exists() {
		newVM = &kubevirtv1.VirtualMachine{
			ObjectMeta: metav1.ObjectMeta{
				Name:        t.vmRestore.Spec.Target.Name,
				Namespace:   t.vmRestore.Namespace,
				Labels:      snapshotVM.Labels,
				Annotations: snapshotVM.Annotations,
			},
			Spec:   *snapshotVM.Spec.DeepCopy(),
			Status: kubevirtv1.VirtualMachineStatus{},
		}

	} else {
		newVM = t.vm.DeepCopy()
		newVM.Spec = *snapshotVM.Spec.DeepCopy()
	}

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
	setLastRestoreAnnotation(t.vmRestore, newVM)

	return newVM, nil
}

func (t *vmRestoreTarget) reconcileSpec(restoredVM *kubevirtv1.VirtualMachine) (bool, error) {
	log.Log.Object(t.vmRestore).V(3).Info("Reconcile new VM spec")

	var err error
	if err = t.restoreInstancetypeControllerRevisions(restoredVM); err != nil {
		return false, err
	}

	if !t.Exists() {
		restoredVM, err = patchVM(restoredVM, t.vmRestore.Spec.Patches)
		if err != nil {
			return false, fmt.Errorf("error patching VM %s: %v", restoredVM.Name, err)
		}
		restoredVM, err = t.controller.Client.VirtualMachine(t.vmRestore.Namespace).Create(context.Background(), restoredVM, metav1.CreateOptions{})
	} else {
		restoredVM, err = t.controller.Client.VirtualMachine(restoredVM.Namespace).Update(context.Background(), restoredVM, metav1.UpdateOptions{})
	}
	if err != nil {
		return false, err
	}

	t.UpdateTarget(restoredVM)

	if err = t.claimInstancetypeControllerRevisionsOwnership(t.vm); err != nil {
		return false, err
	}

	return true, nil
}

func findDVTemplateIndex(dvName string, vm *snapshotv1.VirtualMachine) int {
	templateIndex := -1
	for i, dvt := range vm.Spec.DataVolumeTemplates {
		if dvName == dvt.Name {
			templateIndex = i
			break
		}
	}
	return templateIndex
}

func (t *vmRestoreTarget) updatePVCPopulatedForAnnotation(pvc *corev1.PersistentVolumeClaim, dvName string) error {
	updatePVC := pvc.DeepCopy()
	if updatePVC.Annotations[populatedForPVCAnnotation] != dvName {
		if updatePVC.Annotations == nil {
			updatePVC.Annotations = make(map[string]string)
		}
		updatePVC.Annotations[populatedForPVCAnnotation] = dvName
		// datavolume will take ownership
		updatePVC.OwnerReferences = nil
		_, err := t.controller.Client.CoreV1().PersistentVolumeClaims(updatePVC.Namespace).Update(context.Background(), updatePVC, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// findDatavolumesForDeletion find DataVolumes that will no longer
// exist after the vmrestore is completed
func findDatavolumesForDeletion(oldDVTemplates, newDVTemplates []kubevirtv1.DataVolumeTemplateSpec) []string {
	var deletedDataVolumes []string
	for _, cdv := range oldDVTemplates {
		found := false
		for _, ndv := range newDVTemplates {
			if cdv.Name == ndv.Name {
				found = true
				break
			}
		}
		if !found {
			deletedDataVolumes = append(deletedDataVolumes, cdv.Name)
		}
	}
	return deletedDataVolumes
}

func (t *vmRestoreTarget) reconcileDataVolumes(restoredVM *kubevirtv1.VirtualMachine) (bool, error) {
	createdDV := false
	waitingDV := false
	for _, dvt := range restoredVM.Spec.DataVolumeTemplates {
		dv, err := t.controller.getDV(restoredVM.Namespace, dvt.Name)
		if err != nil {
			return false, err
		}
		if dv != nil {
			waitingDV = waitingDV ||
				(dv.Status.Phase != cdiv1.Succeeded &&
					dv.Status.Phase != cdiv1.WaitForFirstConsumer &&
					dv.Status.Phase != cdiv1.PendingPopulation)
			continue
		}
		created, err := t.createDataVolume(restoredVM, dvt)
		if err != nil {
			return false, err
		}
		createdDV = createdDV || created
	}

	if t.Exists() {
		deletedDataVolumes := findDatavolumesForDeletion(t.vm.Spec.DataVolumeTemplates, restoredVM.Spec.DataVolumeTemplates)
		if !equality.Semantic.DeepEqual(t.vmRestore.Status.DeletedDataVolumes, deletedDataVolumes) {
			t.vmRestore.Status.DeletedDataVolumes = deletedDataVolumes

			return true, nil
		}
	}

	return createdDV || waitingDV, nil
}

func (t *vmRestoreTarget) getControllerRevision(namespace, name string) (*appsv1.ControllerRevision, error) {
	revisionKey := cacheKeyFunc(namespace, name)
	obj, exists, err := t.controller.CRInformer.GetStore().GetByKey(revisionKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Unable to find ControllerRevision %s", revisionKey)
	}
	return obj.(*appsv1.ControllerRevision), nil
}

func (t *vmRestoreTarget) restoreInstancetypeControllerRevision(vmSnapshotRevisionName, vmSnapshotName string, vm *kubevirtv1.VirtualMachine) (*appsv1.ControllerRevision, error) {
	snapshotCR, err := t.getControllerRevision(vm.Namespace, vmSnapshotRevisionName)
	if err != nil {
		return nil, err
	}

	// Switch the snapshot and vm names for the restored CR name
	restoredCRName := strings.Replace(vmSnapshotRevisionName, vmSnapshotName, vm.Name, 1)
	restoredCR := snapshotCR.DeepCopy()
	restoredCR.ObjectMeta.Reset()
	restoredCR.ObjectMeta.SetLabels(snapshotCR.Labels)
	restoredCR.Name = restoredCRName

	// If the target VirtualMachine already exists it's likely that the original ControllerRevision is already present.
	// Check that here by attempting to lookup the CR using the generated restoredCRName.
	// Ignore any NotFound errors raised allowing the CR to be restored below.
	if t.Exists() {
		existingCR, err := t.getControllerRevision(vm.Namespace, restoredCRName)
		if err != nil && !errors.IsNotFound(err) {
			return nil, err
		}
		if existingCR != nil {
			// Ensure that the existing CR contains the expected data from the snapshot before returning it
			equal, err := instancetype.CompareRevisions(snapshotCR, existingCR)
			if err != nil {
				return nil, err
			}
			if equal {
				return existingCR, nil
			}
			// Otherwise as CRs are immutable delete the existing CR so we can restore the version from the snapshot below
			if err := t.controller.Client.AppsV1().ControllerRevisions(vm.Namespace).Delete(context.Background(), existingCR.Name, metav1.DeleteOptions{}); err != nil {
				return nil, err
			}
			// As the VirtualMachine already exists here we can also populate the OwnerReference, avoiding the need to do so later during claimInstancetypeControllerRevisionOwnership
			restoredCR.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(vm, kubevirtv1.VirtualMachineGroupVersionKind)}
		}
	}

	restoredCR, err = t.controller.Client.AppsV1().ControllerRevisions(vm.Namespace).Create(context.Background(), restoredCR, metav1.CreateOptions{})
	// This might not be our first time through the reconcile loop so accommodate previous calls to restoreInstancetypeControllerRevision by ignoring unexpected existing CRs for now.
	// TODO - Check the contents of the existing CR here against that of the snapshot CR
	if err != nil && !errors.IsAlreadyExists(err) {
		return nil, err
	}

	return restoredCR, nil
}

func (t *vmRestoreTarget) restoreInstancetypeControllerRevisions(vm *kubevirtv1.VirtualMachine) error {
	if vm.Spec.Instancetype != nil && vm.Spec.Instancetype.RevisionName != "" {
		restoredCR, err := t.restoreInstancetypeControllerRevision(vm.Spec.Instancetype.RevisionName, t.vmRestore.Spec.VirtualMachineSnapshotName, vm)
		if err != nil {
			return err
		}
		vm.Spec.Instancetype.RevisionName = restoredCR.Name
	}

	if vm.Spec.Preference != nil && vm.Spec.Preference.RevisionName != "" {
		restoredCR, err := t.restoreInstancetypeControllerRevision(vm.Spec.Preference.RevisionName, t.vmRestore.Spec.VirtualMachineSnapshotName, vm)
		if err != nil {
			return err
		}
		vm.Spec.Preference.RevisionName = restoredCR.Name
	}

	return nil
}

func (t *vmRestoreTarget) claimInstancetypeControllerRevisionOwnership(revisionName string, vm *kubevirtv1.VirtualMachine) error {
	cr, err := t.getControllerRevision(vm.Namespace, revisionName)
	if err != nil {
		return err
	}

	if !metav1.IsControlledBy(cr, vm) {
		cr.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(vm, kubevirtv1.VirtualMachineGroupVersionKind)}
		_, err = t.controller.Client.AppsV1().ControllerRevisions(vm.Namespace).Update(context.Background(), cr, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *vmRestoreTarget) claimInstancetypeControllerRevisionsOwnership(vm *kubevirtv1.VirtualMachine) error {
	if vm.Spec.Instancetype != nil && vm.Spec.Instancetype.RevisionName != "" {
		if err := t.claimInstancetypeControllerRevisionOwnership(vm.Spec.Instancetype.RevisionName, vm); err != nil {
			return err
		}
	}

	if vm.Spec.Preference != nil && vm.Spec.Preference.RevisionName != "" {
		if err := t.claimInstancetypeControllerRevisionOwnership(vm.Spec.Preference.RevisionName, vm); err != nil {
			return err
		}
	}

	return nil
}

func (t *vmRestoreTarget) createDataVolume(restoredVM *kubevirtv1.VirtualMachine, dvt kubevirtv1.DataVolumeTemplateSpec) (bool, error) {
	pvc, err := t.controller.getPVC(restoredVM.Namespace, dvt.Name)
	if err != nil {
		return false, err
	}
	if pvc == nil {
		return false, fmt.Errorf("when creating restore dv pvc %s/%s does not exist and should",
			t.vmRestore.Namespace, dvt.Name)
	}
	if pvc.Annotations[populatedForPVCAnnotation] != dvt.Name || len(pvc.OwnerReferences) > 0 {
		return false, nil
	}

	newDataVolume, err := typesutil.GenerateDataVolumeFromTemplate(t.controller.Client, dvt, restoredVM.Namespace, restoredVM.Spec.Template.Spec.PriorityClassName)
	if err != nil {
		return false, fmt.Errorf("Unable to create restore DataVolume manifest: %v", err)
	}

	if newDataVolume.Annotations == nil {
		newDataVolume.Annotations = make(map[string]string)
	}
	newDataVolume.Annotations[RestoreNameAnnotation] = t.vmRestore.Name
	newDataVolume.Annotations[cdiv1.AnnPrePopulated] = "true"

	if _, err = t.controller.Client.CdiClient().CdiV1beta1().DataVolumes(restoredVM.Namespace).Create(context.Background(), newDataVolume, metav1.CreateOptions{}); err != nil {
		t.controller.Recorder.Eventf(t.vm, corev1.EventTypeWarning, restoreDataVolumeCreateErrorEvent, "Error creating restore DataVolume %s: %v", newDataVolume.Name, err)
		return false, fmt.Errorf("Failed to create restore DataVolume: %v", err)
	}

	return true, nil
}

func (t *vmRestoreTarget) Own(obj metav1.Object) {
	if !t.Exists() {
		return
	}

	obj.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         kubevirtv1.GroupVersion.String(),
			Kind:               "VirtualMachine",
			Name:               t.vm.Name,
			UID:                t.vm.UID,
			Controller:         pointer.P(true),
			BlockOwnerDeletion: pointer.P(true),
		},
	})

	return
}

func (ctrl *VMRestoreController) deleteObsoleteDataVolumes(vmRestore *snapshotv1.VirtualMachineRestore) error {
	for _, dvName := range vmRestore.Status.DeletedDataVolumes {
		objKey := cacheKeyFunc(vmRestore.Namespace, dvName)
		_, exists, err := ctrl.DataVolumeInformer.GetStore().GetByKey(objKey)
		if err != nil {
			return err
		}

		if exists {
			err = ctrl.Client.CdiClient().CdiV1beta1().DataVolumes(vmRestore.Namespace).
				Delete(context.Background(), dvName, metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (t *vmRestoreTarget) TargetRestored() bool {
	return t.Exists() && hasLastRestoreAnnotation(t.vmRestore, t.vm)
}
func (t *vmRestoreTarget) UID() types.UID {
	return t.vm.UID
}

func (t *vmRestoreTarget) Exists() bool {
	return t.vm != nil
}

func sourceAndTargetAreDifferent(target restoreTarget, vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	targetUID := target.UID()
	return vmSnapshot.Status != nil && vmSnapshot.Status.SourceUID != nil && targetUID != *vmSnapshot.Status.SourceUID
}

func (ctrl *VMRestoreController) getVMSnapshot(vmRestore *snapshotv1.VirtualMachineRestore) (*snapshotv1.VirtualMachineSnapshot, error) {
	objKey := cacheKeyFunc(vmRestore.Namespace, vmRestore.Spec.VirtualMachineSnapshotName)
	obj, exists, err := ctrl.VMSnapshotInformer.GetStore().GetByKey(objKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("VMSnapshot %s does not exist", objKey)
	}

	vmSnapshot := obj.(*snapshotv1.VirtualMachineSnapshot).DeepCopy()
	if vmSnapshotFailed(vmSnapshot) {
		return nil, fmt.Errorf("VMSnapshot %s failed and is invalid to use", objKey)
	} else if !VmSnapshotReady(vmSnapshot) {
		return nil, fmt.Errorf("VMSnapshot %s not ready", objKey)
	}

	if vmSnapshot.Status.VirtualMachineSnapshotContentName == nil {
		return nil, fmt.Errorf("no snapshot content name in %s", objKey)
	}
	return vmSnapshot, nil
}

func (ctrl *VMRestoreController) getSnapshotContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (*snapshotv1.VirtualMachineSnapshotContent, error) {
	objKey := cacheKeyFunc(vmSnapshot.Namespace, *vmSnapshot.Status.VirtualMachineSnapshotContentName)
	obj, exists, err := ctrl.VMSnapshotContentInformer.GetStore().GetByKey(objKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, fmt.Errorf("VMSnapshotContent %s does not exist", objKey)
	}

	vmSnapshotContent := obj.(*snapshotv1.VirtualMachineSnapshotContent).DeepCopy()
	if !vmSnapshotContentReady(vmSnapshotContent) {
		return nil, fmt.Errorf("VMSnapshotContent %s not ready", objKey)
	}

	return vmSnapshotContent, nil
}

func (ctrl *VMRestoreController) getVM(namespace, name string) (vm *kubevirtv1.VirtualMachine, err error) {
	objKey := cacheKeyFunc(namespace, name)
	obj, exists, err := ctrl.VMInformer.GetStore().GetByKey(objKey)
	if err != nil || !exists {
		return nil, err
	}

	return obj.(*kubevirtv1.VirtualMachine).DeepCopy(), nil
}

func patchVM(vm *kubevirtv1.VirtualMachine, patches []string) (*kubevirtv1.VirtualMachine, error) {
	if len(patches) == 0 {
		return vm, nil
	}

	log.Log.V(3).Object(vm).Infof("patching restore target. VM: %s. patches: %+v", vm.Name, patches)

	marshalledVM, err := json.Marshal(vm)
	if err != nil {
		return vm, fmt.Errorf("cannot marshall VM %s: %v", vm.Name, err)
	}

	jsonPatch := "[\n" + strings.Join(patches, ",\n") + "\n]"

	patch, err := jsonpatch.DecodePatch([]byte(jsonPatch))
	if err != nil {
		return vm, fmt.Errorf("cannot decode vm patches %s: %v", jsonPatch, err)
	}

	modifiedMarshalledVM, err := patch.Apply(marshalledVM)
	if err != nil {
		return vm, fmt.Errorf("failed to apply patch for VM %s: %v", jsonPatch, err)
	}

	vm = &kubevirtv1.VirtualMachine{}
	err = json.Unmarshal(modifiedMarshalledVM, vm)
	if err != nil {
		return vm, fmt.Errorf("cannot unmarshal modified marshalled vm %s: %v", string(modifiedMarshalledVM), err)
	}

	log.Log.V(3).Object(vm).Infof("patching restore target completed. Modified VM: %s", string(modifiedMarshalledVM))

	return vm, nil
}

func (ctrl *VMRestoreController) getDV(namespace, name string) (*cdiv1.DataVolume, error) {
	objKey := cacheKeyFunc(namespace, name)
	obj, exists, err := ctrl.DataVolumeInformer.GetStore().GetByKey(objKey)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	return obj.(*cdiv1.DataVolume).DeepCopy(), nil
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
	volumeBackup *snapshotv1.VolumeBackup,
	volumeRestore *snapshotv1.VolumeRestore,
	sourceVmName, sourceVmNamespace string,
) error {
	if volumeBackup == nil || volumeBackup.VolumeSnapshotName == nil {
		log.Log.Errorf("VolumeSnapshot name missing %+v", volumeBackup)
		return fmt.Errorf("missing VolumeSnapshot name")
	}

	if vmRestore == nil {
		return fmt.Errorf("missing vmRestore")
	}
	volumeSnapshot, err := ctrl.VolumeSnapshotProvider.GetVolumeSnapshot(vmRestore.Namespace, *volumeBackup.VolumeSnapshotName)
	if err != nil {
		return err
	}
	if volumeSnapshot == nil {
		return fmt.Errorf("missing volumeSnapshot %s", *volumeBackup.VolumeSnapshotName)
	}

	if volumeRestore == nil {
		return fmt.Errorf("missing volumeRestore")
	}
	pvc, err := CreateRestorePVCDefFromVMRestore(vmRestore.Name, volumeRestore.PersistentVolumeClaimName, volumeSnapshot, volumeBackup, sourceVmName, sourceVmNamespace)
	if err != nil {
		return err
	}
	target.Own(pvc)

	_, err = ctrl.Client.CoreV1().PersistentVolumeClaims(vmRestore.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func CreateRestorePVCDef(restorePVCName string, volumeSnapshot *vsv1.VolumeSnapshot, volumeBackup *snapshotv1.VolumeBackup) (*corev1.PersistentVolumeClaim, error) {
	if volumeBackup == nil || volumeBackup.VolumeSnapshotName == nil {
		return nil, fmt.Errorf("VolumeSnapshot name missing %+v", volumeBackup)
	}
	apiGroup := vsv1.GroupName
	sourcePVC := volumeBackup.PersistentVolumeClaim.DeepCopy()
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        restorePVCName,
			Labels:      sourcePVC.Labels,
			Annotations: sourcePVC.Annotations,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      sourcePVC.Spec.AccessModes,
			Resources:        sourcePVC.Spec.Resources,
			StorageClassName: sourcePVC.Spec.StorageClassName,
			VolumeMode:       sourcePVC.Spec.VolumeMode,
			DataSource: &corev1.TypedLocalObjectReference{
				APIGroup: &apiGroup,
				Kind:     "VolumeSnapshot",
				Name:     *volumeBackup.VolumeSnapshotName,
			},
			DataSourceRef: &corev1.TypedObjectReference{
				APIGroup: &apiGroup,
				Kind:     "VolumeSnapshot",
				Name:     *volumeBackup.VolumeSnapshotName,
			},
		},
	}

	if volumeSnapshot == nil {
		return nil, fmt.Errorf("VolumeSnapshot missing %+v", volumeSnapshot)
	}
	if volumeSnapshot.Status != nil && volumeSnapshot.Status.RestoreSize != nil {
		restorePVCSize, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		// Update restore pvc size to be the maximum between the source PVC and the restore size
		if !ok || restorePVCSize.Cmp(*volumeSnapshot.Status.RestoreSize) < 0 {
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *volumeSnapshot.Status.RestoreSize
		}
	}

	for _, prefix := range restoreAnnotationsToDelete {
		for anno := range pvc.Annotations {
			if strings.HasPrefix(anno, prefix) {
				delete(pvc.Annotations, anno)
			}
		}
	}

	return pvc, nil
}

func getRestoreAnnotationValue(restore *snapshotv1.VirtualMachineRestore) string {
	return fmt.Sprintf("%s-%s", restore.Name, restore.UID)
}

func hasLastRestoreAnnotation(restore *snapshotv1.VirtualMachineRestore, obj metav1.Object) bool {
	return obj.GetAnnotations()[lastRestoreAnnotation] == getRestoreAnnotationValue(restore)
}

func setLastRestoreAnnotation(restore *snapshotv1.VirtualMachineRestore, obj metav1.Object) {
	if obj.GetAnnotations() == nil {
		obj.SetAnnotations(make(map[string]string))
	}
	obj.GetAnnotations()[lastRestoreAnnotation] = getRestoreAnnotationValue(restore)
}

func CreateRestorePVCDefFromVMRestore(vmRestoreName, restorePVCName string, volumeSnapshot *vsv1.VolumeSnapshot, volumeBackup *snapshotv1.VolumeBackup, sourceVmName, sourceVmNamespace string) (*corev1.PersistentVolumeClaim, error) {
	pvc, err := CreateRestorePVCDef(restorePVCName, volumeSnapshot, volumeBackup)
	if err != nil {
		return nil, err
	}
	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}

	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}
	pvc.Labels[restoreSourceNameLabel] = sourceVmName
	pvc.Labels[restoreSourceNamespaceLabel] = sourceVmNamespace
	pvc.Annotations[RestoreNameAnnotation] = vmRestoreName
	return pvc, nil
}

func updateRestoreCondition(r *snapshotv1.VirtualMachineRestore, c snapshotv1.Condition) {
	r.Status.Conditions = updateCondition(r.Status.Conditions, c, true)
}

// Returns a set of volumes not for restore
// Currently only memory dump volumes should not be restored
func (ctrl *VMRestoreController) volumesNotForRestore(content *snapshotv1.VirtualMachineSnapshotContent) (sets.String, error) {
	noRestore := sets.NewString()

	volumes, err := storageutils.GetVolumes(content.Spec.Source.VirtualMachine, ctrl.Client, storageutils.WithBackendVolume)
	if err != nil {
		return noRestore, err
	}

	for _, volume := range volumes {
		if volume.MemoryDump != nil {
			noRestore.Insert(volume.Name)
		}
	}

	return noRestore, nil
}

func getRestoreVolumeBackup(volName string, content *snapshotv1.VirtualMachineSnapshotContent) (*snapshotv1.VolumeBackup, error) {
	for _, vb := range content.Spec.VolumeBackups {
		if vb.VolumeName == volName {
			return &vb, nil
		}
	}
	return &snapshotv1.VolumeBackup{}, fmt.Errorf("volume backup for volume %s not found", volName)
}
