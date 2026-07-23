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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

// MigrationPolicy holds migration policy (i.e. configurations) to apply to a VM or group of VMs
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
// +genclient:nonNamespaced
type MigrationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MigrationPolicySpec `json:"spec" valid:"required"`
	// +nullable
	Status MigrationPolicyStatus `json:"status,omitempty"`
}

type MigrationPolicySpec struct {
	Selectors *Selectors `json:"selectors"`

	//+optional
	AllowAutoConverge *bool `json:"allowAutoConverge,omitempty"`
	//+optional
	BandwidthPerMigration *resource.Quantity `json:"bandwidthPerMigration,omitempty"`
	//+optional
	CompletionTimeoutPerGiB *int64 `json:"completionTimeoutPerGiB,omitempty"`
	//+optional
	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=2000000
	MaxDowntimeMs *uint64 `json:"maxDowntimeMs,omitempty"`
	//+optional
	AllowPostCopy *bool `json:"allowPostCopy,omitempty"`
	//+optional
	AllowWorkloadDisruption *bool `json:"allowWorkloadDisruption,omitempty"`
	// ExperimentalMigrationOptions is an alpha API. It is intended for experimental
	// purposes only and will be removed in the future.
	//+optional
	ExperimentalMigrationOptions *k6tv1.ExperimentalMigrationOptions `json:"experimental,omitempty"`
}

type LabelSelector map[string]string

type Selectors struct {
	//+optional
	NamespaceSelector LabelSelector `json:"namespaceSelector,omitempty"`
	//+optional
	VirtualMachineInstanceSelector LabelSelector `json:"virtualMachineInstanceSelector,omitempty"`
}

type MigrationPolicyStatus struct {
}

// MigrationPolicyList is a list of MigrationPolicy
//
// +k8s:openapi-gen=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MigrationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	// +listType=atomic
	Items []MigrationPolicy `json:"items"`
}
