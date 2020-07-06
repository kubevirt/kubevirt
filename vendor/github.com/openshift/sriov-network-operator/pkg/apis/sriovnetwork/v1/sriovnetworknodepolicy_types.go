package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SriovNetworkNodePolicySpec defines the desired state of SriovNetworkNodePolicy
// +k8s:openapi-gen=true
// +kubebuilder:pruning:PreserveUnknownFields
type SriovNetworkNodePolicySpec struct {
	// SRIOV Network device plugin endpoint resource name
	ResourceName string `json:"resourceName"`
	// NodeSelector selects the nodes to be configured
	NodeSelector map[string]string `json:"nodeSelector"`
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=99
	// Priority of the policy, higher priority policies can override lower ones.
	Priority int `json:"priority,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=9000
	// MTU of VF
	Mtu int `json:"mtu,omitempty"`
	// +kubebuilder:validation:Minimum=0
	// Number of VFs for each PF
	NumVfs int `json:"numVfs"`
	// NicSelector selects the NICs to be configured
	NicSelector SriovNetworkNicSelector `json:"nicSelector"`
	// +kubebuilder:validation:Enum=netdevice;vfio-pci
	// The driver type for configured VFs. Allowed value "netdevice", "vfio-pci". Defaults to netdevice.
	DeviceType string `json:"deviceType,omitempty"`
	// RDMA mode. Defaults to false.
	IsRdma bool `json:"isRdma,omitempty"`
	// +kubebuilder:validation:Enum=eth;ETH;ib;IB
	// NIC Link Type. Allowed value "eth", "ETH", "ib", and "IB".
	LinkType string `json:"linkType,omitempty"`
}

// +k8s:openapi-gen=false
type SriovNetworkNicSelector struct {
	// The vendor hex code of SR-IoV device. Allowed value "8086", "15b3".
	Vendor string `json:"vendor,omitempty"`
	// The device hex code of SR-IoV device. Allowed value "158b", "1015", "1017".
	DeviceID string `json:"deviceID,omitempty"`
	// PCI address of SR-IoV PF.
	RootDevices []string `json:"rootDevices,omitempty"`
	// Name of SR-IoV PF.
	PfNames []string `json:"pfNames,omitempty"`
}

// SriovNetworkNodePolicyStatus defines the observed state of SriovNetworkNodePolicy
// +k8s:openapi-gen=true
type SriovNetworkNodePolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SriovNetworkNodePolicy is the Schema for the sriovnetworknodepolicies API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=sriovnetworknodepolicies,scope=Namespaced
type SriovNetworkNodePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkNodePolicySpec   `json:"spec,omitempty"`
	Status SriovNetworkNodePolicyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SriovNetworkNodePolicyList contains a list of SriovNetworkNodePolicy
type SriovNetworkNodePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SriovNetworkNodePolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SriovNetworkNodePolicy{}, &SriovNetworkNodePolicyList{})
}
