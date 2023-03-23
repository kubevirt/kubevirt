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

type VirtualMachineMigrationResourceQuotaStatus struct {
	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// AdditionalMigrationResources specifies the extra resources needed during virtual machine migration,
	//above the resourceQuota limit. This field helps ensure a successful migration by allowing you to define
	//the additional resources required, such as CPU, memory, and storage.
	// +optional
	AdditionalMigrationResources corev1.ResourceList `json:"additionalMigrationResources,omitempty" protobuf:"bytes,1,rep,name=hard,casttype=ResourceList,castkey=ResourceName"`

	// +optional
	MigrationsToBlockingResourceQuotas map[string][]string `json:"migrationsToBlockingResourceQuotas,omitempty"`

	// +optional
	// +listType=atomic
	OriginalBlockingResourceQuotas []ResourceQuotaNameAndSpec `json:"originalBlockingResourceQuotas,omitempty"`
}

type ResourceQuotaNameAndSpec struct {
	// +optional
	Name string                   `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Spec corev1.ResourceQuotaSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
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
