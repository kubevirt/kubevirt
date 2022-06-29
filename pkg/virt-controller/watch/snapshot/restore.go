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
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	pvcRestoreAnnotation = "restore.kubevirt.io/name"

	populatedForPVCAnnotation = "cdi.kubevirt.io/storage.populatedFor"

	lastRestoreAnnotation = "restore.kubevirt.io/lastRestoreUID"

	restoreSourceNameLabel = "restore.kubevirt.io/source-vm-name"

	restoreSourceNamespaceLabel = "restore.kubevirt.io/source-vm-namespace"

	restoreCompleteEvent = "VirtualMachineRestoreComplete"

	restoreErrorEvent = "VirtualMachineRestoreError"
)

type restoreTarget interface {
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
	"volume.kubernetes.io",
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
	if !equality.Semantic.DeepEqual(vmRestoreIn, vmRestoreOut) {
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
	if !equality.Semantic.DeepEqual(original, updated) {
		if _, err := ctrl.Client.VirtualMachineRestore(updated.Namespace).Update(context.Background(), updated, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *VMRestoreController) reconcileVolumeRestores(vmRestore *snapshotv1.VirtualMachineRestore, target restoreTarget) (bool, error) {
	content, err := ctrl.getSnapshotContent(vmRestore)
	if err != nil {
		return false, err
	}

	noRestore := volumesNotForRestore(content)

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
			if err = ctrl.createRestorePVC(vmRestore, target, backup, restore, content.Spec.Source.VirtualMachine.Name, content.Spec.Source.VirtualMachine.Namespace); err != nil {
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

func (t *vmRestoreTarget) UpdateDoneRestore() (bool, error) {
	if !t.doesTargetVMExist() {
		return true, nil
	}

	if t.vm.Status.RestoreInProgress == nil || *t.vm.Status.RestoreInProgress != t.vmRestore.Name {
		return false, nil
	}

	vmCopy := t.vm.DeepCopy()

	vmCopy.Status.RestoreInProgress = nil
	if vmCopy.Status.MemoryDumpRequest != nil {
		vmCopy.Status.MemoryDumpRequest = nil
	}
	return true, t.controller.vmStatusUpdater.UpdateStatus(vmCopy)
}

func (t *vmRestoreTarget) UpdateRestoreInProgress() error {
	if !t.doesTargetVMExist() {
		return nil
	}

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
	if !t.doesTargetVMExist() {
		return true, nil
	}

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

func stripIdentityInfo(vm *kubevirtv1.VirtualMachine) {
	vmTemplate := vm.Spec.Template
	if vmTemplate == nil {
		return
	}

	fw := vmTemplate.Spec.Domain.Firmware
	if fw == nil {
		return
	}

	fw.UUID = ""
}

func (t *vmRestoreTarget) Reconcile() (bool, error) {
	log.Log.Object(t.vmRestore).V(3).Info("Reconciling VM")

	restoreID := fmt.Sprintf("%s-%s", t.vmRestore.Name, t.vmRestore.UID)

	if t.doesTargetVMExist() {
		if lastRestoreID, ok := t.vm.Annotations[lastRestoreAnnotation]; ok && lastRestoreID == restoreID {
			return false, nil
		}
	}

	content, err := t.controller.getSnapshotContent(t.vmRestore)
	if err != nil {
		return false, err
	}

	snapshotVM := content.Spec.Source.VirtualMachine
	if snapshotVM == nil {
		return false, fmt.Errorf("unexpected snapshot source")
	}

	var newTemplates = make([]kubevirtv1.DataVolumeTemplateSpec, len(snapshotVM.Spec.DataVolumeTemplates))
	var newVolumes []kubevirtv1.Volume
	var deletedDataVolumes []string
	updatedStatus := false

	for i, t := range snapshotVM.Spec.DataVolumeTemplates {
		t.DeepCopyInto(&newTemplates[i])
	}

	for _, v := range snapshotVM.Spec.Template.Spec.Volumes {
		nv := v.DeepCopy()
		if nv.DataVolume != nil || nv.PersistentVolumeClaim != nil {
			for k := range t.vmRestore.Status.Restores {
				vr := &t.vmRestore.Status.Restores[k]
				if vr.VolumeName != nv.Name {
					continue
				}

				pvc, err := t.controller.getPVC(t.vmRestore.Namespace, vr.PersistentVolumeClaimName)
				if err != nil {
					return false, err
				}

				if pvc == nil {
					return false, fmt.Errorf("pvc %s/%s does not exist and should", t.vmRestore.Namespace, vr.PersistentVolumeClaimName)
				}

				if nv.DataVolume != nil {
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
				} else {
					nv.PersistentVolumeClaim.ClaimName = vr.PersistentVolumeClaimName
				}
			}
		} else if nv.MemoryDump != nil {
			// don't restore memory dump volume in the new spec
			continue
		}
		newVolumes = append(newVolumes, *nv)
	}

	if t.doesTargetVMExist() && updatedStatus {
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

	var newVM *kubevirtv1.VirtualMachine
	if !t.doesTargetVMExist() {
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

		stripIdentityInfo(newVM)
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
	if newVM.Annotations == nil {
		newVM.Annotations = make(map[string]string)
	}
	newVM.Annotations[lastRestoreAnnotation] = restoreID

	newVM, err = patchVM(newVM, t.vmRestore.Spec.Patches)
	if err != nil {
		return false, fmt.Errorf("error patching VM %s: %v", newVM.Name, err)
	}

	if !t.doesTargetVMExist() {
		newVM, err = t.controller.Client.VirtualMachine(t.vmRestore.Namespace).Create(newVM)
	} else {
		newVM, err = t.controller.Client.VirtualMachine(newVM.Namespace).Update(newVM)
	}
	if err != nil {
		return false, err
	}
	t.vm = newVM

	return true, nil
}

func (t *vmRestoreTarget) Own(obj metav1.Object) {
	if !t.doesTargetVMExist() {
		return
	}

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

func (t *vmRestoreTarget) doesTargetVMExist() bool {
	return t.vm != nil
}

func (ctrl *VMRestoreController) getSnapshotContent(vmRestore *snapshotv1.VirtualMachineRestore) (*snapshotv1.VirtualMachineSnapshotContent, error) {
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
		return nil, fmt.Errorf("cannot marshall VM %s: %v", vm.Name, err)
	}

	jsonPatch := "[\n" + strings.Join(patches, ",\n") + "\n]"

	patch, err := jsonpatch.DecodePatch([]byte(jsonPatch))
	if err != nil {
		return nil, fmt.Errorf("cannot decode vm patches %s: %v", jsonPatch, err)
	}

	modifiedMarshalledVM, err := patch.Apply(marshalledVM)
	if err != nil {
		return nil, fmt.Errorf("failed to apply patch for VM %s: %v", jsonPatch, err)
	}

	err = json.Unmarshal(modifiedMarshalledVM, vm)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal modified marshalled vm %s: %v", string(modifiedMarshalledVM), err)
	}

	log.Log.V(3).Object(vm).Infof("patching restore target completed. Modified VM: %s", string(modifiedMarshalledVM))

	return vm, nil
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
	sourceVmName, sourceVmNamespace string,
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

	volumeSnapshot, err := ctrl.VolumeSnapshotProvider.GetVolumeSnapshot(vmRestore.Namespace, *volumeBackup.VolumeSnapshotName)
	if err != nil {
		return err
	}

	if volumeSnapshot == nil {
		log.Log.Errorf("VolumeSnapshot %s is missing", *volumeBackup.VolumeSnapshotName)
		return fmt.Errorf("missing VolumeSnapshot %s", *volumeBackup.VolumeSnapshotName)
	}

	if volumeSnapshot.Status != nil && volumeSnapshot.Status.RestoreSize != nil {
		restorePVCSize, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
		// Update restore pvc size to be the maximum between the source PVC and the restore size
		if !ok || restorePVCSize.Cmp(*volumeSnapshot.Status.RestoreSize) < 0 {
			pvc.Spec.Resources.Requests[corev1.ResourceStorage] = *volumeSnapshot.Status.RestoreSize
		}
	}

	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}
	pvc.Labels[restoreSourceNameLabel] = sourceVmName
	pvc.Labels[restoreSourceNamespaceLabel] = sourceVmNamespace

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

	apiGroup := vsv1.GroupName
	pvc.Spec.DataSource = &corev1.TypedLocalObjectReference{
		APIGroup: &apiGroup,
		Kind:     "VolumeSnapshot",
		Name:     *volumeBackup.VolumeSnapshotName,
	}
	pvc.Spec.VolumeName = ""

	target.Own(pvc)

	_, err = ctrl.Client.CoreV1().PersistentVolumeClaims(vmRestore.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func updateRestoreCondition(r *snapshotv1.VirtualMachineRestore, c snapshotv1.Condition) {
	r.Status.Conditions = updateCondition(r.Status.Conditions, c, true)
}

// Returns a set of volumes not for restore
// Currently only memory dump volumes should not be restored
func volumesNotForRestore(content *snapshotv1.VirtualMachineSnapshotContent) sets.String {
	volumes := content.Spec.Source.VirtualMachine.Spec.Template.Spec.Volumes
	noRestore := sets.NewString()

	for _, volume := range volumes {
		if volume.MemoryDump != nil {
			noRestore.Insert(volume.Name)
		}
	}

	return noRestore
}

func getRestoreVolumeBackup(volName string, content *snapshotv1.VirtualMachineSnapshotContent) (snapshotv1.VolumeBackup, error) {
	for _, vb := range content.Spec.VolumeBackups {
		if vb.VolumeName == volName {
			return vb, nil
		}
	}
	return snapshotv1.VolumeBackup{}, fmt.Errorf("volume backup for volume %s not found", volName)
}
