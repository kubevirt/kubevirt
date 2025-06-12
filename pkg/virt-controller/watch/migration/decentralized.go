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

package migration

import (
	"context"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
)

func (c *Controller) initializeMigrateSourceState(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) {
	if vmi.Status.MigrationState == nil || vmi.IsMigrationCompleted() {
		vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{}
	}
	if vmi.Status.MigrationState.SourceState == nil {
		vmi.Status.MigrationState.SourceState = &virtv1.VirtualMachineInstanceMigrationSourceState{}
	}
	if vmi.Status.MigrationState.TargetState == nil {
		vmi.Status.MigrationState.TargetState = &virtv1.VirtualMachineInstanceMigrationTargetState{}
	}
	vmi.Status.MigrationState.SourceState.MigrationUID = migration.UID
	vmi.Status.MigrationState.SourceState.VirtualMachineInstanceUID = &vmi.UID

	vmi.Status.MigrationState.TargetState.SyncAddress = &migration.Spec.SendTo.ConnectURL
}

func (c *Controller) initializeMigrateTargetState(migration *virtv1.VirtualMachineInstanceMigration, vmi *virtv1.VirtualMachineInstance) {
	if vmi.Status.MigrationState == nil || vmi.IsMigrationCompleted() {
		vmi.Status.MigrationState = &virtv1.VirtualMachineInstanceMigrationState{}
	}
	if vmi.Status.MigrationState.TargetState == nil {
		vmi.Status.MigrationState.TargetState = &virtv1.VirtualMachineInstanceMigrationTargetState{}
	}
	vmi.Status.MigrationState.TargetState.Pod = vmi.Status.MigrationState.TargetPod
	vmi.Status.MigrationState.TargetState.Node = vmi.Status.MigrationState.TargetNode
	vmi.Status.MigrationState.TargetState.MigrationUID = migration.UID
}

func (c *Controller) appendMigratedVolume(vmi *virtv1.VirtualMachineInstance, claimName string, volume virtv1.Volume) error {
	key := controller.NamespacedKey(vmi.Namespace, claimName)
	obj, exists, err := c.pvcStore.GetByKey(key)
	if err != nil {
		return err
	}
	if exists {
		pvc := obj.(*k8sv1.PersistentVolumeClaim)
		vmi.Status.MigratedVolumes = append(vmi.Status.MigratedVolumes, virtv1.StorageMigratedVolumeInfo{
			VolumeName: volume.Name,
			SourcePVCInfo: &virtv1.PersistentVolumeClaimInfo{
				ClaimName:   claimName,
				AccessModes: pvc.Spec.AccessModes,
				VolumeMode:  pvc.Spec.VolumeMode,
				Requests:    pvc.Spec.Resources.Requests,
				Capacity:    pvc.Status.Capacity,
			},
			DestinationPVCInfo: &virtv1.PersistentVolumeClaimInfo{
				ClaimName:   claimName,
				AccessModes: pvc.Spec.AccessModes,
				VolumeMode:  pvc.Spec.VolumeMode,
				Requests:    pvc.Spec.Resources.Requests,
				Capacity:    pvc.Status.Capacity,
			},
		})
	}
	return nil
}

func (c *Controller) patchMigratedVolumesForDecentralizedMigration(vmi *virtv1.VirtualMachineInstance) error {
	vmiCopy := vmi.DeepCopy()
	vmiCopy.Status.MigratedVolumes = []virtv1.StorageMigratedVolumeInfo{}
	// Mark all DV/PVC volumes as migrateable in the VMI status.
	for _, volume := range vmiCopy.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			if err := c.appendMigratedVolume(vmiCopy, volume.PersistentVolumeClaim.ClaimName, volume); err != nil {
				return err
			}
		} else if volume.DataVolume != nil {
			if err := c.appendMigratedVolume(vmiCopy, volume.DataVolume.Name, volume); err != nil {
				return err
			}
		}
	}
	patch, err := patch.New(
		patch.WithTest("/status/migratedVolumes", vmi.Status.MigratedVolumes),
		patch.WithReplace("/status/migratedVolumes", vmiCopy.Status.MigratedVolumes),
	).GeneratePayload()
	if err != nil {
		return err
	}
	vmi, err = c.clientset.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}
