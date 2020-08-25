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
	"strings"
	"time"

	vsv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	snapshotv1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	sourceFinalizer = "snapshot.kubevirt.io/snapshot-source-protection"

	vmSnapshotFinalizer = "snapshot.kubevirt.io/vmsnapshot-protection"

	vmSnapshotContentFinalizer = "snapshot.kubevirt.io/vmsnapshotcontent-protection"

	defaultVolumeSnapshotClassAnnotation = "snapshot.storage.kubernetes.io/is-default-class"

	vmSnapshotContentCreateEvent = "SuccessfulVirtualMachineSnapshotContentCreate"

	volumeSnapshotCreateEvent = "SuccessfulVolumeSnapshotCreate"

	volumeSnapshotMissingEvent = "VolumeSnapshotMissing"
)

type snapshotSource interface {
	Locked() bool
	Lock() (bool, error)
	Unlock() error
	Spec() snapshotv1.SourceSpec
	PersistentVolumeClaims() map[string]string
}

type vmSnapshotSource struct {
	client   kubecli.KubevirtClient
	vm       *kubevirtv1.VirtualMachine
	snapshot *snapshotv1.VirtualMachineSnapshot
}

func cacheKeyFunc(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func vmSnapshotReady(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshot.Status != nil && vmSnapshot.Status.ReadyToUse != nil && *vmSnapshot.Status.ReadyToUse
}

func vmSnapshotError(vmSnapshot *snapshotv1.VirtualMachineSnapshot) *snapshotv1.VirtualMachineSnapshotError {
	if vmSnapshot.Status != nil && vmSnapshot.Status.Error != nil {
		return vmSnapshot.Status.Error
	}
	return nil
}

func vmSnapshotProgressing(vmSnapshot *snapshotv1.VirtualMachineSnapshot) bool {
	return vmSnapshotError(vmSnapshot) == nil &&
		(vmSnapshot.Status == nil || vmSnapshot.Status.ReadyToUse == nil || *vmSnapshot.Status.ReadyToUse == false)
}

func getVMSnapshotContentName(vmSnapshot *snapshotv1.VirtualMachineSnapshot) string {
	if vmSnapshot.Status != nil && vmSnapshot.Status.VirtualMachineSnapshotContentName != nil {
		return *vmSnapshot.Status.VirtualMachineSnapshotContentName
	}

	return fmt.Sprintf("%s-%s", "vmsnapshot-content", vmSnapshot.UID)
}

func translateError(e *vsv1beta1.VolumeSnapshotError) *snapshotv1.VirtualMachineSnapshotError {
	if e == nil {
		return nil
	}

	return &snapshotv1.VirtualMachineSnapshotError{
		Message: e.Message,
		Time:    e.Time,
	}
}

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
}

func (ctrl *SnapshotController) updateVMSnapshot(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	log.Log.V(3).Infof("Updating VirtualMachineSnapshot %s/%s", vmSnapshot.Namespace, vmSnapshot.Name)

	// Make sure status is initialized
	if vmSnapshot.Status == nil {
		return ctrl.updateSnapshotStatus(vmSnapshot)
	}

	source, err := ctrl.getSnapshotSource(vmSnapshot)
	if err != nil {
		return err
	}

	// unlock the source if done/error
	if !vmSnapshotProgressing(vmSnapshot) && source != nil && source.Locked() {
		if err = source.Unlock(); err != nil {
			return err
		}

		return nil
	}

	// check deleted
	if vmSnapshot.DeletionTimestamp != nil {
		return ctrl.cleanupVMSnapshot(vmSnapshot)
	}

	if source != nil && vmSnapshotProgressing(vmSnapshot) {
		// attempt to lock source
		// if fails will attempt again when source is updated
		if !source.Locked() {
			locked, err := source.Lock()
			if err != nil {
				return err
			}

			log.Log.V(3).Infof("Attempt to lock source returned: %t", locked)

			return nil
		}

		// add source finalizer and maybe other stuff
		updated, err := ctrl.initVMSnapshot(vmSnapshot)
		if updated || err != nil {
			return err
		}

		content, err := ctrl.getContent(vmSnapshot)
		if err != nil {
			return err
		}

		// create content if does not exist
		if content == nil {
			return ctrl.createContent(vmSnapshot)
		}
	}

	if err = ctrl.updateSnapshotStatus(vmSnapshot); err != nil {
		return err
	}

	return nil
}

