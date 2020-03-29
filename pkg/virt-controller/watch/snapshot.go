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

	k8ssnapshotv1beta1 "github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubevirtv1 "kubevirt.io/client-go/api/v1"
	vmsnapshotv1alpha1 "kubevirt.io/client-go/apis/snapshot/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
)

const (
	sourceFinalizer = "snapshot.kubevirt.io/snapshot-source-protection"

	vmSnapshotFinalizer = "snapshot.kubevirt.io/vmsnapshot-protection"

	vmSnapshotContentFinalizer = "snapshot.kubevirt.io/vmsnapshotcontent-protection"

	defaultVolumeSnapshotClassAnnotation = "snapshot.storage.kubernetes.io/is-default-class"
)

type snapshotSource interface {
	Locked() bool
	Lock() (bool, error)
	Unlock() error
	Spec() vmsnapshotv1alpha1.SourceSpec
	PersistentVolumeClaims() map[string]string
}

type vmSnapshotSource struct {
	client   kubecli.KubevirtClient
	vm       *kubevirtv1.VirtualMachine
	snapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot
}

func cacheKeyFunc(namespace, name string) string {
	return fmt.Sprintf("%s/%s", namespace, name)
}

func vmSnapshotError(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) *vmsnapshotv1alpha1.VirtualMachineSnapshotError {
	if vmSnapshot.Status != nil && vmSnapshot.Status.Error != nil {
		return vmSnapshot.Status.Error
	}
	return nil
}

func vmSnapshotProgressing(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) bool {
	return vmSnapshotError(vmSnapshot) == nil &&
		(vmSnapshot.Status == nil || vmSnapshot.Status.ReadyToUse == nil || *vmSnapshot.Status.ReadyToUse == false)
}

func getVMSnapshotContentName(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) string {
	if vmSnapshot.Status != nil && vmSnapshot.Status.VirtualMachineSnapshotContentName != nil {
		return *vmSnapshot.Status.VirtualMachineSnapshotContentName
	}

	return fmt.Sprintf("%s-%s", "vmsnapshot-content", vmSnapshot.UID)
}

// variable so can be overridden in tests
var currentTime = func() *metav1.Time {
	t := metav1.Now()
	return &t
}

