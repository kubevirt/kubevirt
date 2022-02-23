package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/shared"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=networkaddonsconfigs,scope=Cluster
// +k8s:openapi-gen=true
// +kubebuilder:storageversion

// NetworkAddonsConfig is the Schema for the networkaddonsconfigs API
type NetworkAddonsConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   shared.NetworkAddonsConfigSpec   `json:"spec,omitempty"`
	Status shared.NetworkAddonsConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NetworkAddonsConfigList contains a list of NetworkAddonsConfig
type NetworkAddonsConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkAddonsConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkAddonsConfig{}, &NetworkAddonsConfigList{})
}