func (ctrl *SnapshotController) updateVMSnapshotContent(content *snapshotv1.VirtualMachineSnapshotContent) error {
	log.Log.V(3).Infof("Updating VirtualMachineSnapshotContent %s/%s", content.Namespace, content.Name)

	var volueSnapshotStatus []snapshotv1.VolumeSnapshotStatus
	var deletedSnapshots, skippedSnapshots []string

	currentlyReady := content.Status != nil && content.Status.ReadyToUse != nil && *content.Status.ReadyToUse
	currentlyError := content.Status != nil && content.Status.Error != nil

	for _, volumeBackup := range content.Spec.VolumeBackups {
		if volumeBackup.VolumeSnapshotName == nil {
			continue
		}

		vsName := *volumeBackup.VolumeSnapshotName

		volumeSnapshot, err := ctrl.getVolumeSnapshot(content.Namespace, vsName)
		if err != nil {
			return err
		}

		if volumeSnapshot == nil {
			// check if snapshot was deleted
			if currentlyReady {
				log.Log.Warningf("VolumeSnapshot %s no longer exists", vsName)
				ctrl.recorder.Eventf(
					content,
					corev1.EventTypeWarning,
					volumeSnapshotMissingEvent,
					"VolumeSnapshot %s no longer exists",
					vsName,
				)
				deletedSnapshots = append(deletedSnapshots, vsName)
				continue
			}

			if currentlyError {
				log.Log.V(3).Infof("Not creating snapshot %s because in error state", vsName)
				skippedSnapshots = append(skippedSnapshots, vsName)
				continue
			}

			volumeSnapshot, err = ctrl.createVolumeSnapshot(content, volumeBackup)
			if err != nil {
				return err
			}
		}

		vss := snapshotv1.VolumeSnapshotStatus{
			VolumeSnapshotName: volumeSnapshot.Name,
		}

		if volumeSnapshot.Status != nil {
			vss.ReadyToUse = volumeSnapshot.Status.ReadyToUse
			vss.CreationTime = volumeSnapshot.Status.CreationTime
			vss.Error = translateError(volumeSnapshot.Status.Error)
		}

		volueSnapshotStatus = append(volueSnapshotStatus, vss)
	}

	ready := true
	errorMessage := ""
	contentCpy := content.DeepCopy()
	if contentCpy.Status == nil {
		contentCpy.Status = &snapshotv1.VirtualMachineSnapshotContentStatus{}
	}

	if len(deletedSnapshots) > 0 {
		ready = false
		errorMessage = fmt.Sprintf("VolumeSnapshots (%s) missing", strings.Join(deletedSnapshots, ","))
	} else if len(skippedSnapshots) > 0 {
		ready = false
		errorMessage = fmt.Sprintf("VolumeSnapshots (%s) skipped because in error state", strings.Join(skippedSnapshots, ","))
	} else {
		for _, vss := range volueSnapshotStatus {
			if vss.ReadyToUse == nil || *vss.ReadyToUse == false {
				ready = false
			}

			if vss.Error != nil {
				errorMessage = "VolumeSnapshot in error state"
				break
			}
		}
	}

	if ready && (contentCpy.Status.ReadyToUse == nil || *contentCpy.Status.ReadyToUse == false) {
		contentCpy.Status.CreationTime = currentTime()
	}

	if errorMessage != "" &&
		(contentCpy.Status.Error == nil ||
			contentCpy.Status.Error.Message == nil ||
			*contentCpy.Status.Error.Message != errorMessage) {
		contentCpy.Status.Error = &snapshotv1.VirtualMachineSnapshotError{
			Time:    currentTime(),
			Message: &errorMessage,
		}
	}

	contentCpy.Status.ReadyToUse = &ready
	contentCpy.Status.VolumeSnapshotStatus = volueSnapshotStatus

	if !reflect.DeepEqual(content, contentCpy) {
		if _, err := ctrl.client.VirtualMachineSnapshotContent(contentCpy.Namespace).Update(contentCpy); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *SnapshotController) createVolumeSnapshot(
	content *snapshotv1.VirtualMachineSnapshotContent,
	volumeBackup snapshotv1.VolumeBackup,
) (*vsv1beta1.VolumeSnapshot, error) {
	log.Log.Infof("Attempting to create VolumeSnapshot %s", *volumeBackup.VolumeSnapshotName)

	sc := volumeBackup.PersistentVolumeClaim.Spec.StorageClassName
	if sc == nil {
		return nil, fmt.Errorf("%s/%s VolumeSnapshot requested but no storage class",
			content.Namespace, volumeBackup.PersistentVolumeClaim.Name)
	}

	volumeSnapshotClass, err := ctrl.getVolumeSnapshotClass(*sc)
	if err != nil {
		log.Log.Warningf("Couldn't find VolumeSnapshotClass for %s", *sc)
		return nil, err
	}

	t := true
	snapshot := &vsv1beta1.VolumeSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name: *volumeBackup.VolumeSnapshotName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         snapshotv1.SchemeGroupVersion.String(),
					Kind:               "VirtualMachineSnapshotContent",
					Name:               content.Name,
					UID:                content.UID,
					Controller:         &t,
					BlockOwnerDeletion: &t,
				},
			},
		},
		Spec: vsv1beta1.VolumeSnapshotSpec{
			Source: vsv1beta1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &volumeBackup.PersistentVolumeClaim.Name,
			},
			VolumeSnapshotClassName: &volumeSnapshotClass,
		},
	}

	volumeSnapshot, err := ctrl.client.KubernetesSnapshotClient().SnapshotV1beta1().
		VolumeSnapshots(content.Namespace).
		Create(snapshot)
	if err != nil {
		return nil, err
	}

	ctrl.recorder.Eventf(
		content,
		corev1.EventTypeNormal,
		volumeSnapshotCreateEvent,
		"Successfully created VolumeSnapshot %s",
		snapshot.Name,
	)

	return volumeSnapshot, nil
}

