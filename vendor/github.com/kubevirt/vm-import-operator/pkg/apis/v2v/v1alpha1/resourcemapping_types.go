package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ResourceMappingSpec defines the desired state of ResourceMapping
// +k8s:openapi-gen=true
type ResourceMappingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html

	// +optional
	OvirtMappings *OvirtMappings `json:"ovirt,omitempty"`
}

// OvirtMappings defines the mappings of ovirt resources to kubevirt
// +k8s:openapi-gen=true
type OvirtMappings struct {
	// NetworkMappings defines the mapping of vnic profile to network attachment definition
	// When providing source network by name, the format is 'network name/vnic profile name'.
	// When providing source network by ID, the ID represents the vnic profile ID.
	// A logical network from ovirt can be mapped to multiple network attachment definitions
	// on kubevirt by using vnic profile to network attachment definition mapping.
	// +optional
	NetworkMappings *[]ResourceMappingItem `json:"networkMappings,omitempty"`

	// StorageMappings defines the mapping of storage domains to storage classes
	// +optional
	StorageMappings *[]ResourceMappingItem `json:"storageMappings,omitempty"`

	// DiskMappings defines the mapping of disks to storage classes
	// DiskMappings.Source.ID represents the disk ID on ovirt (as opposed to disk-attachment ID)
	// DiskMappings.Source.Name represents the disk alias on ovirt
	// DiskMappings is respected only when provided in context of a single VM import within VirtualMachineImport
	// +optional
	DiskMappings *[]ResourceMappingItem `json:"diskMappings,omitempty"`
}

// Source defines how to identify a resource on the provider, either by ID or by name
// +k8s:openapi-gen=true
type Source struct {
	// +optional
	Name *string `json:"name,omitempty"`

	// +optional
	ID *string `json:"id,omitempty"`
}

// ResourceMappingItem defines the mapping of a single resource from the provider to kubevirt
// +k8s:openapi-gen=true
type ResourceMappingItem struct {
	Source Source           `json:"source"`
	Target ObjectIdentifier `json:"target"`

	// +optional
	Type *string `json:"type,omitempty"`
}

// ResourceMappingStatus defines the observed state of ResourceMapping
// +k8s:openapi-gen=true
type ResourceMappingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceMapping is the Schema for the ResourceMappings API
// +k8s:openapi-gen=true
// +genclient
// +kubebuilder:subresource:status
type ResourceMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceMappingSpec   `json:"spec,omitempty"`
	Status ResourceMappingStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ResourceMappingList contains a list of ResourceMapping
type ResourceMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ResourceMapping{}, &ResourceMappingList{})
}
