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

// VirtualMachineMigrationResourceQuota defines resources that should be reserved for a VMI migration
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
type VirtualMachineMigrationResourceQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineMigrationResourceQuotaSpec   `json:"spec" valid:"required"`
	Status VirtualMachineMigrationResourceQuotaStatus `json:"status,omitempty"`
}

type VirtualMachineMigrationResourceQuotaSpec struct {
	// AdditionalMigrationResources specifies the extra resources needed during virtual machine migration,
	//above the resourceQuota limit. This field helps ensure a successful migration by allowing you to define
	//the additional resources required, such as CPU, memory, and storage.
	// +optional
	AdditionalMigrationResources corev1.ResourceList `json:"additionalMigrationResources,omitempty" protobuf:"bytes,1,rep,name=hard,casttype=ResourceList,castkey=ResourceName"`
}

type VirtualMachineMigrationResourceQuotaPhase string

const (
	PhaseUnset                     VirtualMachineMigrationResourceQuotaPhase = ""
	WaitingForTargetPodToBeCreated VirtualMachineMigrationResourceQuotaPhase = "Processing"
	RestoringQuota                 VirtualMachineMigrationResourceQuotaPhase = "SafeToModify"
)

type VirtualMachineMigrationResourceQuotaStatus struct {
	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	Phase VirtualMachineMigrationResourceQuotaPhase `json:"phase,omitempty"`

	// +optional
	// +nullable
	NamespaceLocked *bool `json:"namespaceLocked,omitempty"`

	// +optional
	MigrationsToBlockingResourceQuotas map[string][]string `json:"migrationsToBlockingResourceQuotas,omitempty"`

	// +optional
	// +listType=atomic
	OriginalBlockingResourceQuotas []corev1.ResourceQuota `json:"originalBlockingResourceQuotas,omitempty"`

	// +optional
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`
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

// VirtualMachineMigrationResourceQuota List is a list of VirtualMachineMigrationResourceQuotas
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineMigrationResourceQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=atomic
	Items []VirtualMachineMigrationResourceQuota `json:"items"`
}
