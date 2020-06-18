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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
)

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
	// VirtualMachineSnapshotContentDelete is the default and causes the
	// VirtualMachineSnapshotContent to be deleted
	VirtualMachineSnapshotContentDelete DeletionPolicy = "Delete"

	// VirtualMachineSnapshotContentRetain is the default and causes the
	// VirtualMachineSnapshotContent to stay around
	VirtualMachineSnapshotContentRetain DeletionPolicy = "Retain"
)

// VirtualMachineSnapshotSpec is the spec for a VirtualMachineSnapshot resource
type VirtualMachineSnapshotSpec struct {
	Source corev1.TypedLocalObjectReference `json:"source"`

	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`
}

// VirtualMachineSnapshotStatus is the status for a VirtualMachineSnapshot resource
type VirtualMachineSnapshotStatus struct {
	// +optional
	VirtualMachineSnapshotContentName *string `json:"virtualMachineSnapshotContentName,omitempty"`

	// +optional
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *VirtualMachineSnapshotError `json:"error,omitempty"`

	// +optional
	Conditions []VirtualMachineSnapshotCondition `json:"conditions,omitempty"`
}

// VirtualMachineSnapshotError is the last error encountered while creating the snapshot
type VirtualMachineSnapshotError struct {
	// +optional
	Time *metav1.Time `json:"time,omitempty"`

	// +optional
	Message *string `json:"message,omitempty"`
}

// VirtualMachineSnapshotConditionType is the const type for VirtualMachineSnapshotConditions
type VirtualMachineSnapshotConditionType string

const (
	// VirtualMachineSnapshotConditionReady is the "ready" condition type
	VirtualMachineSnapshotConditionReady VirtualMachineSnapshotConditionType = "Ready"

	// VirtualMachineSnapshotConditionProgressing is the "progressing" condition type
	VirtualMachineSnapshotConditionProgressing VirtualMachineSnapshotConditionType = "Progressing"
)

// VirtualMachineSnapshotCondition defines snapshot conditions
type VirtualMachineSnapshotCondition struct {
	Type VirtualMachineSnapshotConditionType `json:"type"`

	Status corev1.ConditionStatus `json:"status"`

	// +optional
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`

	// +optional
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

// SourceSpec contains the appropriate spec for the resource being snapshotted
type SourceSpec struct {
	// +optional
	VirtualMachine *v1.VirtualMachine `json:"virtualMachine,omitempty"`
}

// VolumeBackup contains the data neeed to restore a PVC
type VolumeBackup struct {
	DiskName string `json:"diskName"`

	PersistentVolumeClaim corev1.PersistentVolumeClaim `json:"persistentVolumeClaim"`

	// +optional
	VolumeSnapshotName *string `json:"volumeSnapshotName,omitempty"`
}

// VirtualMachineSnapshotContentStatus is the status for a VirtualMachineSnapshotStatus resource
type VirtualMachineSnapshotContentStatus struct {
	// +optional
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *VirtualMachineSnapshotError `json:"error,omitempty"`

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
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *VirtualMachineSnapshotError `json:"error,omitempty"`
}
