/*
Copyright 2021 The CDI Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// this has to be here otherwise informer-gen doesn't recognize it
// see https://github.com/kubernetes/code-generator/issues/59
// +genclient:nonNamespaced

// Deprecated for removal in v1.
//
// ObjectTransfer is the cluster scoped object transfer resource
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:shortName=ot;ots,scope=Cluster
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="The phase of the ObjectTransfer"
// +kubebuilder:subresource:status
type ObjectTransfer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ObjectTransferSpec `json:"spec"`

	// +optional
	Status ObjectTransferStatus `json:"status"`
}

// TransferSource is the source of a ObjectTransfer
type TransferSource struct {
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	Kind string `json:"kind"`

	Namespace string `json:"namespace"`

	Name string `json:"name"`

	RequiredAnnotations map[string]string `json:"requiredAnnotations,omitempty"`
}

// TransferTarget is the target of an ObjectTransfer
type TransferTarget struct {
	Namespace *string `json:"namespace,omitempty"`

	Name *string `json:"name,omitempty"`
}

// ObjectTransferSpec specifies the source/target of the transfer
type ObjectTransferSpec struct {
	Source TransferSource `json:"source"`

	Target TransferTarget `json:"target"`

	ParentName *string `json:"parentName,omitempty"`
}

// ObjectTransferPhase is the phase of the ObjectTransfer
type ObjectTransferPhase string

const (
	// ObjectTransferEmpty is the empty transfer phase
	ObjectTransferEmpty ObjectTransferPhase = ""

	// ObjectTransferPending is the pending transfer phase
	ObjectTransferPending ObjectTransferPhase = "Pending"

	// ObjectTransferRunning is the running transfer phase
	ObjectTransferRunning ObjectTransferPhase = "Running"

	// ObjectTransferComplete is the complete transfer phase
	ObjectTransferComplete ObjectTransferPhase = "Complete"

	// ObjectTransferError is the (terminal) error transfer phase
	ObjectTransferError ObjectTransferPhase = "Error"
)

// ObjectTransferConditionType is the type of ObjectTransferCondition
type ObjectTransferConditionType string

const (
	// ObjectTransferConditionComplete is the "complete" condition
	ObjectTransferConditionComplete ObjectTransferConditionType = "Complete"
)

// ObjectTransferCondition contains condition data
type ObjectTransferCondition struct {
	Type               ObjectTransferConditionType `json:"type"`
	Status             corev1.ConditionStatus      `json:"status"`
	LastTransitionTime metav1.Time                 `json:"lastTransitionTime,omitempty"`
	LastHeartbeatTime  metav1.Time                 `json:"lastHeartbeatTime,omitempty"`
	Reason             string                      `json:"reason,omitempty"`
	Message            string                      `json:"message,omitempty"`
}

// ObjectTransferStatus is the status of the ObjectTransfer
type ObjectTransferStatus struct {
	// Data is a place for intermediary state.  Or anything really.
	Data map[string]string `json:"data,omitempty"`

	// Phase is the current phase of the transfer
	Phase ObjectTransferPhase `json:"phase,omitempty"`

	Conditions []ObjectTransferCondition `json:"conditions,omitempty"`
}

// ObjectTransferList provides the needed parameters to do request a list of ObjectTransfers from the system
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type ObjectTransferList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items provides a list of ObjectTransfers
	Items []ObjectTransfer `json:"items"`
}
