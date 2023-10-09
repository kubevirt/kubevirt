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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	VolumeMigrationFinalizer string = "foregroundDeleteVolumeMigration"
	VolumeMigrationLabel     string = "kubevirt.io/volume-migration"
)

// VolumeMigration defines the operation of moving the storage to another
// storage backend.
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VolumeMigration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              VolumeMigrationSpec `json:"spec" valid:"required"`
	// +optional
	Status VolumeMigrationStatus `json:"status,omitempty"`
}

// VolumeMigrationList is a list of VolumeMigration resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VolumeMigrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// +listType=atomic
	Items []VolumeMigration `json:"items"`
}

// SourceReclaimPolicy describes how the source PVC will be treated after the storage migration completes.
// The policies follows the same behavior as the RetainPolicy for PVs:
//
//	https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming
type SourceReclaimPolicy string

const (
	SourceReclaimPolicyDelete SourceReclaimPolicy = "Delete"
	SourceReclaimPolicyRetain SourceReclaimPolicy = "Retain"
)

type MigratedVolume struct {
	SourceClaim         string              `json:"sourceClaim" valid:"required"`
	DestinationClaim    string              `json:"destinationClaim" valid:"required"`
	SourceReclaimPolicy SourceReclaimPolicy `json:"sourceReclaimPolicy,omitempty"`
}

// VolumeMigrationSpec is the spec for a VolumeMigration resource
type VolumeMigrationSpec struct {
	MigratedVolume []MigratedVolume `json:"migratedVolume,omitempty"`
}

const (
	ReasonRejectHotplugVolumes           = "Hotplug volumes aren't supported to be migrated yet"
	ReasonRejectShareableVolumes         = "Shareable disks aren't supported to be migrated"
	ReasonRejectFilesystemVolumes        = "Filesystem volumes aren't supported to be migrated"
	ReasonRejectLUNVolumes               = "LUN disks aren't supported to be migrated yet"
	ReasonRejectedMultipleVMIs           = "Volumes need to belong to the same VMI"
	ReasonRejectedPending                = "Volumes need to belong to a running VMI"
	ReasonRejectedMultipleVMIsAndPending = "Volumes need to belong to the same running VMI"
)

type VolumeMigrationPhase string

const (
	VolumeMigrationPhasePending    VolumeMigrationPhase = "Pending"
	VolumeMigrationPhaseScheduling VolumeMigrationPhase = "Scheduling"
	VolumeMigrationPhaseRunning    VolumeMigrationPhase = "Running"
	VolumeMigrationPhaseSucceeded  VolumeMigrationPhase = "Succeeded"
	VolumeMigrationPhaseFailed     VolumeMigrationPhase = "Failed"
	VolumeMigrationPhaseUnknown    VolumeMigrationPhase = "Unknown"
)

type MigratedVolumeValidation string

const (
	MigratedVolumeValidationValid    MigratedVolumeValidation = "Valid"
	MigratedVolumeValidationPending  MigratedVolumeValidation = "Pending"
	MigratedVolumeValidationRejected MigratedVolumeValidation = "Rejected"
)

type VolumeMigrationPhaseTransitionTimestamp struct {
	Phase                    VolumeMigrationPhase `json:"phase,omitempty"`
	PhaseTransitionTimestamp metav1.Time          `json:"phaseTransitionTimestamp,omitempty"`
}

type VolumeMigrationState struct {
	MigratedVolume `json:",inline"`
	Validation     MigratedVolumeValidation `json:"validation,omitempty"`
	Reason         *string                  `json:"reason,omitempty"`
}

type VolumeMigrationStatus struct {
	VolumeMigrationStates       []VolumeMigrationState                    `json:"volumeMigrationStates,omitempty"`
	VirtualMachineInstanceName  *string                                   `json:"virtualMachineInstanceName,omitempty"`
	VirtualMachineMigrationName *string                                   `json:"virtualMachineMigrationName,omitempty"`
	StartTimestamp              *metav1.Time                              `json:"startTimestamp,omitempty"`
	EndTimestamp                *metav1.Time                              `json:"endTimestamp,omitempty"`
	Phase                       VolumeMigrationPhase                      `json:"phase,omitempty"`
	PhaseTransitionTimestamps   []VolumeMigrationPhaseTransitionTimestamp `json:"phaseTransitionTimestamps,omitempty"`
}

func (sm *VolumeMigration) GetVirtualMachiheInstanceMigrationName(vmiName string) string {
	return fmt.Sprintf("%s-%s", sm.Name, vmiName)
}
