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
 *
 */

package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/controller"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	storageutils "kubevirt.io/kubevirt/pkg/storage/utils"
	utils "kubevirt.io/kubevirt/pkg/util"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
	launcherapi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	sourceFinalizer = "snapshot.kubevirt.io/snapshot-source-protection"
	failedFreezeMsg = "Failed freezing vm"
)

var (
	ErrVolumeDoesntExist  = errors.New("volume doesnt exist")
	ErrVolumeNotBound     = errors.New("volume not bound")
	ErrVolumeNotPopulated = errors.New("volume not populated")
)

type snapshotSource interface {
	UpdateSourceState() error
	UID() types.UID
	Locked() bool
	LockMsg() string
	Lock() (bool, error)
	Unlock() (bool, error)
	Online() bool
	GuestAgent() bool
	Frozen() bool
	Freeze() error
	Unfreeze() error
	Spec() (snapshotv1.SourceSpec, error)
	PersistentVolumeClaims() (map[string]string, error)
}

type sourceState struct {
	online     bool
	guestAgent bool
	frozen     bool
	locked     bool
	lockMsg    string
}

type vmSnapshotSource struct {
	vm         *kubevirtv1.VirtualMachine
	snapshot   *snapshotv1.VirtualMachineSnapshot
	controller *VMSnapshotController
	state      *sourceState
}

func (s *vmSnapshotSource) UpdateSourceState() error {
	vmi, exists, err := s.controller.getVMI(s.vm)
	if err != nil {
		return err
	}

	online := exists

	condManager := controller.NewVirtualMachineInstanceConditionManager()
	guestAgent := exists && condManager.HasCondition(vmi, kubevirtv1.VirtualMachineInstanceAgentConnected)

	locked := s.vm.Status.SnapshotInProgress != nil &&
		*s.vm.Status.SnapshotInProgress == s.snapshot.Name &&
		controller.HasFinalizer(s.vm, sourceFinalizer)
	lockMsg := "Source not locked"
	if locked {
		lockMsg = "Source locked and operation in progress"
	}
	frozen := exists && vmi.Status.FSFreezeStatus == launcherapi.FSFrozen

	s.state = &sourceState{
		online:     online,
		guestAgent: guestAgent,
		locked:     locked,
		frozen:     frozen,
		lockMsg:    lockMsg,
	}

	return nil
}

func (s *vmSnapshotSource) UID() types.UID {
	return s.vm.UID
}

func (s *vmSnapshotSource) Locked() bool {
	return s.state.locked
}

func (s *vmSnapshotSource) LockMsg() string {
	return s.state.lockMsg
}

func (s *vmSnapshotSource) Lock() (bool, error) {
	if s.Locked() {
		return true, nil
	}

	pvcNames, err := s.pvcNames()
	if err != nil {
		if storageutils.IsErrNoBackendPVC(err) {
			// No backend pvc when we should have one, lets wait
			// TODO: Improve this error handling
			return false, nil
		}
		return false, err
	}

	err = s.verifyVolumes(pvcNames.List())
	if err != nil {
		switch errors.Unwrap(err) {
		case ErrVolumeDoesntExist, ErrVolumeNotBound, ErrVolumeNotPopulated:
			s.state.lockMsg += fmt.Sprintf(" source %s/%s %s", s.vm.Namespace, s.vm.Name, err.Error())
			log.Log.Error(s.state.lockMsg)
			return false, nil
		default:
			return false, err
		}
	}

	if !s.Online() {
		pods, err := watchutil.PodsUsingPVCs(s.controller.PodInformer, s.vm.Namespace, pvcNames)
		if err != nil {
			return false, err
		}

		if len(pods) > 0 {
			s.state.lockMsg += fmt.Sprintf(" source is offline but %d pods using PVCs %+v", len(pods), slices.Collect(maps.Keys(pvcNames)))
			log.Log.V(3).Info(s.state.lockMsg)
			return false, nil
		}
	}

	if s.vm.Status.SnapshotInProgress != nil && *s.vm.Status.SnapshotInProgress != s.snapshot.Name {
		s.state.lockMsg += fmt.Sprintf(" snapshot %q in progress", *s.vm.Status.SnapshotInProgress)
		log.Log.V(3).Info(s.state.lockMsg)
		return false, nil
	}

	vmCopy := s.vm.DeepCopy()

	if vmCopy.Status.SnapshotInProgress == nil {
		vmCopy.Status.SnapshotInProgress = &s.snapshot.Name
		vmCopy, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).UpdateStatus(context.Background(), vmCopy, metav1.UpdateOptions{})
		if err != nil {
			return false, err
		}
	}

	if !controller.HasFinalizer(vmCopy, sourceFinalizer) {
		log.Log.Infof("Adding VM snapshot finalizer to %s", s.vm.Name)
		controller.AddFinalizer(vmCopy, sourceFinalizer)
		patch, err := generateFinalizerPatch(s.vm.Finalizers, vmCopy.Finalizers)
		if err != nil {
			return false, err
		}

		vmCopy, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).Patch(context.Background(), vmCopy.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
		if err != nil {
			return false, err
		}
	}

	s.vm = vmCopy
	s.state.locked = true
	s.state.lockMsg = "Source locked and operation in progress"

	return true, nil
}