func (ctrl *SnapshotController) getSnapshotSource(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (snapshotSource, error) {
	switch vmSnapshot.Spec.Source.Kind {
	case "VirtualMachine":
		vm, err := ctrl.getVM(vmSnapshot)
		if err != nil {
			return nil, err
		}

		if vm == nil {
			return nil, nil
		}

		return &vmSnapshotSource{
			client:   ctrl.client,
			vm:       vm,
			snapshot: vmSnapshot,
		}, nil
	}

	return nil, fmt.Errorf("unknown source %+v", vmSnapshot.Spec.Source)
}

func (ctrl *SnapshotController) initVMSnapshot(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (bool, error) {
	if controller.HasFinalizer(vmSnapshot, vmSnapshotFinalizer) {
		return false, nil
	}

	vmSnapshotCpy := vmSnapshot.DeepCopy()
	controller.AddFinalizer(vmSnapshotCpy, vmSnapshotFinalizer)

	if _, err := ctrl.client.VirtualMachineSnapshot(vmSnapshot.Namespace).Update(vmSnapshotCpy); err != nil {
		return false, err
	}

	return true, nil
}

func (ctrl *SnapshotController) cleanupVMSnapshot(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	// TODO check restore in progress

	if vmSnapshotProgressing(vmSnapshot) {
		// will put the snapshot in error state
		return ctrl.updateSnapshotStatus(vmSnapshot)
	}

	content, err := ctrl.getContent(vmSnapshot)
	if err != nil {
		return err
	}

	if content != nil {
		if controller.HasFinalizer(content, vmSnapshotContentFinalizer) {
			cpy := content.DeepCopy()
			controller.RemoveFinalizer(cpy, vmSnapshotContentFinalizer)

			_, err := ctrl.client.VirtualMachineSnapshotContent(cpy.Namespace).Update(cpy)
			if err != nil {
				return err
			}
		}

		if vmSnapshot.Spec.DeletionPolicy == nil ||
			*vmSnapshot.Spec.DeletionPolicy == snapshotv1.VirtualMachineSnapshotContentDelete {
			log.Log.V(2).Infof("Deleting vmsnapshotcontent %s/%s", content.Namespace, content.Name)

			err = ctrl.client.VirtualMachineSnapshotContent(vmSnapshot.Namespace).Delete(content.Name, &metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return err
			}
		} else {
			log.Log.V(2).Infof("NOT deleting vmsnapshotcontent %s/%s", content.Namespace, content.Name)
		}
	}

	if controller.HasFinalizer(vmSnapshot, vmSnapshotFinalizer) {
		vmSnapshotCpy := vmSnapshot.DeepCopy()
		controller.RemoveFinalizer(vmSnapshotCpy, vmSnapshotFinalizer)

		_, err := ctrl.client.VirtualMachineSnapshot(vmSnapshotCpy.Namespace).Update(vmSnapshotCpy)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *SnapshotController) createContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	source, err := ctrl.getSnapshotSource(vmSnapshot)
	if err != nil {
		return err
	}

	var volumeBackups []snapshotv1.VolumeBackup
	for diskName, pvcName := range source.PersistentVolumeClaims() {
		pvc, err := ctrl.getSnapshotPVC(vmSnapshot.Namespace, pvcName)
		if err != nil {
			return err
		}

		if pvc == nil {
			log.Log.Warningf("No VolumeSnapshotClass for %s/%s", vmSnapshot.Namespace, pvcName)
			continue
		}

		pvcCpy := pvc.DeepCopy()
		pvcCpy.Status = corev1.PersistentVolumeClaimStatus{}
		volumeSnapshotName := fmt.Sprintf("vmsnapshot-%s-disk-%s", vmSnapshot.UID, diskName)

		vb := snapshotv1.VolumeBackup{
			DiskName:              diskName,
			PersistentVolumeClaim: *pvcCpy,
			VolumeSnapshotName:    &volumeSnapshotName,
		}

		volumeBackups = append(volumeBackups, vb)
	}

	content := &snapshotv1.VirtualMachineSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       getVMSnapshotContentName(vmSnapshot),
			Namespace:  vmSnapshot.Namespace,
			Finalizers: []string{vmSnapshotContentFinalizer},
		},
		Spec: snapshotv1.VirtualMachineSnapshotContentSpec{
			VirtualMachineSnapshotName: &vmSnapshot.Name,
			Source:                     source.Spec(),
			VolumeBackups:              volumeBackups,
		},
	}

	_, err = ctrl.client.VirtualMachineSnapshotContent(content.Namespace).Create(content)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	ctrl.recorder.Eventf(
		vmSnapshot,
		corev1.EventTypeNormal,
		vmSnapshotContentCreateEvent,
		"Successfully created VirtualMachineSnapshotContent %s",
		content.Name,
	)

	return nil
}

