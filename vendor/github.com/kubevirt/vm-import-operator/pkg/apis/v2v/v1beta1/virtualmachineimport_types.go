package v1beta1

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualMachineImportSpec defines the desired state of VirtualMachineImport
// +k8s:openapi-gen=true
type VirtualMachineImportSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	ProviderCredentialsSecret ObjectIdentifier `json:"providerCredentialsSecret"`
	// +optional
	ResourceMapping *ObjectIdentifier              `json:"resourceMapping,omitempty"`
	Source          VirtualMachineImportSourceSpec `json:"source"`

	// +optional
	TargetVMName *string `json:"targetVmName,omitempty"`

	// +optional
	StartVM *bool `json:"startVm,omitempty"`
}

// VirtualMachineImportSourceSpec defines the source provider and the internal mapping resources
// +k8s:openapi-gen=true
type VirtualMachineImportSourceSpec struct {
	// +optional
	Ovirt *VirtualMachineImportOvirtSourceSpec `json:"ovirt,omitempty"`
	// +optional
	Vmware *VirtualMachineImportVmwareSourceSpec `json:"vmware,omitempty"`
}

// VirtualMachineImportOvirtSourceSpec defines the mapping resources and the VM identity for oVirt source provider
// +k8s:openapi-gen=true
type VirtualMachineImportOvirtSourceSpec struct {
	VM VirtualMachineImportOvirtSourceVMSpec `json:"vm"`

	// +optional
	Mappings *OvirtMappings `json:"mappings,omitempty"`
}

// VirtualMachineImportVmwareSourceSpec defines the mapping resources and the VM identity for vmware source provider
// +k8s:openapi-gen=true
type VirtualMachineImportVmwareSourceSpec struct {
	VM VirtualMachineImportVmwareSourceVMSpec `json:"vm"`

	// +optional
	Mappings *VmwareMappings `json:"mappings,omitempty"`
}

// ObjectIdentifier defines how a resource should be identified on kubevirt
// +k8s:openapi-gen=true
type ObjectIdentifier struct {
	Name string `json:"name"`

	// +optional
	Namespace *string `json:"namespace,omitempty"`
}

// VirtualMachineImportOvirtSourceVMSpec defines how to identify the VM in oVirt
// +k8s:openapi-gen=true
type VirtualMachineImportOvirtSourceVMSpec struct {
	// +optional
	ID *string `json:"id,omitempty"`

	// +optional
	Name *string `json:"name,omitempty"`

	// +optional
	Cluster *VirtualMachineImportOvirtSourceVMClusterSpec `json:"cluster,omitempty"`
}

// VirtualMachineImportVmwareSourceVMSpec defines how to identify the VM in vCenter
// +k8s:openapi-gen=true
type VirtualMachineImportVmwareSourceVMSpec struct {
	// UUID of virtual machine
	// +optional
	ID *string `json:"id,omitempty"`

	// +optional
	Name *string `json:"name,omitempty"`
}

// VirtualMachineImportOvirtSourceVMClusterSpec defines the source cluster's identity of the VM in oVirt
// +k8s:openapi-gen=true
// +optional
type VirtualMachineImportOvirtSourceVMClusterSpec struct {
	// +optional
	ID *string `json:"id,omitempty"`

	// +optional
	Name *string `json:"name,omitempty"`
}

// VirtualMachineImportStatus defines the observed state of VirtualMachineImport
// +k8s:openapi-gen=true
type VirtualMachineImportStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// +optional
	TargetVMName string `json:"targetVmName"`

	// +optional
	Conditions []VirtualMachineImportCondition `json:"conditions"`

	// +optional
	DataVolumes []DataVolumeItem `json:"dataVolumes,omitempty"`
}

// VirtualMachineImportConditionType defines the condition of VM import
// +k8s:openapi-gen=true
type VirtualMachineImportConditionType string

// These are valid conditions of of VM import.
const (
	// Succeeded represents status of the VM import process being completed successfully
	Succeeded VirtualMachineImportConditionType = "Succeeded"

	// Valid represents the status of the validation of the mapping rules and eligibility of source VM for import
	Valid VirtualMachineImportConditionType = "Valid"

	// MappingRulesVerified represents the status of the VM import mapping rules checking
	MappingRulesVerified VirtualMachineImportConditionType = "MappingRulesVerified"

	// Processing represents the status of the VM import process while in progress
	Processing VirtualMachineImportConditionType = "Processing"
)

// SucceededConditionReason defines the reasons for the Succeeded condition of VM import
// +k8s:openapi-gen=true
type SucceededConditionReason string

// These are valid reasons for the Succeeded conditions of VM import.
const (
	// ValidationFailed represents a failure to validate the eligibility of the VM for import
	ValidationFailed SucceededConditionReason = "ValidationFailed"

	// VMTemplateMatchingFailed represents a failure to match VM template
	VMTemplateMatchingFailed SucceededConditionReason = "VMTemplateMatchingFailed"

	// VMCreationFailed represents a failure to create the VM entity
	VMCreationFailed SucceededConditionReason = "VMCreationFailed"

	// DataVolumeCreationFailed represents a failure to create data volumes based on source VM disks
	DataVolumeCreationFailed SucceededConditionReason = "DataVolumeCreationFailed"

	// VirtualMachineReady represents the completion of the vm import
	VirtualMachineReady SucceededConditionReason = "VirtualMachineReady"

	// VirtualMachineRunning represents the completion of the vm import and vm in running state
	VirtualMachineRunning SucceededConditionReason = "VirtualMachineRunning"
)

