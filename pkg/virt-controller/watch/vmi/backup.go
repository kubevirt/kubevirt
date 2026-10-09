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

package vmi

import (
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	backupv1 "kubevirt.io/api/backup/v1alpha1"
	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/controller"
)

const offlineBackupInProgressReason = "OfflineBackupInProgress"

func (c *Controller) addVMBackup(obj any) {
	if !c.clusterConfig.OfflineIncrementalBackupEnabled() {
		return
	}
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}
	backup, ok := obj.(*backupv1.VirtualMachineBackup)
	if !ok {
		return
	}
	vmName := c.resolveBackupVMName(backup)
	if vmName == "" {
		return
	}
	c.Queue.Add(types.NamespacedName{Namespace: backup.Namespace, Name: vmName}.String())
}

func (c *Controller) offlineBackupInProgress(vmi *virtv1.VirtualMachineInstance) bool {
	if !c.clusterConfig.OfflineIncrementalBackupEnabled() {
		return false
	}
	for _, obj := range c.vmBackupStore.List() {
		backup, ok := obj.(*backupv1.VirtualMachineBackup)
		if !ok || backup.Namespace != vmi.Namespace {
			continue
		}
		if !isOfflineBackupProgressing(backup) {
			continue
		}
		if c.resolveBackupVMName(backup) == vmi.Name {
			return true
		}
	}
	return false
}

func (c *Controller) resolveBackupVMName(backup *backupv1.VirtualMachineBackup) string {
	if backup.Spec.Source.Kind == backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind {
		key := types.NamespacedName{Namespace: backup.Namespace, Name: backup.Spec.Source.Name}.String()
		obj, exists, err := c.vmBackupTrackerStore.GetByKey(key)
		if err != nil || !exists {
			return ""
		}
		return obj.(*backupv1.VirtualMachineBackupTracker).Spec.Source.Name
	}
	return backup.Spec.Source.Name
}

func isOfflineBackupProgressing(backup *backupv1.VirtualMachineBackup) bool {
	if backup.Status == nil || backup.Status.Offline == nil || !*backup.Status.Offline {
		return false
	}
	if backup.DeletionTimestamp != nil {
		return false
	}
	if meta.IsStatusConditionTrue(backup.Status.Conditions, string(backupv1.ConditionComplete)) ||
		meta.IsStatusConditionTrue(backup.Status.Conditions, string(backupv1.ConditionFailed)) {
		return false
	}
	return meta.IsStatusConditionTrue(backup.Status.Conditions, string(backupv1.ConditionProgressing))
}

func (c *Controller) syncBackupInProgressCondition(vmi *virtv1.VirtualMachineInstance) {
	cm := controller.NewVirtualMachineInstanceConditionManager()
	inProgress := c.offlineBackupInProgress(vmi)
	hasCondition := cm.HasCondition(vmi, virtv1.VirtualMachineInstanceBackupInProgress)

	if inProgress && !hasCondition {
		cm.UpdateCondition(vmi, &virtv1.VirtualMachineInstanceCondition{
			Type:               virtv1.VirtualMachineInstanceBackupInProgress,
			Status:             k8sv1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             offlineBackupInProgressReason,
			Message:            "virt-launcher creation is held while an offline backup is in progress",
		})
	} else if !inProgress && hasCondition {
		cm.RemoveCondition(vmi, virtv1.VirtualMachineInstanceBackupInProgress)
	}
}
