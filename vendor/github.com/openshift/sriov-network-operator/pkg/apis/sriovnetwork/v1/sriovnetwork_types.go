package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SriovNetworkSpec defines the desired state of SriovNetwork
// +k8s:openapi-gen=true
type SriovNetworkSpec struct {
	// Namespace of the NetworkAttachmentDefinition custom resource
	NetworkNamespace string `json:"networkNamespace,omitempty"`
	// SRIOV Network device plugin endpoint resource name
	ResourceName string `json:"resourceName"`
	//Capabilities to be configured for this network.
	//Capabilities supported: (mac|ips), e.g. '{"mac": true}'
	Capabilities string `json:"capabilities,omitempty"`
	//IPAM configuration to be used for this network.
	IPAM string `json:"ipam,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=4096
	// VLAN ID to assign for the VF. Defaults to 0.
	Vlan int `json:"vlan,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=7
	// VLAN QoS ID to assign for the VF. Defaults to 0.
	VlanQoS int `json:"vlanQoS,omitempty"`
	// VF spoof check, (on|off)
	// +kubebuilder:validation:Enum={"on","off"}
	SpoofChk string `json:"spoofChk,omitempty"`
	// VF trust mode (on|off)
	// +kubebuilder:validation:Enum={"on","off"}
	Trust string `json:"trust,omitempty"`
	// VF link state (enable|disable|auto)
	// +kubebuilder:validation:Enum={"auto","enable","disable"}
	LinkState string `json:"linkState,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// Minimum tx rate, in Mbps, for the VF. Defaults to 0 (no rate limiting). min_tx_rate should be <= max_tx_rate.
	MinTxRate *int `json:"minTxRate,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// Maximum tx rate, in Mbps, for the VF. Defaults to 0 (no rate limiting)
	MaxTxRate *int `json:"maxTxRate,omitempty"`
}

// SriovNetworkStatus defines the observed state of SriovNetwork
// +k8s:openapi-gen=true
type SriovNetworkStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SriovNetwork is the Schema for the sriovnetworks API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=sriovnetworks,scope=Namespaced
type SriovNetwork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkSpec   `json:"spec,omitempty"`
	Status SriovNetworkStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SriovNetworkList contains a list of SriovNetwork
type SriovNetworkList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SriovNetwork `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SriovNetwork{}, &SriovNetworkList{})
}
