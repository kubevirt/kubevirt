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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"
)

// VirtualMachineMigrationResourceQuota defines resources that should be reserved for a VMI migration
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=vmmrq;vmmrqs,categories=all
// +kubebuilder:subresource:status
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

// this has to be here otherwise informer-gen doesn't recognize it
// see https://github.com/kubernetes/code-generator/issues/59
// +genclient:nonNamespaced

// MTQ is the MTQ Operator CRD
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=mtq;mtqs,scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
type MTQ struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MTQSpec `json:"spec"`
	// +optional
	Status MTQStatus `json:"status"`
}

// CertConfig contains the tunables for TLS certificates
type CertConfig struct {
	// The requested 'duration' (i.e. lifetime) of the Certificate.
	Duration *metav1.Duration `json:"duration,omitempty"`

	// The amount of time before the currently issued certificate's `notAfter`
	// time that we will begin to attempt to renew the certificate.
	RenewBefore *metav1.Duration `json:"renewBefore,omitempty"`
}

// MTQCertConfig has the CertConfigs for MTQ
type MTQCertConfig struct {
	// CA configuration
	// CA certs are kept in the CA bundle as long as they are valid
	CA *CertConfig `json:"ca,omitempty"`

	// Server configuration
	// Certs are rotated and discarded
	Server *CertConfig `json:"server,omitempty"`
}

// MTQSpec defines our specification for the MTQ installation
type MTQSpec struct {
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// PullPolicy describes a policy for if/when to pull a container image
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty" valid:"required"`
	// Rules on which nodes MTQ infrastructure pods will be scheduled
	Infra sdkapi.NodePlacement `json:"infra,omitempty"`
	// Restrict on which nodes MTQ workload pods will be scheduled
	Workloads sdkapi.NodePlacement `json:"workload,omitempty"`
	// certificate configuration
	CertConfig *MTQCertConfig `json:"certConfig,omitempty"`
	// PriorityClass of the MTQ control plane
	PriorityClass *MTQPriorityClass `json:"priorityClass,omitempty"`
}

// MTQPriorityClass defines the priority class of the MTQ control plane.
type MTQPriorityClass string

// MTQPhase is the current phase of the MTQ deployment
type MTQPhase string

// MTQStatus defines the status of the installation
type MTQStatus struct {
	sdkapi.Status `json:",inline"`
}

// MTQList provides the needed parameters to do request a list of MTQs from the system
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MTQList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items provides a list of MTQs
	Items []MTQ `json:"items"`
}
