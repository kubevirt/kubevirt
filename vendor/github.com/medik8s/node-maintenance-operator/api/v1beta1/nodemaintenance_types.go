/*
Copyright 2021.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// NodeMaintenanceFinalizer is a finalizer for a NodeMaintenance CR deletion
	NodeMaintenanceFinalizer string = "foregroundDeleteNodeMaintenance"
)

// MaintenancePhase contains the phase of maintenance
type MaintenancePhase string

const (
	// MaintenanceRunning - maintenance has started its proccessing
	MaintenanceRunning MaintenancePhase = "Running"
	// MaintenanceSucceeded - maintenance has finished succesfuly, cordoned the node and evicted all pods (that could be evicted)
	MaintenanceSucceeded MaintenancePhase = "Succeeded"
	// MaintenanceFailed - maintenance has failed
	MaintenanceFailed MaintenancePhase = "Failed"
)

// NodeMaintenanceSpec defines the desired state of NodeMaintenance
type NodeMaintenanceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Node name to apply maintanance on/off
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	NodeName string `json:"nodeName"`
	// Reason for maintanance
	//+operator-sdk:csv:customresourcedefinitions:type=spec
	Reason string `json:"reason,omitempty"`
}

// NodeMaintenanceStatus defines the observed state of NodeMaintenance
type NodeMaintenanceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase is the represtation of the maintenance progress (Running,Succeeded,Failed)
	//+operator-sdk:csv:customresourcedefinitions:type=status
	Phase MaintenancePhase `json:"phase,omitempty"`
	// LastError represents the latest error if any in the latest reconciliation
	//+operator-sdk:csv:customresourcedefinitions:type=status
	LastError string `json:"lastError,omitempty"`
	// PendingPods is a list of pending pods for eviction
	//+operator-sdk:csv:customresourcedefinitions:type=status
	PendingPods []string `json:"pendingPods,omitempty"`
	// TotalPods is the total number of all pods on the node from the start
	//+operator-sdk:csv:customresourcedefinitions:type=status
	TotalPods int `json:"totalpods,omitempty"`
	// EvictionPods is the total number of pods up for eviction from the start
	//+operator-sdk:csv:customresourcedefinitions:type=status
	EvictionPods int `json:"evictionPods,omitempty"`
	// Consecutive number of errors upon obtaining a lease
	//+operator-sdk:csv:customresourcedefinitions:type=status
	ErrorOnLeaseCount int `json:"errorOnLeaseCount,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster,shortName=nm

// NodeMaintenance is the Schema for the nodemaintenances API
// +operator-sdk:csv:customresourcedefinitions:resources={{"NodeMaintenance","v1beta1","nodemaintenances"}}
type NodeMaintenance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeMaintenanceSpec   `json:"spec,omitempty"`
	Status NodeMaintenanceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodeMaintenanceList contains a list of NodeMaintenance
type NodeMaintenanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeMaintenance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeMaintenance{}, &NodeMaintenanceList{})
}
