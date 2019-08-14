package v1alpha1

import (
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	CatalogSourceCRDAPIVersion = operators.GroupName + "/" + GroupVersion
	CatalogSourceKind          = "CatalogSource"
)

type CatalogSourceSpec struct {
	SourceType string   `json:"sourceType"`
	ConfigMap  string   `json:"configMap,omitempty"`
	Secrets    []string `json:"secrets,omitempty"`

	// Metadata
	DisplayName string `json:"displayName,omitempty"`
	Description string `json:"description,omitempty"`
	Publisher   string `json:"publisher,omitempty"`
	Icon        Icon   `json:"icon,omitempty"`
}

type CatalogSourceStatus struct {
	ConfigMapResource *ConfigMapResourceReference `json:"configMapReference,omitempty"`
	LastSync          metav1.Time                 `json:"lastSync,omitempty"`
}
type ConfigMapResourceReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`

	UID             types.UID `json:"uid,omitempty"`
	ResourceVersion string    `json:"resourceVersion,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient
type CatalogSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   CatalogSourceSpec   `json:"spec"`
	Status CatalogSourceStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CatalogSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []CatalogSource `json:"items"`
}
