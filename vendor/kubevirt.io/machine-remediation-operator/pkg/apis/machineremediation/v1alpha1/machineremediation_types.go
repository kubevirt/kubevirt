package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RemediationType contains type of the remediation
type RemediationType string

const (
	// RemediationTypeReboot contains reboot type of the remediation
	RemediationTypeReboot RemediationType = "Reboot"
	// RemediationTypeRecreate contains re-create type of the remediation
	RemediationTypeRecreate RemediationType = "Re-Create"
)

// RemediationState contains state of the remediation
type RemediationState string

const (
	// RemediationStateStarted contains remediation state when the machine remediation object was created
	RemediationStateStarted RemediationState = "Started"
	// RemediationStatePowerOff contains remediation state when the host power offed by the controller
	RemediationStatePowerOff RemediationState = "PowerOff"
	// RemediationStatePowerOn contains remediation state when the host power oned again by the controller
	RemediationStatePowerOn RemediationState = "PowerOn"
	// RemediationStateSucceeded contains remediation state when the operation succeeded
	RemediationStateSucceeded RemediationState = "Succeeded"
	// RemediationStateFailed contains remediation state when the operation failed
	RemediationStateFailed RemediationState = "Failed"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineRemediation is the schema for the MachineRemediation API
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mr;mrs
// +k8s:openapi-gen=true
type MachineRemediation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of MachineRemediation
	Spec MachineRemediationSpec `json:"spec,omitempty"`

	// Most recently observed status of MachineRemediation resource
	Status MachineRemediationStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MachineRemediationList contains a list of MachineRemediation
type MachineRemediationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineRemediation `json:"items"`
}

// MachineRemediationSpec defines the spec of MachineRemediation
type MachineRemediationSpec struct {
	// Type contains the type of the remediation
	Type RemediationType `json:"type,omitempty" valid:"required"`
	// MachineName contains the name of machine that should be remediate
	MachineName string `json:"machineName,omitempty" valid:"required"`
}

// MachineRemediationStatus defines the observed status of MachineRemediation
type MachineRemediationStatus struct {
	State     RemediationState `json:"state,omitempty"`
	Reason    string           `json:"reason,omitempty"`
	StartTime *metav1.Time     `json:"startTime,omitempty"`
	EndTime   *metav1.Time     `json:"endTime,omitempty"`
}
