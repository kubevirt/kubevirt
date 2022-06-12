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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

const DefaultFailureDeadline = 5 * time.Minute

// VirtualMachineSnapshot defines the operation of snapshotting a VM
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineSnapshotSpec `json:"spec"`

	// +optional
	Status *VirtualMachineSnapshotStatus `json:"status,omitempty"`
}

// DeletionPolicy defines that to do with VirtualMachineSnapshot
// when VirtualMachineSnapshot is deleted
type DeletionPolicy string

const (
	// VirtualMachineSnapshotContentDelete causes the
	// VirtualMachineSnapshotContent to be deleted
	VirtualMachineSnapshotContentDelete DeletionPolicy = "Delete"

	// VirtualMachineSnapshotContentRetain causes the
	// VirtualMachineSnapshotContent to stay around
	VirtualMachineSnapshotContentRetain DeletionPolicy = "Retain"
)

// VirtualMachineSnapshotSpec is the spec for a VirtualMachineSnapshot resource
type VirtualMachineSnapshotSpec struct {
	Source corev1.TypedLocalObjectReference `json:"source"`

	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// This time represents the number of seconds we permit the vm snapshot
	// to take. In case we pass this deadline we mark this snapshot
	// as failed.
	// Defaults to DefaultFailureDeadline - 5min
	// +optional
	FailureDeadline *metav1.Duration `json:"failureDeadline,omitempty"`
}

// Indication is a way to indicate the state of the vm when taking the snapshot
type Indication string

const (
	VMSnapshotOnlineSnapshotIndication Indication = "Online"
	VMSnapshotNoGuestAgentIndication   Indication = "NoGuestAgent"
	VMSnapshotGuestAgentIndication     Indication = "GuestAgent"
)

// VirtualMachineSnapshotPhase is the current phase of the VirtualMachineSnapshot
type VirtualMachineSnapshotPhase string

const (
	PhaseUnset VirtualMachineSnapshotPhase = ""
	InProgress VirtualMachineSnapshotPhase = "InProgress"
	Succeeded  VirtualMachineSnapshotPhase = "Succeeded"
	Failed     VirtualMachineSnapshotPhase = "Failed"
	Unknown    VirtualMachineSnapshotPhase = "Unknown"
)

