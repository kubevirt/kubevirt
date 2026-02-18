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
 * Copyright 2025 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BackupMode is the const type for the backup possible modes
type BackupMode string

const (
	// PushMode defines backup which pushes the backup output
	// to a provided PVC - this is the default behavior
	PushMode BackupMode = "Push"
)

// BackupVolumeInfo contains information about a volume included in a backup
type BackupVolumeInfo struct {
	// VolumeName is the volume name from VMI spec
	VolumeName string `json:"volumeName"`
	// DiskTarget is the disk target device name at backup time
	DiskTarget string `json:"diskTarget"`
}

type BackupCheckpoint struct {
	Name         string       `json:"name,omitempty"`
	CreationTime *metav1.Time `json:"creationTime,omitempty"`
	// Volumes lists volumes and their disk targets at backup time
	// +optional
	// +listType=atomic
	Volumes []BackupVolumeInfo `json:"volumes,omitempty"`
}

// BackupType is the const type for the backup possible types
type BackupType string

const (
	// Full defines full backup, all the data is in the backup
	Full BackupType = "Full"
	// Incremental defines incremental backup, only changes from given checkpoint
	// are in the backup
	Incremental BackupType = "Incremental"
)

// BackupCmd is the const type for the backup possible commands
type BackupCmd string

const (
	Start BackupCmd = "Start"
)

// BackupOptions are options used to configure virtual machine backup job
type BackupOptions struct {
	BackupName      string       `json:"backupName,omitempty"`
	Cmd             BackupCmd    `json:"cmd,omitempty"`
	Mode            BackupMode   `json:"mode,omitempty"`
	BackupStartTime *metav1.Time `json:"backupStartTime,omitempty"`
	Incremental     *string      `json:"incremental,omitempty"`
	PushPath        *string      `json:"pushPath,omitempty"`
	SkipQuiesce     bool         `json:"skipQuiesce,omitempty"`
}

// VirtualMachineBackupTracker defines the way to track the latest checkpoint of
// a backup solution for a vm
// +k8s:openapi-gen=true
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineBackupTracker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineBackupTrackerSpec `json:"spec"`

	// +optional
	Status *VirtualMachineBackupTrackerStatus `json:"status,omitempty"`
}

// VirtualMachineBackupTrackerSpec is the spec for a VirtualMachineBackupTracker resource
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec is immutable after creation"
type VirtualMachineBackupTrackerSpec struct {
	// Source specifies the VM that this backupTracker is associated with
	// +kubebuilder:validation:XValidation:rule="has(self.apiGroup) && self.apiGroup == 'kubevirt.io'",message="apiGroup must be kubevirt.io"
	// +kubebuilder:validation:XValidation:rule="self.kind == 'VirtualMachine'",message="kind must be VirtualMachine"
	// +kubebuilder:validation:XValidation:rule="self.name != ''",message="name is required"
	Source corev1.TypedLocalObjectReference `json:"source"`
}

type VirtualMachineBackupTrackerStatus struct {
	// +optional
	// LatestCheckpoint is the metadata of the checkpoint of
	// the latest performed backup
	LatestCheckpoint *BackupCheckpoint `json:"latestCheckpoint,omitempty"`

	// +optional
	// CheckpointRedefinitionRequired is set to true by virt-handler when the VM
	// restarts and has a checkpoint that needs to be redefined in libvirt.
	// virt-controller will process this flag, attempt redefinition, and clear it.
	CheckpointRedefinitionRequired *bool `json:"checkpointRedefinitionRequired,omitempty"`
}

// VirtualMachineBackupTrackerList is a list of VirtualMachineBackupTracker resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineBackupTrackerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	// +listType=atomic
	Items []VirtualMachineBackupTracker `json:"items"`
}

// VirtualMachineBackup defines the operation of backing up a VM
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineBackupSpec `json:"spec"`

	// +optional
	Status *VirtualMachineBackupStatus `json:"status,omitempty"`
}

// VirtualMachineBackupList is a list of VirtualMachineBackup resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// +listType=atomic
	Items []VirtualMachineBackup `json:"items"`
}

// VirtualMachineBackupSpec is the spec for a VirtualMachineBackup resource
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec is immutable after creation"
// +kubebuilder:validation:XValidation:rule="(has(self.mode) && self.mode != 'Push') || (has(self.pvcName) && self.pvcName != \"\")",message="pvcName must be provided when mode is unset or Push"
type VirtualMachineBackupSpec struct {
	// Source specifies the backup source - either a VirtualMachine or a VirtualMachineBackupTracker.
	// When Kind is VirtualMachine: performs a backup of the specified VM.
	// When Kind is VirtualMachineBackupTracker: uses the tracker to get the source VM
	// and the base checkpoint for incremental backup. The tracker will be updated
	// with the new checkpoint after backup completion.
	// +kubebuilder:validation:XValidation:rule="has(self.apiGroup)",message="apiGroup is required"
	// +kubebuilder:validation:XValidation:rule="!has(self.apiGroup) || self.apiGroup == 'kubevirt.io' || self.apiGroup == 'backup.kubevirt.io'",message="apiGroup must be kubevirt.io or backup.kubevirt.io"
	// +kubebuilder:validation:XValidation:rule="!has(self.apiGroup) || (self.apiGroup == 'kubevirt.io' && self.kind == 'VirtualMachine') || (self.apiGroup == 'backup.kubevirt.io' && self.kind == 'VirtualMachineBackupTracker')",message="kind must be VirtualMachine for kubevirt.io or VirtualMachineBackupTracker for backup.kubevirt.io"
	// +kubebuilder:validation:XValidation:rule="self.name != ''",message="name is required"
	Source corev1.TypedLocalObjectReference `json:"source"`
	// +optional
	// +kubebuilder:validation:Enum=Push
	// Mode specifies the way the backup output will be recieved
	Mode *BackupMode `json:"mode,omitempty"`
	// +optional
	// PvcName required in push mode. Specifies the name of the PVC
	// where the backup output will be stored
	PvcName *string `json:"pvcName,omitempty"`
	// +optional
	// SkipQuiesce indicates whether the VM's filesystem shoule not be quiesced before the backup
	SkipQuiesce bool `json:"skipQuiesce,omitempty"`
	// +optional
	// ForceFullBackup indicates that a full backup is desired
	ForceFullBackup bool `json:"forceFullBackup,omitempty"`
}

// VirtualMachineBackupStatus is the status for a VirtualMachineBackup resource
type VirtualMachineBackupStatus struct {
	// +optional
	// Type indicates if the backup was full or incremental
	Type BackupType `json:"type,omitempty"`
	// +optional
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`
	// +optional
	// CheckpointName the name of the checkpoint created for the current backup
	CheckpointName *string `json:"checkpointName,omitempty"`
	// +optional
	// +listType=atomic
	// IncludedVolumes lists the volumes that were included in the backup
	IncludedVolumes []BackupVolumeInfo `json:"includedVolumes,omitempty"`
}

// ConditionType is the const type for Conditions
type ConditionType string

const (
	// ConditionDone indicates the backup was completed
	ConditionDone ConditionType = "Done"

	// ConditionProgressing indicates the backup is in progress
	ConditionProgressing ConditionType = "Progressing"

	// ConditionInitializing indicates the backup is initializing
	ConditionInitializing ConditionType = "Initializing"

	// ConditionDeleting indicates the backup is deleteing
	ConditionDeleting ConditionType = "Deleting"
)

// Condition defines conditions
type Condition struct {
	Type ConditionType `json:"type"`

	Status corev1.ConditionStatus `json:"status"`

	// +optional
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`

	// +optional
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`
}
