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
	"k8s.io/apimachinery/pkg/types"
)

// BackupMode is the const type for the backup possible modes
type BackupMode string

const (
	// PushMode defines backup which pushes the backup output
	// to a provided PVC - this is the default behavior
	PushMode BackupMode = "Push"
	// PullMode defines backup which exposes a pull endpoint
	// containing the backup disks and metadata
	PullMode BackupMode = "Pull"
)

// BackupVolumeInfo contains information about a volume included in a backup
type BackupVolumeInfo struct {
	// VolumeName is the volume name from VMI spec
	VolumeName string `json:"volumeName"`
	// DiskTarget is the disk target device name at backup time
	DiskTarget string `json:"diskTarget"`
	// DataEndpoint is the URL of the endpoint for read for pull mode
	// Deprecated: still populated for backward compatibility
	// Use Links.Internal or Links.External for structured endpoint access
	// with explicit internal/external distinction
	DataEndpoint string `json:"dataEndpoint,omitempty"`
	// MapEndpoint is the URL of the endpoint for map for pull mode
	// Deprecated: still populated for backward compatibility
	// Use Links.Internal or Links.External for structured endpoint access
	// with explicit internal/external distinction
	MapEndpoint string `json:"mapEndpoint,omitempty"`
}

// BackupLinks contains internal and external links for accessing backup data in pull mode
// Internal links use in-cluster service DNS (ClusterIP), while external links
// use a Route or Ingress hostname via virt-exportproxy
type BackupLinks struct {
	// Internal contains endpoints reachable from within the cluster
	// +optional
	Internal *BackupLink `json:"internal,omitempty"`
	// External contains endpoints reachable from outside the cluster
	// +optional
	External *BackupLink `json:"external,omitempty"`
}

// BackupLink contains a CA certificate and per-volume endpoints for one network path
type BackupLink struct {
	// Cert is the CA certificate bundle for TLS verification
	Cert string `json:"cert"`
	// Volumes lists the data and map endpoints for each backed-up volume
	// +listType=map
	// +listMapKey=volumeName
	// +optional
	Volumes []BackupVolumeLink `json:"volumes,omitempty"`
}

// BackupVolumeLink contains the data and map endpoint URLs for a single volume
type BackupVolumeLink struct {
	// VolumeName identifies the volume these endpoints belong to
	VolumeName string `json:"volumeName"`
	// DataEndpoint is the URL for reading backup data
	DataEndpoint string `json:"dataEndpoint"`
	// MapEndpoint is the URL for reading the changed block map
	MapEndpoint string `json:"mapEndpoint"`
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
	Start  BackupCmd = "Start"
	Abort  BackupCmd = "Abort"
	Export BackupCmd = "Export"
)

// BackupOptions are options used to configure virtual machine backup job
type BackupOptions struct {
	BackupName       string       `json:"backupName,omitempty"`
	Cmd              BackupCmd    `json:"cmd,omitempty"`
	Mode             BackupMode   `json:"mode,omitempty"`
	BackupStartTime  *metav1.Time `json:"backupStartTime,omitempty"`
	Incremental      *string      `json:"incremental,omitempty"`
	TargetPath       *string      `json:"targetPath,omitempty"`
	SkipQuiesce      bool         `json:"skipQuiesce,omitempty"`
	ExportServerAddr *string      `json:"exportServerAddr,omitempty"`
	ExportServerName *string      `json:"exportServerName,omitempty"`
	BackupKey        *string      `json:"backupKey,omitempty"`
	BackupCert       *string      `json:"backupCert,omitempty"`
	CACert           *string      `json:"caCert,omitempty"`
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
// +kubebuilder:validation:XValidation:rule="has(self.pvcName) && self.pvcName != \"\"",message="pvcName is required"
// +kubebuilder:validation:XValidation:rule="!has(self.mode) || self.mode != 'Pull' || (has(self.tokenSecretRef) && self.tokenSecretRef != \"\")",message="tokenSecretRef is required when mode is Pull"
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
	// +kubebuilder:validation:Enum=Push;Pull
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
	// +optional
	// TokenSecretRef is the name of the secret that
	// will be used to pull the backup from an associated endpoint
	TokenSecretRef string `json:"tokenSecretRef,omitempty"`
	// +optional
	// TtlDuration limits the lifetime of a pull mode backup and its export
	// If this field is set, after this duration has passed from counting from CreationTimestamp,
	// the backup is eligible to be automatically considered as complete.
	// If this field is omitted, a reasonable default is applied.
	// +optional
	TTLDuration *metav1.Duration `json:"ttlDuration,omitempty"`
}