func (ctrl *SnapshotController) getSnapshotPVC(namespace string, volumeName string) (*corev1.PersistentVolumeClaim, error) {
	obj, exists, err := ctrl.pvcInformer.GetStore().GetByKey(cacheKeyFunc(namespace, volumeName))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	pvc := obj.(*corev1.PersistentVolumeClaim)

	if pvc.Spec.VolumeName == "" {
		log.Log.Warningf("Unbound PVC %s/%s", pvc.Namespace, pvc.Name)
		return nil, nil
	}

	if pvc.Spec.StorageClassName == nil {
		log.Log.Warningf("No storage class for PVC %s/%s", pvc.Namespace, pvc.Name)
		return nil, nil
	}

	volumeSnapshotClass, err := ctrl.getVolumeSnapshotClass(*pvc.Spec.StorageClassName)
	if err != nil {
		return nil, err
	}

	if volumeSnapshotClass != "" {
		return pvc, nil
	}

	return nil, nil
}

func (ctrl *SnapshotController) getVolumeSnapshotClass(storageClassName string) (string, error) {
	obj, exists, err := ctrl.storageClassInformer.GetStore().GetByKey(storageClassName)
	if !exists || err != nil {
		return "", err
	}

	storageClass := obj.(*storagev1.StorageClass)

	var matches []vsv1beta1.VolumeSnapshotClass
	volumeSnapshotClasses := ctrl.getVolumeSnapshotClasses()
	for _, volumeSnapshotClass := range volumeSnapshotClasses {
		if volumeSnapshotClass.Driver == storageClass.Provisioner {
			matches = append(matches, volumeSnapshotClass)
		}
	}

	if len(matches) == 0 {
		log.Log.Warningf("No VolumeSnapshotClass for %s", storageClassName)
		return "", nil
	}

	if len(matches) == 1 {
		return matches[0].Name, nil
	}

	for _, volumeSnapshotClass := range matches {
		for annotation := range volumeSnapshotClass.Annotations {
			if annotation == defaultVolumeSnapshotClassAnnotation {
				return volumeSnapshotClass.Name, nil
			}
		}
	}

	return "", fmt.Errorf("%d matching VolumeSnapshotClasses for %s", len(matches), storageClassName)
}

