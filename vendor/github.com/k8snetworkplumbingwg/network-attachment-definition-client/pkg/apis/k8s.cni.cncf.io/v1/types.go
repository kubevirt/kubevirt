package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +resourceName=network-attachment-definitions

type NetworkAttachmentDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NetworkAttachmentDefinitionSpec `json:"spec"`
}

type NetworkAttachmentDefinitionSpec struct {
	Config string `json:"config"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type NetworkAttachmentDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []NetworkAttachmentDefinition `json:"items"`
}

// DNS contains values interesting for DNS resolvers
// +k8s:deepcopy-gen=false
type DNS struct {
	Nameservers []string `json:"nameservers,omitempty"`
	Domain      string   `json:"domain,omitempty"`
	Search      []string `json:"search,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// NetworkStatus is for network status annotation for pod
// +k8s:deepcopy-gen=false
type NetworkStatus struct {
	Name      string   `json:"name"`
	Interface string   `json:"interface,omitempty"`
	IPs       []string `json:"ips,omitempty"`
	Mac       string   `json:"mac,omitempty"`
	Default   bool     `json:"default,omitempty"`
	DNS       DNS      `json:"dns,omitempty"`
}

// PortMapEntry for CNI PortMapEntry
// +k8s:deepcopy-gen=false
type PortMapEntry struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
	HostIP        string `json:"hostIP,omitempty"`
}

// BandwidthEntry for CNI BandwidthEntry
// +k8s:deepcopy-gen=false
type BandwidthEntry struct {
	IngressRate  int `json:"ingressRate"`
	IngressBurst int `json:"ingressBurst"`

	EgressRate  int `json:"egressRate"`
	EgressBurst int `json:"egressBurst"`
}

// NetworkSelectionElement represents one element of the JSON format
// Network Attachment Selection Annotation as described in section 4.1.2
// of the CRD specification.
// +k8s:deepcopy-gen=false
type NetworkSelectionElement struct {
	// Name contains the name of the Network object this element selects
	Name string `json:"name"`
	// Namespace contains the optional namespace that the network referenced
	// by Name exists in
	Namespace string `json:"namespace,omitempty"`
	// IPRequest contains an optional requested IP address for this network
	// attachment
	IPRequest string `json:"ips,omitempty"`
	// MacRequest contains an optional requested MAC address for this
	// network attachment
	MacRequest string `json:"mac,omitempty"`
	// InterfaceRequest contains an optional requested name for the
	// network interface this attachment will create in the container
	InterfaceRequest string `json:"interface,omitempty"`
	// PortMappingsRequest contains an optional requested port mapping
	// for the network
	PortMappingsRequest []*PortMapEntry `json:"portMappings,omitempty"`
	// BandwidthRequest contains an optional requested bandwidth for
	// the network
	BandwidthRequest *BandwidthEntry `json:"bandwidth,omitempty"`
	// CNIArgs contains additional CNI arguments for the network interface
	CNIArgs *map[string]interface{} `json:"cni-args"`
	// GatewayRequest contains default route IP address for the pod
	GatewayRequest []net.IP `json:"default-route,omitempty"`
}

const (
	// Pod annotation for network-attachment-definition
	NetworkAttachmentAnnot = "k8s.v1.cni.cncf.io/networks"
	// Pod annotation for network status
	NetworkStatusAnnot = "k8s.v1.cni.cncf.io/network-status"
	// Old Pod annotation for network status (which is used before but it will be obsolated)
	OldNetworkStatusAnnot = "k8s.v1.cni.cncf.io/networks-status"
)

// NoK8sNetworkError indicates error, no network in kubernetes
// +k8s:deepcopy-gen=false
type NoK8sNetworkError struct {
	Message string
}

func (e *NoK8sNetworkError) Error() string { return string(e.Message) }
