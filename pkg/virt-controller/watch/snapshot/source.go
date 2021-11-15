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
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	appsv1 "k8s.io/api/apps/v1"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1alpha1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/controller"
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
		pods, err := podsUsingPVCs(s.controller.PodInformer, s.vm.Namespace, pvcNames)
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

	log.Log.Infof("Adding VM snapshot finalizer to %s", s.vm.Name)

	vmCopy := s.vm.DeepCopy()

	if vmCopy.Status.SnapshotInProgress == nil {
		vmCopy.Status.SnapshotInProgress = &s.snapshot.Name
		// unfortunately, status updater does not return the updated resource
		// but the controller is watching VMs so will get notified
		// returning here because following Update will always block
		return false, s.controller.vmStatusUpdater.UpdateStatus(vmCopy)
	}

	if !controller.HasFinalizer(vmCopy, sourceFinalizer) {
		controller.AddFinalizer(vmCopy, sourceFinalizer)
		_, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
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
		vmCopy, err = s.controller.Client.VirtualMachine(vmCopy.Namespace).Update(vmCopy)
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

func (s *vmSnapshotSource) getVMRevision() (*kubevirtv1.VirtualMachine, error) {
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

	vmRevision := &kubevirtv1.VirtualMachine{}
	err = json.Unmarshal(cr.Data.Raw, vmRevision)
	if err != nil {
		return nil, err
	}
	return vmRevision, nil
}

func (s *vmSnapshotSource) Spec() (snapshotv1.SourceSpec, error) {
	online, err := s.Online()
	if err != nil {
		return snapshotv1.SourceSpec{}, err
	}

	var vmCpy *kubevirtv1.VirtualMachine
	if online {
		vmCpy, err = s.getVMRevision()
		if err != nil {
			return snapshotv1.SourceSpec{}, err
		}
		labels := map[string]string{}
		for k, v := range s.vm.ObjectMeta.Labels {
			labels[k] = v
		}
		vmCpy.ObjectMeta.Labels = labels

		annotations := map[string]string{}
		for k, v := range s.vm.ObjectMeta.Annotations {
			annotations[k] = v
		}
		vmCpy.ObjectMeta.Annotations = annotations
		vmi, exists, err := s.controller.getVMI(s.vm)
		if err != nil {
			return snapshotv1.SourceSpec{}, err
		}
		if !exists {
			return snapshotv1.SourceSpec{}, fmt.Errorf("can't get online snapshot spec, vmi doesn't exist")
		}
		vmi.Spec.Volumes = s.vm.Spec.Template.Spec.Volumes
		vmi.Spec.Domain.Devices.Disks = s.vm.Spec.Template.Spec.Domain.Devices.Disks
		vmCpy.Spec.Template.Spec = vmi.Spec
	} else {
		vmCpy = s.vm.DeepCopy()
		vmCpy.Status = kubevirtv1.VirtualMachineStatus{}
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
	err = s.controller.Client.VirtualMachineInstance(s.vm.Namespace).Freeze(s.vm.Name, getFailureDeadline(s.snapshot))
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
	err = s.controller.Client.VirtualMachineInstance(s.vm.Namespace).Unfreeze(s.vm.Name)
	if err != nil {
		return err
	}

	return nil
}

func (s *vmSnapshotSource) PersistentVolumeClaims() (map[string]string, error) {
	vm := s.vm
	online, err := s.Online()
	if err != nil {
		return map[string]string{}, err
	}

	if online {
		vm, err = s.getVMRevision()
		if err != nil {
			return map[string]string{}, err
		}
	}
	return getPVCsFromVolumes(vm.Spec.Template.Spec.Volumes), nil
}

func (s *vmSnapshotSource) pvcNames() sets.String {
	pvcs := getPVCsFromVolumes(s.vm.Spec.Template.Spec.Volumes)
	ss := sets.NewString()
	for _, pvc := range pvcs {
		ss.Insert(pvc)
	}
	return ss
}
