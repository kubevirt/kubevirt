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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package snapshot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	appsv1 "k8s.io/api/apps/v1"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"
	utils "kubevirt.io/kubevirt/pkg/util"
	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
	launcherapi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	sourceFinalizer = "snapshot.kubevirt.io/snapshot-source-protection"
)

type snapshotSource interface {
	UID() types.UID
	Locked() bool
	Lock() (bool, error)
	Unlock() (bool, error)
	Online() (bool, error)
	GuestAgent() (bool, error)
	Frozen() (bool, error)
	Freeze() error
	Unfreeze() error
	Spec() (snapshotv1.SourceSpec, error)
	PersistentVolumeClaims() (map[string]string, error)
}

type vmSnapshotSource struct {
	vm         *kubevirtv1.VirtualMachine
	snapshot   *snapshotv1.VirtualMachineSnapshot
	controller *VMSnapshotController
}

func (s *vmSnapshotSource) UID() types.UID {
	return s.vm.UID
}

func (s *vmSnapshotSource) Locked() bool {
	return s.vm.Status.SnapshotInProgress != nil &&
		*s.vm.Status.SnapshotInProgress == s.snapshot.Name &&
		controller.HasFinalizer(s.vm, sourceFinalizer)
}

func (s *vmSnapshotSource) Lock() (bool, error) {
	if s.Locked() {
		return true, nil
	}

	vmOnline, err := s.Online()
	if err != nil {
		return false, err
	}

	if !vmOnline {
		pvcNames := s.pvcNames()
		pods, err := watchutil.PodsUsingPVCs(s.controller.PodInformer, s.vm.Namespace, pvcNames)
		if err != nil {
			return false, err
		}

		if len(pods) > 0 {
			log.Log.V(3).Infof("Vm is offline but %d pods using PVCs %+v", len(pods), pvcNames)
			return false, nil
		}
	}

	if s.vm.Status.SnapshotInProgress != nil && *s.vm.Status.SnapshotInProgress != s.snapshot.Name {
		log.Log.V(3).Infof("Snapshot %s in progress", *s.vm.Status.SnapshotInProgress)
		return false, nil
	}

	vmCopy := s.vm.DeepCopy()

	if vmCopy.Status.SnapshotInProgress == nil {
		vmCopy.Status.SnapshotInProgress = &s.snapshot.Name
		// unfortunately, status updater does not return the updated resource
		// but the controller is watching VMs so will get notified
		// returning here because following Update will always block
		return false, s.controller.vmStatusUpdater.UpdateStatus(vmCopy)
	}

	if !controller.HasFinalizer(vmCopy, sourceFinalizer) {
		log.Log.Infof("Adding VM snapshot finalizer to %s", s.vm.Name)
		controller.AddFinalizer(vmCopy, sourceFinalizer)
		_, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).Update(context.Background(), vmCopy)
		if err != nil {
			return false, err
		}
	}

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
		vmCopy, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).Update(context.Background(), vmCopy)
		if err != nil {
			return false, err
		}
	}

	vmCopy.Status.SnapshotInProgress = nil
	err = s.controller.vmStatusUpdater.UpdateStatus(vmCopy)
	if err != nil {
		return true, err
	}

	return true, nil
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
	if err != nil && !errors.IsAlreadyExists(err) {
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
	online, err := s.Online()
	if err != nil {
		return snapshotv1.SourceSpec{}, err
	}

	vmCpy := &snapshotv1.VirtualMachine{}
	metaObj := *getSimplifiedMetaObject(s.vm.ObjectMeta)

	if online {
		vmCpy, err = s.getVMRevision()
		if err != nil {
			return snapshotv1.SourceSpec{}, err
		}
		vmCpy.ObjectMeta = metaObj

		vmCpy.Spec.Template.Spec.Volumes = s.vm.Spec.Template.Spec.Volumes
		vmCpy.Spec.Template.Spec.Domain.Devices.Disks = s.vm.Spec.Template.Spec.Domain.Devices.Disks
	} else {
		vmCpy.ObjectMeta = metaObj
		vmCpy.Spec = *s.vm.Spec.DeepCopy()
		vmCpy.Status = kubevirtv1.VirtualMachineStatus{}
	}

	if err = s.captureInstancetypeControllerRevisions(vmCpy); err != nil {
		return snapshotv1.SourceSpec{}, err
	}

	return snapshotv1.SourceSpec{
		VirtualMachine: vmCpy,
	}, nil
}

func (s *vmSnapshotSource) Online() (bool, error) {
	vmRunning, err := checkVMRunning(s.vm)
	if err != nil {
		return false, err
	}

	exists, err := s.controller.checkVMIRunning(s.vm)
	if err != nil {
		return false, err
	}

	return (vmRunning || exists), nil
}

func (s *vmSnapshotSource) GuestAgent() (bool, error) {
	condManager := controller.NewVirtualMachineInstanceConditionManager()
	vmi, exists, err := s.controller.getVMI(s.vm)
	if err != nil || !exists {
		return false, err
	}

	return condManager.HasCondition(vmi, kubevirtv1.VirtualMachineInstanceAgentConnected), nil
}

func (s *vmSnapshotSource) Frozen() (bool, error) {
	vmi, exists, err := s.controller.getVMI(s.vm)
	if err != nil || !exists {
		return false, err
	}

	return vmi.Status.FSFreezeStatus == launcherapi.FSFrozen, nil
}

func (s *vmSnapshotSource) Freeze() error {
	if !s.Locked() {
		return fmt.Errorf("attempting to freeze unlocked VM")
	}

	exists, err := s.GuestAgent()
	if !exists || err != nil {
		return err
	}

	log.Log.V(3).Infof("Freezing vm %s file system before taking the snapshot", s.vm.Name)

	startTime := time.Now()
	err = s.controller.Client.VirtualMachineInstance(s.vm.Namespace).Freeze(context.Background(), s.vm.Name, getFailureDeadline(s.snapshot))
	timeTrack(startTime, fmt.Sprintf("Freezing vmi %s", s.vm.Name))
	if err != nil {
		return err
	}

	return nil
}

func (s *vmSnapshotSource) Unfreeze() error {
	if !s.Locked() {
		return nil
	}

	exists, err := s.GuestAgent()
	if !exists || err != nil {
		return err
	}

	log.Log.V(3).Infof("Unfreezing vm %s file system after taking the snapshot", s.vm.Name)

	defer timeTrack(time.Now(), fmt.Sprintf("Unfreezing vmi %s", s.vm.Name))
	err = s.controller.Client.VirtualMachineInstance(s.vm.Namespace).Unfreeze(context.Background(), s.vm.Name)
	if err != nil {
		return err
	}

	return nil
}

func (s *vmSnapshotSource) PersistentVolumeClaims() (map[string]string, error) {
	return storagetypes.GetPVCsFromVolumes(s.vm.Spec.Template.Spec.Volumes), nil
}

func (s *vmSnapshotSource) pvcNames() sets.String {
	pvcs := storagetypes.GetPVCsFromVolumes(s.vm.Spec.Template.Spec.Volumes)
	ss := sets.NewString()
	for _, pvc := range pvcs {
		ss.Insert(pvc)
	}
	return ss
}
