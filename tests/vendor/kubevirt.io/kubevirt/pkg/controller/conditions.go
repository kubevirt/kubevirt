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
 * Copyright 2019 Red Hat, Inc.
 *
 */
package controller

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	cdiv1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"

	v1 "kubevirt.io/client-go/apis/core/v1"
)

type VirtualMachineConditionManager struct {
}

func NewVirtualMachineConditionManager() *VirtualMachineConditionManager {
	return &VirtualMachineConditionManager{}
}

func (d *VirtualMachineConditionManager) GetCondition(vm *v1.VirtualMachine, cond v1.VirtualMachineConditionType) *v1.VirtualMachineCondition {
	if vm == nil {
		return nil
	}
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			return &c
		}
	}
	return nil
}

func (d *VirtualMachineConditionManager) HasCondition(vm *v1.VirtualMachine, cond v1.VirtualMachineConditionType) bool {
	return d.GetCondition(vm, cond) != nil
}

func (d *VirtualMachineConditionManager) RemoveCondition(vm *v1.VirtualMachine, cond v1.VirtualMachineConditionType) {
	var conds []v1.VirtualMachineCondition
	for _, c := range vm.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	vm.Status.Conditions = conds
}

type VirtualMachineInstanceConditionManager struct {
}

// UpdateCondition updates the given VirtualMachineCondition, unless it is already set with the same status and reason.
func (d *VirtualMachineConditionManager) UpdateCondition(vm *v1.VirtualMachine, cond *v1.VirtualMachineCondition) {
	for i, c := range vm.Status.Conditions {
		if c.Type != cond.Type {
			continue
		}

		if c.Status != cond.Status || c.Reason != cond.Reason {
			vm.Status.Conditions[i] = *cond
		}

		return
	}

	vm.Status.Conditions = append(vm.Status.Conditions, *cond)
}

func (d *VirtualMachineInstanceConditionManager) CheckFailure(vmi *v1.VirtualMachineInstance, syncErr error, reason string) (changed bool) {
	if syncErr != nil {
		if d.HasConditionWithStatusAndReason(vmi, v1.VirtualMachineInstanceSynchronized, k8sv1.ConditionFalse, reason) {
			return false
		}
		if d.HasCondition(vmi, v1.VirtualMachineInstanceSynchronized) {
			d.RemoveCondition(vmi, v1.VirtualMachineInstanceSynchronized)
		}
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			Type:               v1.VirtualMachineInstanceSynchronized,
			Reason:             reason,
			Message:            syncErr.Error(),
			LastTransitionTime: metav1.Now(),
			Status:             k8sv1.ConditionFalse,
		})
		return true
	} else if d.HasCondition(vmi, v1.VirtualMachineInstanceSynchronized) {
		d.RemoveCondition(vmi, v1.VirtualMachineInstanceSynchronized)
		return true
	}
	return false
}

func (d *VirtualMachineInstanceConditionManager) GetCondition(vmi *v1.VirtualMachineInstance, cond v1.VirtualMachineInstanceConditionType) *v1.VirtualMachineInstanceCondition {
	if vmi == nil {
		return nil
	}
	for _, c := range vmi.Status.Conditions {
		if c.Type == cond {
			return &c
		}
	}
	return nil
}

func (d *VirtualMachineInstanceConditionManager) HasCondition(vmi *v1.VirtualMachineInstance, cond v1.VirtualMachineInstanceConditionType) bool {
	return d.GetCondition(vmi, cond) != nil
}

func (d *VirtualMachineInstanceConditionManager) HasConditionWithStatus(vmi *v1.VirtualMachineInstance, cond v1.VirtualMachineInstanceConditionType, status k8sv1.ConditionStatus) bool {
	c := d.GetCondition(vmi, cond)
	return c != nil && c.Status == status
}

func (d *VirtualMachineInstanceConditionManager) HasConditionWithStatusAndReason(vmi *v1.VirtualMachineInstance, cond v1.VirtualMachineInstanceConditionType, status k8sv1.ConditionStatus, reason string) bool {
	c := d.GetCondition(vmi, cond)
	return c != nil && c.Status == status && c.Reason == reason
}

func (d *VirtualMachineInstanceConditionManager) RemoveCondition(vmi *v1.VirtualMachineInstance, cond v1.VirtualMachineInstanceConditionType) {
	var conds []v1.VirtualMachineInstanceCondition
	for _, c := range vmi.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	vmi.Status.Conditions = conds
}

