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

	v1 "kubevirt.io/api/core/v1"
	poolv1 "kubevirt.io/api/pool/v1alpha1"
)

type VirtualMachinePoolConditionManager struct {
}

func NewVirtualMachinePoolConditionManager() *VirtualMachinePoolConditionManager {
	return &VirtualMachinePoolConditionManager{}
}

func (d *VirtualMachinePoolConditionManager) GetCondition(pool *poolv1.VirtualMachinePool, cond poolv1.VirtualMachinePoolConditionType) *poolv1.VirtualMachinePoolCondition {
	if pool == nil {
		return nil
	}
	for _, c := range pool.Status.Conditions {
		if c.Type == cond {
			return &c
		}
	}
	return nil
}

func (d *VirtualMachinePoolConditionManager) HasCondition(pool *poolv1.VirtualMachinePool, cond poolv1.VirtualMachinePoolConditionType) bool {
	return d.GetCondition(pool, cond) != nil
}

func (d *VirtualMachinePoolConditionManager) RemoveCondition(pool *poolv1.VirtualMachinePool, cond poolv1.VirtualMachinePoolConditionType) {
	var conds []poolv1.VirtualMachinePoolCondition
	for _, c := range pool.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	pool.Status.Conditions = conds
}

// UpdateCondition updates the given VirtualMachinePoolCondition, unless it is already set with the same status and reason.
func (d *VirtualMachinePoolConditionManager) UpdateCondition(pool *poolv1.VirtualMachinePool, cond *poolv1.VirtualMachinePoolCondition) {
	for i, c := range pool.Status.Conditions {
		if c.Type != cond.Type {
			continue
		}

		if c.Status != cond.Status || c.Reason != cond.Reason {
			pool.Status.Conditions[i] = *cond
		}

		return
	}

	pool.Status.Conditions = append(pool.Status.Conditions, *cond)
}

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

func (d *VirtualMachineConditionManager) HasConditionWithStatus(vm *v1.VirtualMachine, cond v1.VirtualMachineConditionType, status k8sv1.ConditionStatus) bool {
	c := d.GetCondition(vm, cond)
	return c != nil && c.Status == status
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

func NewVirtualMachineInstanceConditionManager() *VirtualMachineInstanceConditionManager {
	return &VirtualMachineInstanceConditionManager{}
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

func (d *VirtualMachineInstanceConditionManager) GetPodCondition(pod *k8sv1.Pod, conditionType k8sv1.PodConditionType) *k8sv1.PodCondition {
	for _, cond := range pod.Status.Conditions {
		if cond.Type == conditionType {
			return &cond
		}
	}
	return nil
}

func (d *VirtualMachineInstanceConditionManager) ConditionsEqual(vmi1, vmi2 *v1.VirtualMachineInstance) bool {
	if len(vmi1.Status.Conditions) != len(vmi2.Status.Conditions) {
		return false
	}

	for _, cond1 := range vmi1.Status.Conditions {
		if !d.HasConditionWithStatusAndReason(vmi2, cond1.Type, cond1.Status, cond1.Reason) {
			return false
		}
	}

	return true
}

type VirtualMachineInstanceMigrationConditionManager struct {
}

func NewVirtualMachineInstanceMigrationConditionManager() *VirtualMachineInstanceMigrationConditionManager {
	return &VirtualMachineInstanceMigrationConditionManager{}
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

type PodConditionManager struct {
}

func NewPodConditionManager() *PodConditionManager {
	return &PodConditionManager{}
}

func (d *PodConditionManager) GetCondition(pod *k8sv1.Pod, cond k8sv1.PodConditionType) *k8sv1.PodCondition {
	if pod == nil {
		return nil
	}
	for _, c := range pod.Status.Conditions {
		if c.Type == cond {
			return &c
		}
	}
	return nil
}

func (d *PodConditionManager) HasCondition(pod *k8sv1.Pod, cond k8sv1.PodConditionType) bool {
	return d.GetCondition(pod, cond) != nil
}

func (d *PodConditionManager) HasConditionWithStatus(pod *k8sv1.Pod, cond k8sv1.PodConditionType, status k8sv1.ConditionStatus) bool {
	c := d.GetCondition(pod, cond)
	return c != nil && c.Status == status
}

func (d *PodConditionManager) HasConditionWithStatusAndReason(pod *k8sv1.Pod, cond k8sv1.PodConditionType, status k8sv1.ConditionStatus, reason string) bool {
	c := d.GetCondition(pod, cond)
	return c != nil && c.Status == status && c.Reason == reason
}

func (d *PodConditionManager) RemoveCondition(pod *k8sv1.Pod, cond k8sv1.PodConditionType) {
	var conds []k8sv1.PodCondition
	for _, c := range pod.Status.Conditions {
		if c.Type == cond {
			continue
		}
		conds = append(conds, c)
	}
	pod.Status.Conditions = conds
}

// UpdateCondition updates the given PodCondition, unless it is already set with the same status and reason.
func (d *PodConditionManager) UpdateCondition(pod *k8sv1.Pod, cond *k8sv1.PodCondition) {
	for i, c := range pod.Status.Conditions {
		if c.Type != cond.Type {
			continue
		}

		if c.Status != cond.Status || c.Reason != cond.Reason {
			pod.Status.Conditions[i] = *cond
		}

		return
	}

	pod.Status.Conditions = append(pod.Status.Conditions, *cond)
}

func (d *PodConditionManager) ConditionsEqual(pod1, pod2 *k8sv1.Pod) bool {
	if len(pod1.Status.Conditions) != len(pod2.Status.Conditions) {
		return false
	}

	for _, cond1 := range pod1.Status.Conditions {
		if !d.HasConditionWithStatusAndReason(pod2, cond1.Type, cond1.Status, cond1.Reason) {
			return false
		}
	}

	return true
}