func (s *vmSnapshotSource) Unlock() (bool, error) {
	if s.vm.Status.SnapshotInProgress == nil || *s.vm.Status.SnapshotInProgress != s.snapshot.Name {
		return false, nil
	}

	var err error
	vmCopy := s.vm.DeepCopy()

	if controller.HasFinalizer(vmCopy, sourceFinalizer) {
		controller.RemoveFinalizer(vmCopy, sourceFinalizer)
		patch, err := generateFinalizerPatch(s.vm.Finalizers, vmCopy.Finalizers)
		if err != nil {
			return false, err
		}

		vmCopy, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).Patch(context.Background(), vmCopy.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
		if err != nil {
			return false, err
		}
	}

	vmCopy.Status.SnapshotInProgress = nil
	vmCopy, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).UpdateStatus(context.Background(), vmCopy, metav1.UpdateOptions{})
	if err != nil {
		return false, err
	}

	s.vm = vmCopy

	return true, nil
}

func (s *vmSnapshotSource) verifyVolumes(pvcNames []string) error {
	for _, pvcName := range pvcNames {
		obj, exists, err := s.controller.PVCInformer.GetStore().GetByKey(cacheKeyFunc(s.vm.Namespace, pvcName))
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("%w: %s", ErrVolumeDoesntExist, pvcName)
		}

		pvc := obj.(*corev1.PersistentVolumeClaim).DeepCopy()
		if pvc.Status.Phase != corev1.ClaimBound {
			return fmt.Errorf("%w: %s", ErrVolumeNotBound, pvcName)
		}
		getDVFunc := func(name, namespace string) (*cdiv1.DataVolume, error) {
			dv, err := storagetypes.GetDataVolumeFromCache(namespace, name, s.controller.DVInformer.GetStore())
			if err != nil {
				return nil, err
			}
			if dv == nil {
				return nil, fmt.Errorf("Data volume %s/%s doesnt exist", namespace, name)
			}
			return dv, err
		}
		if populated, err := cdiv1.IsPopulated(pvc, getDVFunc); !populated || err != nil {
			if err != nil {
				return err
			}
			return fmt.Errorf("%w: %s", ErrVolumeNotPopulated, pvcName)
		}
	}

	return nil
}

func (s *vmSnapshotSource) getVMRevision() (*snapshotv1.VirtualMachine, error) {
	vmi, exists, err := s.controller.getVMI(s.vm)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("can't get vm revision, vmi doesn't exist")
	}

	crName := vmi.Status.VirtualMachineRevisionName
	storeObj, exists, err := s.controller.CRInformer.GetStore().GetByKey(cacheKeyFunc(vmi.Namespace, crName))
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("vm revision %s doesn't exist", crName)
	}

	cr, ok := storeObj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, fmt.Errorf("unexpected resource %+v", storeObj)
	}

	vmRevision := &snapshotv1.VirtualMachine{}
	err = json.Unmarshal(cr.Data.Raw, vmRevision)
	if err != nil {
		return nil, err
	}
	return vmRevision, nil
}

func (s *vmSnapshotSource) getControllerRevision(namespace, name string) (*appsv1.ControllerRevision, error) {
	crKey := cacheKeyFunc(namespace, name)
	obj, exists, err := s.controller.CRInformer.GetStore().GetByKey(crKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Unable to fetch ControllerRevision %s", crKey)
	}
	cr, ok := obj.(*appsv1.ControllerRevision)
	if !ok {
		return nil, fmt.Errorf("Unexpected object format returned by informer")
	}
	return cr, nil
}

func (s *vmSnapshotSource) captureInstancetypeControllerRevision(namespace, revisionName string) (string, error) {
	existingCR, err := s.getControllerRevision(namespace, revisionName)
	if err != nil {
		return "", err
	}

	snapshotCR := existingCR.DeepCopy()
	snapshotCR.ObjectMeta.Reset()
	snapshotCR.ObjectMeta.SetLabels(existingCR.Labels)

	// We strip out the source VM name from the CR name and replace it with the snapshot name
	snapshotCR.Name = strings.Replace(existingCR.Name, s.snapshot.Spec.Source.Name, s.snapshot.Name, 1)

	// Ensure GVK is set before we attempt to create the controller OwnerReference below
	obj, err := utils.GenerateKubeVirtGroupVersionKind(s.snapshot)
	if err != nil {
		return "", err
	}
	snapshot, ok := obj.(*snapshotv1.VirtualMachineSnapshot)
	if !ok {
		return "", fmt.Errorf("Unexpected object format returned from GenerateKubeVirtGroupVersionKind")
	}
	snapshotCR.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(snapshot, snapshot.GroupVersionKind())}

	snapshotCR, err = s.controller.Client.AppsV1().ControllerRevisions(s.snapshot.Namespace).Create(context.Background(), snapshotCR, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return "", err
	}

	return snapshotCR.Name, nil
}

