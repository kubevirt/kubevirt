package v1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigStatus defines the observed state of Config
// +k8s:openapi-gen=true
type ConfigStatus struct {
	OperatorVersion string                   `json:"operatorVersion,omitempty"`
	ObservedVersion string                   `json:"observedVersion,omitempty"`
	TargetVersion   string                   `json:"targetVersion,omitempty"`
	Conditions      []conditionsv1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`
	Containers      []Container              `json:"containers,omitempty"`
}

type Container struct {
	Namespace  string `json:"namespace"`
	ParentKind string `json:"parentKind"`
	ParentName string `json:"parentName"`
	Name       string `json:"name"`
	Image      string `json:"image"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubevirtCommonTemplatesBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VersionSpec  `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubevirtNodeLabellerBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ComponentSpec `json:"spec,omitempty"`
	Status ConfigStatus  `json:"status,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubevirtTemplateValidator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TemplateValidatorSpec `json:"spec,omitempty"`
	Status ConfigStatus          `json:"status,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubevirtMetricsAggregation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VersionSpec  `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

// minimal spec: only the version
type VersionSpec struct {
	Version string `json:"version,omitempty"`
}

type ComponentSpec struct {
	Version string `json:"version,omitempty"`
}

type TemplateValidatorSpec struct {
	Version                   string `json:"version,omitempty"`
	TemplateValidatorReplicas int    `json:"templateValidatorReplicas,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubevirtCommonTemplatesBundleList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubevirtCommonTemplatesBundle `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubevirtNodeLabellerBundleList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubevirtNodeLabellerBundle `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubevirtTemplateValidatorList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubevirtTemplateValidator `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type KubevirtMetricsAggregationList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubevirtMetricsAggregation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KubevirtCommonTemplatesBundle{}, &KubevirtCommonTemplatesBundleList{})
	SchemeBuilder.Register(&KubevirtNodeLabellerBundle{}, &KubevirtNodeLabellerBundleList{})
	SchemeBuilder.Register(&KubevirtTemplateValidator{}, &KubevirtTemplateValidatorList{})
	SchemeBuilder.Register(&KubevirtMetricsAggregation{}, &KubevirtMetricsAggregationList{})
}