func (ctrl *SnapshotController) updateSnapshotStatus(vmSnapshot *snapshotv1.VirtualMachineSnapshot) error {
	f := false
	vmSnapshotCpy := vmSnapshot.DeepCopy()
	if vmSnapshotCpy.Status == nil {
		vmSnapshotCpy.Status = &snapshotv1.VirtualMachineSnapshotStatus{
			ReadyToUse: &f,
		}
	}

	if vmSnapshotCpy.DeletionTimestamp != nil {
		// go into error state
		if vmSnapshotProgressing(vmSnapshot) {
			reason := "Snapshot cancelled"
			vmSnapshotCpy.Status.Error = newVirtualMachineSnapshotError(reason)
			updateSnapshotCondition(vmSnapshotCpy, newSnapshotProgressingCondition(corev1.ConditionFalse, reason))
			updateSnapshotCondition(vmSnapshotCpy, newSnapshotReadyCondition(corev1.ConditionFalse, reason))
		}
	} else {
		content, err := ctrl.getContent(vmSnapshot)
		if err != nil {
			return err
		}

		if content != nil && content.Status != nil {
			// content exists and is initialized
			vmSnapshotCpy.Status.VirtualMachineSnapshotContentName = &content.Name
			vmSnapshotCpy.Status.CreationTime = content.Status.CreationTime
			vmSnapshotCpy.Status.ReadyToUse = content.Status.ReadyToUse
			vmSnapshotCpy.Status.Error = content.Status.Error
		}
	}

	if vmSnapshotProgressing(vmSnapshotCpy) {
		source, err := ctrl.getSnapshotSource(vmSnapshot)
		if err != nil {
			return err
		}

		if source != nil {
			if source.Locked() {
				updateSnapshotCondition(vmSnapshotCpy, newSnapshotProgressingCondition(corev1.ConditionTrue, "Source locked and operation in progress"))
			} else {
				updateSnapshotCondition(vmSnapshotCpy, newSnapshotProgressingCondition(corev1.ConditionFalse, "Source not locked"))
			}
		} else {
			updateSnapshotCondition(vmSnapshotCpy, newSnapshotProgressingCondition(corev1.ConditionFalse, "Source does not exist"))
		}
		updateSnapshotCondition(vmSnapshotCpy, newSnapshotReadyCondition(corev1.ConditionFalse, "Not ready"))
	} else if vmSnapshotError(vmSnapshotCpy) != nil {
		updateSnapshotCondition(vmSnapshotCpy, newSnapshotProgressingCondition(corev1.ConditionFalse, "In error state"))
		updateSnapshotCondition(vmSnapshotCpy, newSnapshotReadyCondition(corev1.ConditionFalse, "Error"))
	} else if vmSnapshotReady(vmSnapshotCpy) {
		updateSnapshotCondition(vmSnapshotCpy, newSnapshotProgressingCondition(corev1.ConditionFalse, "Operation complete"))
		updateSnapshotCondition(vmSnapshotCpy, newSnapshotReadyCondition(corev1.ConditionTrue, "Operation complete"))
	} else {
		updateSnapshotCondition(vmSnapshotCpy, newSnapshotProgressingCondition(corev1.ConditionUnknown, "Unknown state"))
		updateSnapshotCondition(vmSnapshotCpy, newSnapshotReadyCondition(corev1.ConditionUnknown, "Unknown state"))
	}

	// try to observe snapshot duration after update
	ctrl.tryToObserveSnapshotDuration(vmSnapshotCpy)

	if !reflect.DeepEqual(vmSnapshot, vmSnapshotCpy) {
		if _, err := ctrl.client.VirtualMachineSnapshot(vmSnapshotCpy.Namespace).Update(vmSnapshotCpy); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *SnapshotController) getVM(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (*kubevirtv1.VirtualMachine, error) {
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

func (ctrl *SnapshotController) getContent(vmSnapshot *snapshotv1.VirtualMachineSnapshot) (*snapshotv1.VirtualMachineSnapshotContent, error) {
	contentName := getVMSnapshotContentName(vmSnapshot)
	obj, exists, err := ctrl.vmSnapshotContentInformer.GetStore().GetByKey(cacheKeyFunc(vmSnapshot.Namespace, contentName))
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, nil
	}

	return obj.(*snapshotv1.VirtualMachineSnapshotContent), nil
}

func (s *vmSnapshotSource) Locked() bool {
	return s.vm.Status.SnapshotInProgress != nil && *s.vm.Status.SnapshotInProgress == s.snapshot.Name
}

func (s *vmSnapshotSource) Lock() (bool, error) {
	if s.Locked() {
		return true, nil
	}

	if s.vm.Spec.Running == nil || *s.vm.Spec.Running {
		log.Log.V(3).Infof("Snapshottting a running VM is not supported yet")
		return false, nil
	}

	if s.vm.Status.SnapshotInProgress != nil && *s.vm.Status.SnapshotInProgress != s.snapshot.Name {
		log.Log.V(3).Infof("Snapshot %s in progress", *s.vm.Status.SnapshotInProgress)
		return false, nil
	}

	log.Log.Infof("Adding VM snapshot finalizer to %s", s.vm.Name)

	vmCopy := s.vm.DeepCopy()
	vmCopy.Status.SnapshotInProgress = &s.snapshot.Name
	controller.AddFinalizer(vmCopy, sourceFinalizer)

	_, err := s.client.VirtualMachine(vmCopy.Namespace).UpdateStatus(vmCopy)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *vmSnapshotSource) Unlock() error {
	if !s.Locked() {
		return nil
	}

	vmCopy := s.vm.DeepCopy()
	vmCopy.Status.SnapshotInProgress = nil
	controller.RemoveFinalizer(vmCopy, sourceFinalizer)

	_, err := s.client.VirtualMachine(vmCopy.Namespace).UpdateStatus(vmCopy)
	if err != nil {
		return err
	}

	return nil
}

func (s *vmSnapshotSource) Spec() snapshotv1.SourceSpec {
	vmCpy := s.vm.DeepCopy()
	vmCpy.Status = kubevirtv1.VirtualMachineStatus{}
	return snapshotv1.SourceSpec{
		VirtualMachine: vmCpy,
	}
}

func (s *vmSnapshotSource) PersistentVolumeClaims() map[string]string {
	return getPVCsFromVolumes(s.vm.Spec.Template.Spec.Volumes)
}

func getPVCsFromVolumes(volumes []kubevirtv1.Volume) map[string]string {
	pvcs := map[string]string{}

	for _, volume := range volumes {
		var pvcName string

		if volume.PersistentVolumeClaim != nil {
			pvcName = volume.PersistentVolumeClaim.ClaimName
		} else if volume.DataVolume != nil {
			pvcName = volume.DataVolume.Name
		} else {
			continue
		}

		pvcs[volume.Name] = pvcName
	}

	return pvcs
}

func newVirtualMachineSnapshotError(message string) *snapshotv1.VirtualMachineSnapshotError {
	return &snapshotv1.VirtualMachineSnapshotError{
		Message: &message,
		Time:    currentTime(),
	}
}

func newSnapshotReadyCondition(status corev1.ConditionStatus, reason string) snapshotv1.VirtualMachineSnapshotCondition {
	return snapshotv1.VirtualMachineSnapshotCondition{
		Type:               snapshotv1.VirtualMachineSnapshotConditionReady,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func newSnapshotProgressingCondition(status corev1.ConditionStatus, reason string) snapshotv1.VirtualMachineSnapshotCondition {
	return snapshotv1.VirtualMachineSnapshotCondition{
		Type:               snapshotv1.VirtualMachineSnapshotConditionProgressing,
		Status:             status,
		Reason:             reason,
		LastTransitionTime: *currentTime(),
	}
}

func updateSnapshotCondition(ss *snapshotv1.VirtualMachineSnapshot, c snapshotv1.VirtualMachineSnapshotCondition) {
	found := false
	for i := range ss.Status.Conditions {
		if ss.Status.Conditions[i].Type == c.Type {
			if ss.Status.Conditions[i].Status != c.Status {
				ss.Status.Conditions[i] = c
			}
			found = true
			break
		}
	}

	if !found {
		ss.Status.Conditions = append(ss.Status.Conditions, c)
	}
}

// tryToObserveSnapshotDuration will validate if the snapshot was completed and has the necessary fields to add a new sample to the histogram.
// Will also add the new sample to the histogram if the validation was successful
func (ctrl *SnapshotController) tryToObserveSnapshotDuration(ss *snapshotv1.VirtualMachineSnapshot) {
	if !vmSnapshotProgressing(ss) {
		if vmSnapshotError(ss) != nil {
			timeSpent := ss.Status.Error.Time.Sub(ss.Status.CreationTime.Time)
			ctrl.snapshotMetrics.ObserveSnapshotDuration(float64(timeSpent/time.Second), "failed")
		} else if vmSnapshotReady(ss) {
			for _, condition := range ss.Status.Conditions {
				if condition.Type == snapshotv1.VirtualMachineSnapshotConditionReady {
					timeSpent := condition.LastTransitionTime.Time.Sub(ss.Status.CreationTime.Time)
					ctrl.snapshotMetrics.ObserveSnapshotDuration(float64(timeSpent/time.Second), "succeeded")
					continue
				}
			}
		}
	}
}