func (s *vmSnapshotSource) captureInstancetypeControllerRevisions(vm *snapshotv1.VirtualMachine) error {
	if vm.Spec.Instancetype != nil && vm.Spec.Instancetype.RevisionName != "" {
		snapshotCRName, err := s.captureInstancetypeControllerRevision(vm.Namespace, vm.Spec.Instancetype.RevisionName)
		if err != nil {
			return err
		}
		vm.Spec.Instancetype.RevisionName = snapshotCRName
	}

	if vm.Spec.Preference != nil && vm.Spec.Preference.RevisionName != "" {
		snapshotCRName, err := s.captureInstancetypeControllerRevision(vm.Namespace, vm.Spec.Preference.RevisionName)
		if err != nil {
			return err
		}
		vm.Spec.Preference.RevisionName = snapshotCRName
	}

	return nil
}

func (s *vmSnapshotSource) Spec() (snapshotv1.SourceSpec, error) {
	vmCpy := &snapshotv1.VirtualMachine{}
	metaObj := *getSimplifiedMetaObject(s.vm.ObjectMeta)

	if s.Online() {
		var err error
		vmCpy, err = s.getVMRevision()
		if err != nil {
			return snapshotv1.SourceSpec{}, err
		}
		vmCpy.ObjectMeta = metaObj

		vmCpy.Spec.Template.Spec.Volumes = s.vm.Spec.Template.Spec.Volumes
		vmCpy.Spec.Template.Spec.Domain.Devices.Disks = s.vm.Spec.Template.Spec.Domain.Devices.Disks
		vmCpy.Spec.DataVolumeTemplates = s.vm.Spec.DataVolumeTemplates
	} else {
		vmCpy.ObjectMeta = metaObj
		vmCpy.Spec = *s.vm.Spec.DeepCopy()
		vmCpy.Status = kubevirtv1.VirtualMachineStatus{}
	}

	if err := s.captureInstancetypeControllerRevisions(vmCpy); err != nil {
		return snapshotv1.SourceSpec{}, err
	}

	return snapshotv1.SourceSpec{
		VirtualMachine: vmCpy,
	}, nil
}

func (s *vmSnapshotSource) Online() bool {
	return s.state.online
}

func (s *vmSnapshotSource) GuestAgent() bool {
	return s.state.guestAgent
}

func (s *vmSnapshotSource) Frozen() bool {
	return s.state.frozen
}

func (s *vmSnapshotSource) Freeze() error {
	if !s.Locked() {
		return fmt.Errorf("attempting to freeze unlocked VM")
	}
	if s.Frozen() {
		return nil
	}

	if !s.GuestAgent() {
		if s.Online() {
			log.Log.Warningf("Guest agent does not exist and VM %s is running. Snapshoting without freezing FS. This can result in inconsistent snapshot!", s.vm.Name)
		}
		return nil
	}

	log.Log.V(3).Infof("Freezing vm %s file system before taking the snapshot", s.vm.Name)

	startTime := time.Now()
	err := s.controller.Client.VirtualMachineInstance(s.vm.Namespace).Freeze(context.Background(), s.vm.Name, getFailureDeadline(s.snapshot))
	timeTrack(startTime, fmt.Sprintf("Freezing vmi %s", s.vm.Name))
	if err != nil {
		formattedErr := fmt.Errorf("%s %s: %v", failedFreezeMsg, s.vm.Name, err)
		log.Log.Errorf(formattedErr.Error())
		return formattedErr
	}
	s.state.frozen = true

	return nil
}

func (s *vmSnapshotSource) Unfreeze() error {
	if !s.Locked() || !s.GuestAgent() {
		return nil
	}

	log.Log.V(3).Infof("Unfreezing vm %s file system after taking the snapshot", s.vm.Name)

	defer timeTrack(time.Now(), fmt.Sprintf("Unfreezing vmi %s", s.vm.Name))
	err := s.controller.Client.VirtualMachineInstance(s.vm.Namespace).Unfreeze(context.Background(), s.vm.Name)
	if err != nil {
		return err
	}
	s.state.frozen = false

	return nil
}

func (s *vmSnapshotSource) PersistentVolumeClaims() (map[string]string, error) {
	volumes, err := storageutils.GetVolumes(s.vm, s.controller.Client, storageutils.WithAllVolumes)
	if err != nil {
		return map[string]string{}, err
	}
	return storagetypes.GetPVCsFromVolumes(volumes), nil
}

func (s *vmSnapshotSource) pvcNames() (sets.String, error) {
	ss := sets.NewString()
	pvcs, err := s.PersistentVolumeClaims()
	if err != nil {
		return ss, err
	}
	for _, pvc := range pvcs {
		ss.Insert(pvc)
	}
	return ss, nil
}