// ValidConditionReason defines the reasons for the Valid condition of VM import
// +k8s:openapi-gen=true
type ValidConditionReason string

// These are valid reasons for the Valid conditions of VM import.
const (
	// ValidationCompleted represents the completion of the vm import resource validating
	ValidationCompleted ValidConditionReason = "ValidationCompleted"

	// SecretNotFound represents the nonexistence of the provider's secret
	SecretNotFound ValidConditionReason = "SecretNotFound"

	// ResourceMappingNotFound represents the nonexistence of the mapping resource
	ResourceMappingNotFound ValidConditionReason = "ResourceMappingNotFound"

	// UninitializedProvider represents a failure to initialize the provider
	UninitializedProvider ValidConditionReason = "UninitializedProvider"

	// UnreachableProvider represents a failure to connect to the provider
	UnreachableProvider ValidConditionReason = "UnreachableProvider"

	// SourceVmNotFound represents the nonexistence of the source VM
	SourceVMNotFound ValidConditionReason = "SourceVMNotFound"

	// IncompleteMappingRules represents the inability to prepare the mapping rules
	IncompleteMappingRules ValidConditionReason = "IncompleteMappingRules"

	// ValidationReportedWarnings represents the existence of warnings related to resource mapping validation
	ValidationReportedWarnings ValidConditionReason = "ValidationReportedWarnings"
)

// MappingRulesVerifiedReason defines the reasons for the MappingRulesVerified condition of VM import
// +k8s:openapi-gen=true
type MappingRulesVerifiedReason string

// These are valid reasons for the MappingRulesVerified conditions of VM import.
const (
	// MappingRulesVerificationCompleted represents the completion of the mapping rules checking without warnings or errors
	MappingRulesVerificationCompleted MappingRulesVerifiedReason = "MappingRulesVerificationCompleted"

	// MappingRulesVerificationFailed represents the violation of the mapping rules
	MappingRulesVerificationFailed MappingRulesVerifiedReason = "MappingRulesVerificationFailed"

	// MappingRulesVerificationReportedWarnings represents the existence of warnings as a result of checking the mapping rules
	MappingRulesVerificationReportedWarnings MappingRulesVerifiedReason = "MappingRulesVerificationReportedWarnings"
)

// ProcessingConditionReason defines the reasons for the Processing condition of VM import
// +k8s:openapi-gen=true
type ProcessingConditionReason string

// These are valid reasons for the Processing conditions of VM import.
const (
	// VMTemplateMatching represents the VM template matching process
	VMTemplateMatching ProcessingConditionReason = "VMTemplateMatching"

	// CreatingTargetVM represents the creation of the VM spec
	CreatingTargetVM ProcessingConditionReason = "CreatingTargetVM"

	// CopyingDisks represents the creation of data volumes based on source VM disks
	CopyingDisks ProcessingConditionReason = "CopyingDisks"

	// ProcessingCompleted represents the successful import processing
	ProcessingCompleted ProcessingConditionReason = "ProcessingCompleted"

	// ProcessingFailed represents failed import processing
	ProcessingFailed ProcessingConditionReason = "ProcessingFailed"

	// Pending represents pending for PVC to bound
	Pending ProcessingConditionReason = "Pending"
)

// VirtualMachineImportCondition defines the observed state of VirtualMachineImport conditions
// +k8s:openapi-gen=true
type VirtualMachineImportCondition struct {
	// Type of virtual machine import condition
	Type VirtualMachineImportConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown
	Status k8sv1.ConditionStatus `json:"status"`

	// A brief CamelCase string that describes why the VM import process is in current condition status
	// +optional
	Reason *string `json:"reason,omitempty"`

	// A human-readable message indicating details about last transition
	// +optional
	Message *string `json:"message,omitempty"`

	// The last time we got an update on a given condition
	// +optional
	LastHeartbeatTime *metav1.Time `json:"lastHeartbeatTime,omitempty"`

	// The last time the condition transit from one status to another
	// +optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
}

func (cond VirtualMachineImportCondition) String() string {
	return fmt.Sprintf(
		"VirtualMachineImportCondition{type: %v, status: %v, reason: %v, message: %v}",
		cond.Type, cond.Status, *cond.Reason, *cond.Message,
	)
}

// DataVolumeItem defines the details of a data volume created by the VM import process
// +k8s:openapi-gen=true
type DataVolumeItem struct {
	Name string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineImport is the Schema for the virtualmachineimports API
// +k8s:openapi-gen=true
// +genclient
// +kubebuilder:subresource:status
type VirtualMachineImport struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineImportSpec   `json:"spec,omitempty"`
	Status VirtualMachineImportStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachineImportList contains a list of VirtualMachineImport
type VirtualMachineImportList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachineImport `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachineImport{}, &VirtualMachineImportList{})
}