func (ctrl *SnapshotController) updateVMSnapshot(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) error {
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

func (ctrl *SnapshotController) updateVMSnapshotContent(content *vmsnapshotv1alpha1.VirtualMachineSnapshotContent) error {
	log.Log.V(3).Infof("Updating VirtualMachineSnapshotContent %s/%s", content.Namespace, content.Name)

	if content.Status != nil && content.Status.Error != nil {
		log.Log.V(3).Infof("VolumeSnapshotContent %s/%s in error state, ignoring", content.Namespace, content.Name)
		return nil
	}

	ready := true
	creationTime := currentTime()
	var vmSnapshotError *vmsnapshotv1alpha1.VirtualMachineSnapshotError
	var deletedSnapshots []string

	for _, volmeBackup := range content.Spec.VolumeBackups {
		if volmeBackup.VolumeSnapshotName != nil {
			key := fmt.Sprintf("%s/%s", content.Namespace, *volmeBackup.VolumeSnapshotName)
			obj, exists, err := ctrl.volumeSnapshotInformer.GetStore().GetByKey(key)
			if err != nil {
				return err
			}

			if !exists {
				// check if snapshot was deleted
				if content.Status != nil && content.Status.ReadyToUse != nil && *content.Status.ReadyToUse {
					log.Log.Warningf("VolumeSnapshot %s no longer exists", *volmeBackup.VolumeSnapshotName)
					ready = false
					deletedSnapshots = append(deletedSnapshots, *volmeBackup.VolumeSnapshotName)
					continue
				}

				log.Log.Infof("Attempting to create VolumeSnapshot %s", *volmeBackup.VolumeSnapshotName)

				sc := volmeBackup.PersistentVolumeClaim.Spec.StorageClassName
				if sc == nil {
					return fmt.Errorf("%s/%s VolumeSnapshot requested but no storage class",
						content.Namespace, volmeBackup.PersistentVolumeClaim.Name)
				}

				volumeSnapshotClass, err := ctrl.getVolumeSnapshotClass(*sc)
				if err != nil {
					log.Log.Warningf("Couldn't find VolumeSnapshotClass for %s", *sc)
					return err
				}

				t := true
				snapshot := &k8ssnapshotv1beta1.VolumeSnapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name: *volmeBackup.VolumeSnapshotName,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         vmsnapshotv1alpha1.SchemeGroupVersion.String(),
								Kind:               "VirtualMachineSnapshotContent",
								Name:               content.Name,
								UID:                content.UID,
								Controller:         &t,
								BlockOwnerDeletion: &t,
							},
						},
					},
					Spec: k8ssnapshotv1beta1.VolumeSnapshotSpec{
						Source: k8ssnapshotv1beta1.VolumeSnapshotSource{
							PersistentVolumeClaimName: &volmeBackup.PersistentVolumeClaim.Name,
						},
						VolumeSnapshotClassName: &volumeSnapshotClass,
					},
				}

				_, err = ctrl.client.KubernetesSnapshotClient().SnapshotV1beta1().
					VolumeSnapshots(content.Namespace).
					Create(snapshot)
				if err != nil && !errors.IsAlreadyExists(err) {
					return err
				}

				ready = false
				continue
			} else if err != nil {
				return err
			}

			vs := obj.(*k8ssnapshotv1beta1.VolumeSnapshot)

			if vs.Status != nil {
				if vs.Status.ReadyToUse == nil || *vs.Status.ReadyToUse == false {
					ready = false
				}

				if vs.Status.Error != nil {
					ready = false
					vmSnapshotError = &vmsnapshotv1alpha1.VirtualMachineSnapshotError{
						Time:    vs.Status.Error.Time,
						Message: vs.Status.Error.Message,
					}

					break
				}
			}
		}
	}

	if vmSnapshotError == nil && len(deletedSnapshots) > 0 {
		message := fmt.Sprintf("VolumeSnapshots (%s) missing", strings.Join(deletedSnapshots, ","))
		vmSnapshotError = &vmsnapshotv1alpha1.VirtualMachineSnapshotError{
			Time:    currentTime(),
			Message: &message,
		}
	}

	if !ready {
		creationTime = nil
	}

	contentCpy := content.DeepCopy()
	contentCpy.Status.ReadyToUse = &ready
	contentCpy.Status.CreationTime = creationTime
	contentCpy.Status.Error = vmSnapshotError

	if !reflect.DeepEqual(content, contentCpy) {
		if _, err := ctrl.client.VirtualMachineSnapshotContent(contentCpy.Namespace).Update(contentCpy); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *SnapshotController) getSnapshotSource(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (snapshotSource, error) {
	switch {
	case vmSnapshot.Spec.Source.VirtualMachineName != nil:
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

func (ctrl *SnapshotController) initVMSnapshot(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (bool, error) {
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

func (ctrl *SnapshotController) cleanupVMSnapshot(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) error {
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
			*vmSnapshot.Spec.DeletionPolicy == vmsnapshotv1alpha1.VirtualMachineSnapshotContentDelete {
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

func (ctrl *SnapshotController) createContent(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) error {
	source, err := ctrl.getSnapshotSource(vmSnapshot)
	if err != nil {
		return err
	}

	var volumeBackups []vmsnapshotv1alpha1.VolumeBackup
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

		vb := vmsnapshotv1alpha1.VolumeBackup{
			DiskName:              diskName,
			PersistentVolumeClaim: *pvcCpy,
			VolumeSnapshotName:    &volumeSnapshotName,
		}

		volumeBackups = append(volumeBackups, vb)
	}

	ready := false
	content := &vmsnapshotv1alpha1.VirtualMachineSnapshotContent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       getVMSnapshotContentName(vmSnapshot),
			Namespace:  vmSnapshot.Namespace,
			Finalizers: []string{vmSnapshotContentFinalizer},
		},
		Spec: vmsnapshotv1alpha1.VirtualMachineSnapshotContentSpec{
			VirtualMachineSnapshotName: &vmSnapshot.Name,
			Source:                     source.Spec(),
			VolumeBackups:              volumeBackups,
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
	var provisioner string
	storageClasses := ctrl.storageClassInformer.GetStore().List()
	for _, obj := range storageClasses {
		storageClass := obj.(*storagev1.StorageClass)
		if storageClass.Name == storageClassName {
			provisioner = storageClass.Provisioner
			break
		}
	}

	if provisioner == "" {
		log.Log.Warningf("No StorageClass named %s", storageClassName)
		return "", nil
	}

	var matches []*k8ssnapshotv1beta1.VolumeSnapshotClass
	volumeSnapshotClasses := ctrl.volumeSnapshotClassInformer.GetStore().List()
	for _, obj := range volumeSnapshotClasses {
		volumeSnapshotClass := obj.(*k8ssnapshotv1beta1.VolumeSnapshotClass)
		if volumeSnapshotClass.Driver == provisioner {
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

func (ctrl *SnapshotController) updateSnapshotStatus(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) error {
	vmSnapshotCpy := vmSnapshot.DeepCopy()
	if vmSnapshotCpy.Status == nil {
		vmSnapshotCpy.Status = &vmsnapshotv1alpha1.VirtualMachineSnapshotStatus{}
	}

	if vmSnapshot.DeletionTimestamp != nil {
		// go into error state
		if vmSnapshotProgressing(vmSnapshot) {
			message := "Snapshot cancelled"
			vmSnapshotCpy.Status.Error = &vmsnapshotv1alpha1.VirtualMachineSnapshotError{
				Message: &message,
				Time:    currentTime(),
			}
		}
	} else {
		content, err := ctrl.getContent(vmSnapshot)
		if err != nil {
			return err
		}

		if content != nil {
			vmSnapshotCpy.Status.VirtualMachineSnapshotContentName = &content.Name
			vmSnapshotCpy.Status.CreationTime = content.Status.CreationTime
			vmSnapshotCpy.Status.ReadyToUse = content.Status.ReadyToUse
			vmSnapshotCpy.Status.Error = content.Status.Error
		} else {
			f := false
			vmSnapshotCpy.Status.ReadyToUse = &f
		}
	}

	if !reflect.DeepEqual(vmSnapshot, vmSnapshotCpy) {
		if _, err := ctrl.client.VirtualMachineSnapshot(vmSnapshotCpy.Namespace).Update(vmSnapshotCpy); err != nil {
			return err
		}
	}

	return nil
}

func (ctrl *SnapshotController) getVM(vmSnapshot *vmsnapshotv1alpha1.VirtualMachineSnapshot) (*kubevirtv1.VirtualMachine, error) {
	vmName := vmSnapshot.Spec.Source.VirtualMachineName
	if vmName == nil {
		return nil, fmt.Errorf("VirtualMachine name not specified")
	}

	obj, exists, err := ctrl.vmInformer.GetStore().GetByKey(cacheKeyFunc(vmSnapshot.Namespace, *vmName))
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

	_, err := s.client.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
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

	_, err := s.client.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
	if err != nil {
		return err
	}

	return nil
}

func (s *vmSnapshotSource) Spec() vmsnapshotv1alpha1.SourceSpec {
	vmCpy := s.vm.DeepCopy()
	vmCpy.Status = kubevirtv1.VirtualMachineStatus{}
	return vmsnapshotv1alpha1.SourceSpec{
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
