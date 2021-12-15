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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package v1alpha1

import (
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	virtv1 "kubevirt.io/api/core/v1"
)

const (
	VirtualMachinePoolKind = "VirtualMachinePool"
)

// VirtualMachinePool resource contains a VirtualMachine configuration
// that can be used to replicate multiple VirtualMachine resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +genclient
type VirtualMachinePool struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachinePoolSpec   `json:"spec" valid:"required"`
	Status VirtualMachinePoolStatus `json:"status,omitempty"`
}

//
// +k8s:openapi-gen=true
type VirtualMachineTemplateSpec struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	ObjectMeta metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineSpec contains the VirtualMachine specification.
	Spec virtv1.VirtualMachineSpec `json:"spec,omitempty" valid:"required"`
}

//
// +k8s:openapi-gen=true
type VirtualMachinePoolConditionType string

const (
	// VirtualMachinePoolReplicaFailure is added in a pool when one of its vms
	// fails to be created.
	VirtualMachinePoolReplicaFailure VirtualMachinePoolConditionType = "ReplicaFailure"

	// VirtualMachinePoolReplicaPaused is added in a pool when the pool got paused by the controller.
	// After this condition was added, it is safe to remove or add vms by hand and adjust the replica count manually
	VirtualMachinePoolReplicaPaused VirtualMachinePoolConditionType = "ReplicaPaused"
)

//
// +k8s:openapi-gen=true
type VirtualMachinePoolCondition struct {
	Type   VirtualMachinePoolConditionType `json:"type"`
	Status k8sv1.ConditionStatus           `json:"status"`
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	Reason             string      `json:"reason,omitempty"`
	Message            string      `json:"message,omitempty"`
}

//
// +k8s:openapi-gen=true
type VirtualMachinePoolStatus struct {
	Replicas int32 `json:"replicas,omitempty" optional:"true"`

	// +listType=atomic
	Conditions []VirtualMachinePoolCondition `json:"conditions,omitempty" optional:"true"`

	// Canonical form of the label selector for HPA which consumes it through the scale subresource.
	LabelSelector string `json:"labelSelector,omitempty"`
}

//
// +k8s:openapi-gen=true
type VirtualMachinePoolSpec struct {
	// Number of desired pods. This is a pointer to distinguish between explicit
	// zero and not specified. Defaults to 1.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Label selector for pods. Existing Poolss whose pods are
	// selected by this will be the ones affected by this deployment.
	Selector *metav1.LabelSelector `json:"selector" valid:"required"`

	// Template describes the VM that will be created.
	VirtualMachineTemplate *VirtualMachineTemplateSpec `json:"virtualMachineTemplate" valid:"required"`

	// Indicates that the pool is paused.
	// +optional
	Paused bool `json:"paused,omitempty" protobuf:"varint,7,opt,name=paused"`
}

// VirtualMachinePoolList is a list of VirtualMachinePool resources.
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
type VirtualMachinePoolList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachinePool `json:"items"`
}