// QuiesceStatus represents the outcome of a filesystem quiesce operation
type QuiesceStatus string

const (
	// QuiesceSucceeded indicates the filesystem was successfully quiesced
	QuiesceSucceeded QuiesceStatus = "Succeeded"
	// QuiesceFailed indicates the filesystem quiesce failed
	QuiesceFailed QuiesceStatus = "Failed"
	// QuiesceSkipped indicates the filesystem quiesce was skipped
	QuiesceSkipped QuiesceStatus = "Skipped"
)

// VirtualMachineBackupStatus is the status for a VirtualMachineBackup resource
type VirtualMachineBackupStatus struct {
	// +optional
	// Type indicates if the backup was full or incremental
	Type BackupType `json:"type,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +optional
	// CheckpointName the name of the checkpoint created for the current backup
	CheckpointName *string `json:"checkpointName,omitempty"`
	// +optional
	// EndpointCert is the raw CACert that is to be used when connecting
	// to an exported backup endpoint in pull mode.
	// Deprecated: still populated for backward compatibility
	// Use Links.Internal.Cert or Links.External.Cert for the corresponding CA certificate
	EndpointCert *string `json:"endpointCert,omitempty"`
	// +optional
	// +listType=atomic
	// IncludedVolumes lists the volumes that were included in the backup
	IncludedVolumes []BackupVolumeInfo `json:"includedVolumes,omitempty"`
	// +optional
	// ExportUID tracks the UID of the associated VMExport for pull-mode backups
	// used to detect VMExport recreation and re-initiate the export handshake
	ExportUID *types.UID `json:"exportUID,omitempty"`
	// +optional
	// Links contains structured internal and external endpoints for pull-mode backups
	// Each link includes a CA certificate and per-volume data/map endpoint URLs
	// Clients that need to distinguish between in-cluster and external access paths
	// should use this field instead of the flat EndpointCert/IncludedVolumes fields
	Links *BackupLinks `json:"links,omitempty"`
}

// ConditionType is the const type for Conditions
type ConditionType string

const (
	// ConditionComplete indicates the backup completed successfully
	ConditionComplete ConditionType = "Complete"

	// ConditionFailed indicates the backup has encountered a terminal failure
	ConditionFailed ConditionType = "Failed"

	// ConditionProgressing indicates the backup is in progress
	ConditionProgressing ConditionType = "Progressing"

	// ConditionQuiesced indicates whether the VM filesystem was quiesced (frozen) during backup
	ConditionQuiesced ConditionType = "Quiesced"
)

// Reason constants for ConditionProgressing, ConditionComplete, ConditionFailed, and ConditionQuiesced
const (
	ReasonInitializing         = "Initializing"
	ReasonInitiated            = "Initiated"
	ReasonPreparingExport      = "PreparingExport"
	ReasonExportInitiated      = "ExportInitiated"
	ReasonExportReady          = "ExportReady"
	ReasonAborting             = "Aborting"
	ReasonCompleted            = "Completed"
	ReasonCompletedWithWarning = "CompletedWithWarning"
	ReasonFailed               = "Failed"
	ReasonQuiesceSucceeded     = "QuiesceSucceeded"
	ReasonQuiesceFailed        = "QuiesceFailed"
	ReasonQuiesceSkipped       = "QuiesceSkipped"
	ReasonSourceLost           = "SourceLost"
	ReasonSourceUnhealthy      = "SourceUnhealthy"
	ReasonDeletedDuringInit    = "DeletedDuringInit"
)
