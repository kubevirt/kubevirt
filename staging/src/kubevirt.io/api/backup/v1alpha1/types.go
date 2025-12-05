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

// BackupType is the const type for the backup possible types
type BackupType string

const (
	// Full defines full backup, all the data is in the backup
	Full BackupType = "Full"
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
	PushPath        *string      `json:"pushPath,omitempty"`
	SkipQuiesce     bool         `json:"skipQuiesce,omitempty"`
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
type VirtualMachineBackupSpec struct {
	// +optional
	// Source specifies the VM to backup
	// If not provided, a reference to a VirtualMachineBackupTracker must be specified instead
	Source *corev1.TypedLocalObjectReference `json:"source,omitempty"`
	// +optional
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
