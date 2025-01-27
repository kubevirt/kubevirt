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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// VirtualMachineClone is a CRD that clones one VM into another.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
type VirtualMachineClone struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineCloneSpec   `json:"spec" valid:"required"`
	Status VirtualMachineCloneStatus `json:"status,omitempty"`
}

type VirtualMachineCloneSpec struct {
	// Source is the object that would be cloned. Currently supported source types are:
	// VirtualMachine of kubevirt.io API group,
	// VirtualMachineSnapshot of snapshot.kubevirt.io API group
	Source *corev1.TypedLocalObjectReference `json:"source"`

	// Target is the outcome of the cloning process.
	// Currently supported source types are:
	// - VirtualMachine of kubevirt.io API group
	// - Empty (nil).
	// If the target is not provided, the target type would default to VirtualMachine and a random
	// name would be generated for the target. The target's name can be viewed by
	// inspecting status "TargetName" field below.
	// +optional
	Target *corev1.TypedLocalObjectReference `json:"target,omitempty"`

	// Example use: "!some/key*".
	// For a detailed description, please refer to https://kubevirt.io/user-guide/operations/clone_api/#label-annotation-filters.
	// +optional
	// +listType=atomic
	AnnotationFilters []string `json:"annotationFilters,omitempty"`
	// Example use: "!some/key*".
	// For a detailed description, please refer to https://kubevirt.io/user-guide/operations/clone_api/#label-annotation-filters.
	// +optional
	// +listType=atomic
	LabelFilters []string `json:"labelFilters,omitempty"`

	// NewMacAddresses manually sets that target interfaces' mac addresses. The key is the interface name and the
	// value is the new mac address. If this field is not specified, a new MAC address will
	// be generated automatically, as for any interface that is not included in this map.
	// +optional
	NewMacAddresses map[string]string `json:"newMacAddresses,omitempty"`
	// NewSMBiosSerial manually sets that target's SMbios serial. If this field is not specified, a new serial will
	// be generated automatically.
	// +optional
	NewSMBiosSerial *string `json:"newSMBiosSerial,omitempty"`
}

type VirtualMachineClonePhase string

const (
	PhaseUnset         VirtualMachineClonePhase = ""
	SnapshotInProgress VirtualMachineClonePhase = "SnapshotInProgress"
	CreatingTargetVM   VirtualMachineClonePhase = "CreatingTargetVM"
	RestoreInProgress  VirtualMachineClonePhase = "RestoreInProgress"
	Succeeded          VirtualMachineClonePhase = "Succeeded"
	Failed             VirtualMachineClonePhase = "Failed"
	Unknown            VirtualMachineClonePhase = "Unknown"
)

type VirtualMachineCloneStatus struct {
	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	Phase VirtualMachineClonePhase `json:"phase,omitempty"`

	// +optional
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`

	// +optional
	// +nullable
	SnapshotName *string `json:"snapshotName,omitempty"`

	// +optional
	// +nullable
	RestoreName *string `json:"restoreName,omitempty"`

	// +optional
	// +nullable
	TargetName *string `json:"targetName,omitempty"`
}

// ConditionType is the const type for Conditions
type ConditionType string

const (
	// ConditionReady is the "ready" condition type
	ConditionReady ConditionType = "Ready"

	// ConditionProgressing is the "progressing" condition type
	ConditionProgressing ConditionType = "Progressing"
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

// VirtualMachineCloneList is a list of MigrationPolicy
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineCloneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=atomic
	Items []VirtualMachineClone `json:"items"`
}
