package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SriovNetworkNodeStateSpec defines the desired state of SriovNetworkNodeState
// +k8s:openapi-gen=true
type SriovNetworkNodeStateSpec struct {
	DpConfigVersion string     `json:"dpConfigVersion,omitempty"`
	Interfaces      Interfaces `json:"interfaces,omitempty"`
}

type Interface struct {
	PciAddress string    `json:"pciAddress"`
	NumVfs     int       `json:"numVfs,omitempty"`
	Mtu        int       `json:"mtu,omitempty"`
	Name       string    `json:"name,omitempty"`
	LinkType   string    `json:"linkType,omitempty"`
	VfGroups   []VfGroup `json:"vfGroups,omitempty"`
}

type VfGroup struct {
	ResourceName string `json:"resourceName,omitempty"`
	DeviceType   string `json:"deviceType,omitempty"`
	VfRange      string `json:"vfRange,omitempty"`
	PolicyName   string `json:"policyName,omitempty"`
}

type Interfaces []Interface

type InterfaceExt struct {
	Name       string            `json:"name,omitempty"`
	Mac        string            `json:"mac,omitempty"`
	Driver     string            `json:"driver,omitempty"`
	PciAddress string            `json:"pciAddress"`
	Vendor     string            `json:"vendor,omitempty"`
	DeviceID   string            `json:"deviceID,omitempty"`
	Mtu        int               `json:"mtu,omitempty"`
	NumVfs     int               `json:"numVfs,omitempty"`
	LinkSpeed  string            `json:"linkSpeed,omitempty"`
	LinkType   string            `json:"linkType,omitempty"`
	TotalVfs   int               `json:"totalvfs,omitempty"`
	VFs        []VirtualFunction `json:"Vfs,omitempty"`
}
type InterfaceExts []InterfaceExt

type VirtualFunction struct {
	Name       string `json:"name,omitempty"`
	Mac        string `json:"mac,omitempty"`
	Assigned   string `json:"assigned,omitempty"`
	Driver     string `json:"driver,omitempty"`
	PciAddress string `json:"pciAddress"`
	Vendor     string `json:"vendor,omitempty"`
	DeviceID   string `json:"deviceID,omitempty"`
	Vlan       int    `json:"Vlan,omitempty"`
	Mtu        int    `json:"mtu,omitempty"`
	VfID       int    `json:"vfID"`
}

// SriovNetworkNodeStateStatus defines the observed state of SriovNetworkNodeState
// +k8s:openapi-gen=true
type SriovNetworkNodeStateStatus struct {
	Interfaces    InterfaceExts `json:"interfaces,omitempty"`
	SyncStatus    string        `json:"syncStatus,omitempty"`
	LastSyncError string        `json:"lastSyncError,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// SriovNetworkNodeState is the Schema for the sriovnetworknodestates API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=sriovnetworknodestates,scope=Namespaced
type SriovNetworkNodeState struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SriovNetworkNodeStateSpec   `json:"spec,omitempty"`
	Status SriovNetworkNodeStateStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SriovNetworkNodeStateList contains a list of SriovNetworkNodeState
type SriovNetworkNodeStateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SriovNetworkNodeState `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SriovNetworkNodeState{}, &SriovNetworkNodeStateList{})
}
