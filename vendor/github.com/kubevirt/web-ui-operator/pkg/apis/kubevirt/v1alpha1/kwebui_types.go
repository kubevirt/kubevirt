package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KWebUISpec defines the desired state of KWebUI
type KWebUISpec struct {
	Version  string `json:"version,omitempty"`	// the desired kubevirt-web-ui version to be installed, conforms the docker tag. Example: 1.4.0-4

	RegistryUrl string `json:"registry_url,omitempty"`	// the registry for docker image (ie.: quay.io)
	RegistryNamespace string `json:"registry_namespace,omitempty"` // i.e. "kubevirt"
	ImagePullPolicy string `json:"image_pull_policy,omitempty"` // Always, IfNotPresent, Never

	OpenshiftMasterDefaultSubdomain string `json:"openshift_master_default_subdomain,omitempty"` // optional - workaround if openshift-console is not deployed, otherwise auto-discovered from its ConfigMap
	PublicMasterHostname string `json:"public_master_hostname,omitempty"` // optional - workaround if openshift-console is not deployed, otherwise auto-discovered from its ConfigMap

	Branding string `json:"branding,omitempty"` // optional, default: okdvirt

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// KWebUIStatus defines the observed state of KWebUI
type KWebUIStatus struct {
	Phase string `json:"phase,omitempty"` // one of the Phase* constants
	Message string `json:"message,omitempty"` // extra human-readable message
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KWebUI is the Schema for the kwebuis API
// +k8s:openapi-gen=true
type KWebUI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KWebUISpec   `json:"spec,omitempty"`
	Status KWebUIStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KWebUIList contains a list of KWebUI
type KWebUIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KWebUI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KWebUI{}, &KWebUIList{})
}
