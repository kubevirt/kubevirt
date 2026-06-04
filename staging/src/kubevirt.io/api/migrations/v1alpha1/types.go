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
	"encoding/json"

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
	Selectors                      *Selectors `json:"selectors"`
	k6tv1.VMMigrationConfiguration `json:",inline"`
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

// ApplyMigrationPolicy applies the destination configuration by merging all the non-nil fields
// in the destination config to the corresponding field in the source config.
func (m *MigrationPolicy) ApplyMigrationPolicy(dst *k6tv1.VMIMConfigurationOptions, src *k6tv1.VMMigrationConfiguration) {
	// For backward compatibility, if the policy specifies AllowPostCopy but not AllowWorkloadDisruption,
	// AllowWorkloadDisruption should follow AllowPostCopy.
	if src.AllowWorkloadDisruption == nil && src.AllowPostCopy != nil {
		v := *src.AllowPostCopy
		dst.AllowWorkloadDisruption = &v
	}

	srcJSON, err := json.Marshal(src)
	if err != nil {
		return
	}
	_ = json.Unmarshal(srcJSON, &dst.VMMigrationConfiguration)
}
