package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapNodeUnhealthyConditions contains the name of the unhealthy conditions config map
const ConfigMapNodeUnhealthyConditions = "node-unhealthy-conditions"

// RemediationStrategyType contains type of the remediation that we are wanting to use
type RemediationStrategyType string

const (
	// RemediationStrategyTypeReboot contains name of the reboot remediation strategy
	RemediationStrategyTypeReboot = "Reboot"
	// RemediationStrategyTypeReCreate contains name of the re-create remediation strategy
	RemediationStrategyTypeReCreate = "ReCreate"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineHealthCheck is the Schema for the machinehealthchecks API
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mhc;mhcs
// +k8s:openapi-gen=true
type MachineHealthCheck struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of machine health check policy
	Spec MachineHealthCheckSpec `json:"spec,omitempty"`

	// Most recently observed status of MachineHealthCheck resource
	Status MachineHealthCheckStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineHealthCheckList contains a list of MachineHealthCheck
type MachineHealthCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineHealthCheck `json:"items"`
}

// MachineHealthCheckSpec defines the desired state of MachineHealthCheck
type MachineHealthCheckSpec struct {
	// RemediationStrategy to use in case of problem detection
	// default is machine deletion
	RemediationStrategy *RemediationStrategyType `json:"remediationStrategy,omitempty"`
	// Label selector to match machines whose health will be exercised
	Selector metav1.LabelSelector `json:"selector"`
}

// MachineHealthCheckStatus defines the observed state of MachineHealthCheck
type MachineHealthCheckStatus struct {
	// TODO(alberto)
}