// VirtualMachineSnapshotStatus is the status for a VirtualMachineSnapshot resource
type VirtualMachineSnapshotStatus struct {
	// +optional
	SourceUID *types.UID `json:"sourceUID,omitempty"`

	// +optional
	VirtualMachineSnapshotContentName *string `json:"virtualMachineSnapshotContentName,omitempty"`

	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	Phase VirtualMachineSnapshotPhase `json:"phase,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *Error `json:"error,omitempty"`

	// +optional
	Conditions []Condition `json:"conditions,omitempty"`

	// +optional
	// +listType=set
	Indications []Indication `json:"indications,omitempty"`
}

// Error is the last error encountered during the snapshot/restore
type Error struct {
	// +optional
	Time *metav1.Time `json:"time,omitempty"`

	// +optional
	Message *string `json:"message,omitempty"`
}

// ConditionType is the const type for Conditions
type ConditionType string

const (
	// ConditionReady is the "ready" condition type
	ConditionReady ConditionType = "Ready"

	// ConditionProgressing is the "progressing" condition type
	ConditionProgressing ConditionType = "Progressing"

	// ConditionFailure is the "failure" condition type
	ConditionFailure ConditionType = "Failure"
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

// VirtualMachineSnapshotList is a list of VirtualMachineSnapshot resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualMachineSnapshot `json:"items"`
}

// VirtualMachineSnapshotContent contains the snapshot data
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshotContent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineSnapshotContentSpec `json:"spec"`

	// +optional
	Status *VirtualMachineSnapshotContentStatus `json:"status,omitempty"`
}

// VirtualMachineSnapshotContentSpec is the spec for a VirtualMachineSnapshotContent resource
type VirtualMachineSnapshotContentSpec struct {
	VirtualMachineSnapshotName *string `json:"virtualMachineSnapshotName,omitempty"`

	Source SourceSpec `json:"source"`

	// +optional
	VolumeBackups []VolumeBackup `json:"volumeBackups,omitempty"`
}

type VirtualMachine struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineSpec contains the VirtualMachine specification.
	Spec v1.VirtualMachineSpec `json:"spec,omitempty" valid:"required"`
	// Status holds the current state of the controller and brief information
	// about its associated VirtualMachineInstance
	Status v1.VirtualMachineStatus `json:"status,omitempty"`
}

// SourceSpec contains the appropriate spec for the resource being snapshotted
type SourceSpec struct {
	// +optional
	VirtualMachine *VirtualMachine `json:"virtualMachine,omitempty"`
}

type PersistentVolumeClaim struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired characteristics of a volume requested by a pod author.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	Spec corev1.PersistentVolumeClaimSpec `json:"spec,omitempty"`
}

// VolumeBackup contains the data neeed to restore a PVC
type VolumeBackup struct {
	VolumeName string `json:"volumeName"`

	PersistentVolumeClaim PersistentVolumeClaim `json:"persistentVolumeClaim"`

	// +optional
	VolumeSnapshotName *string `json:"volumeSnapshotName,omitempty"`
}

// VirtualMachineSnapshotContentStatus is the status for a VirtualMachineSnapshotStatus resource
type VirtualMachineSnapshotContentStatus struct {
	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *Error `json:"error,omitempty"`

	// +optional
	VolumeSnapshotStatus []VolumeSnapshotStatus `json:"volumeSnapshotStatus,omitempty"`
}

// VirtualMachineSnapshotContentList is a list of VirtualMachineSnapshot resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshotContentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualMachineSnapshotContent `json:"items"`
}

// VolumeSnapshotStatus is the status of a VolumeSnapshot
type VolumeSnapshotStatus struct {
	VolumeSnapshotName string `json:"volumeSnapshotName"`

	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *Error `json:"error,omitempty"`
}

// VirtualMachineRestore defines the operation of restoring a VM
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineRestoreSpec `json:"spec"`

	// +optional
	Status *VirtualMachineRestoreStatus `json:"status,omitempty"`
}

// VirtualMachineRestoreSpec is the spec for a VirtualMachineRestoreresource
type VirtualMachineRestoreSpec struct {
	// initially only VirtualMachine type supported
	Target corev1.TypedLocalObjectReference `json:"target"`

	VirtualMachineSnapshotName string `json:"virtualMachineSnapshotName"`

	// If the target for the restore does not exist, it will be created. Patches holds JSON patches that would be
	// applied to the target manifest before it's created. Patches should fit the target's Kind.
	//
	// Example for a patch: {"op": "replace", "path": "/metadata/name", "value": "new-vm-name"}
	//
	// +optional
	// +listType=atomic
	Patches []string `json:"patches,omitempty"`
}

// VirtualMachineRestoreStatus is the spec for a VirtualMachineRestoreresource
type VirtualMachineRestoreStatus struct {
	// +optional
	Restores []VolumeRestore `json:"restores,omitempty"`

	// +optional
	RestoreTime *metav1.Time `json:"restoreTime,omitempty"`

	// +optional
	DeletedDataVolumes []string `json:"deletedDataVolumes,omitempty"`

	// +optional
	Complete *bool `json:"complete,omitempty"`

	// +optional
	Conditions []Condition `json:"conditions,omitempty"`
}

// VolumeRestore contains the data neeed to restore a PVC
type VolumeRestore struct {
	VolumeName string `json:"volumeName"`

	PersistentVolumeClaimName string `json:"persistentVolumeClaim"`

	VolumeSnapshotName string `json:"volumeSnapshotName"`

	// +optional
	DataVolumeName *string `json:"dataVolumeName,omitempty"`
}

// VirtualMachineRestoreList is a list of VirtualMachineRestore resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualMachineRestore `json:"items"`
}