// UpdateCondition updates the given VirtualMachineInstanceCondition, unless it is already set with the same status and reason.
func (d *VirtualMachineInstanceConditionManager) UpdateCondition(vmi *v1.VirtualMachineInstance, cond *v1.VirtualMachineInstanceCondition) {
	for i, c := range vmi.Status.Conditions {
		if c.Type != cond.Type {
			continue
		}

		if c.Status != cond.Status || c.Reason != cond.Reason {
			vmi.Status.Conditions[i] = *cond
		}

		return
	}

	vmi.Status.Conditions = append(vmi.Status.Conditions, *cond)
}

// AddPodCondition add pod condition to the VM.
func (d *VirtualMachineInstanceConditionManager) AddPodCondition(vmi *v1.VirtualMachineInstance, cond *k8sv1.PodCondition) {
	if !d.HasCondition(vmi, v1.VirtualMachineInstanceConditionType(cond.Type)) {
		vmi.Status.Conditions = append(vmi.Status.Conditions, v1.VirtualMachineInstanceCondition{
			LastProbeTime:      cond.LastProbeTime,
			LastTransitionTime: cond.LastTransitionTime,
			Message:            cond.Message,
			Reason:             cond.Reason,
			Status:             cond.Status,
			Type:               v1.VirtualMachineInstanceConditionType(cond.Type),
		})
	}
}

func (d *VirtualMachineInstanceConditionManager) PodHasCondition(pod *k8sv1.Pod, conditionType k8sv1.PodConditionType, status k8sv1.ConditionStatus) bool {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == conditionType {
			if cond.Status == status {
				return true
			} else {
				return false
			}
		}
	}
	return false
}

func (d *VirtualMachineInstanceConditionManager) GetPodConditionWithStatus(pod *k8sv1.Pod, conditionType k8sv1.PodConditionType, status k8sv1.ConditionStatus) *k8sv1.PodCondition {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == conditionType {
			if cond.Status == status {
				return &cond
			} else {
				return nil
			}
		}
	}
	return nil
}

func (d *VirtualMachineInstanceConditionManager) GetPodCondition(pod *k8sv1.Pod, conditionType k8sv1.PodConditionType) *k8sv1.PodCondition {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == conditionType {
			return &cond
		}
	}
	return nil
}

func NewVirtualMachineInstanceConditionManager() *VirtualMachineInstanceConditionManager {
	return &VirtualMachineInstanceConditionManager{}
}

type VirtualMachineInstanceMigrationConditionManager struct {
}

func (d *VirtualMachineInstanceMigrationConditionManager) HasCondition(migration *v1.VirtualMachineInstanceMigration, cond v1.VirtualMachineInstanceMigrationConditionType) bool {
	for _, c := range migration.Status.Conditions {
		if c.Type == cond {
			return true
		}
	}
	return false
}

func (d *VirtualMachineInstanceMigrationConditionManager) HasConditionWithStatus(migration *v1.VirtualMachineInstanceMigration, cond v1.VirtualMachineInstanceMigrationConditionType, status k8sv1.ConditionStatus) bool {
	for _, c := range migration.Status.Conditions {
		if c.Type == cond {
			if c.Status == status {
				return true
			}
			return false
		}
	}
	return false
}

func (d *VirtualMachineInstanceMigrationConditionManager) RemoveCondition(migration *v1.VirtualMachineInstanceMigration, cond v1.VirtualMachineInstanceMigrationConditionType) {
	var conds []v1.VirtualMachineInstanceMigrationCondition
	for _, c := range migration.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	migration.Status.Conditions = conds
}
func NewVirtualMachineInstanceMigrationConditionManager() *VirtualMachineInstanceMigrationConditionManager {
	return &VirtualMachineInstanceMigrationConditionManager{}
}

type DataVolumeConditionManager struct {
}

func (d *DataVolumeConditionManager) GetCondition(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeConditionType) *cdiv1.DataVolumeCondition {
	if dv == nil {
		return nil
	}
	for _, c := range dv.Status.Conditions {
		if c.Type == cond {
			return &c
		}
	}
	return nil
}

func (d *DataVolumeConditionManager) HasCondition(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeConditionType) bool {
	return d.GetCondition(dv, cond) != nil
}

func (d *DataVolumeConditionManager) HasConditionWithStatus(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeConditionType, status k8sv1.ConditionStatus) bool {
	c := d.GetCondition(dv, cond)
	return c != nil && c.Status == status
}

func (d *DataVolumeConditionManager) HasConditionWithStatusAndReason(dv *cdiv1.DataVolume, cond cdiv1.DataVolumeConditionType, status k8sv1.ConditionStatus, reason string) bool {
	c := d.GetCondition(dv, cond)
	return c != nil && c.Status == status && c.Reason == reason
}

func NewDataVolumeConditionManager() *DataVolumeConditionManager {
	return &DataVolumeConditionManager{}
}
