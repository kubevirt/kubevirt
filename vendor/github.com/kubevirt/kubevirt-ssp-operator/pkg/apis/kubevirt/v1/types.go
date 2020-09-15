package v1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigStatus defines the observed state of Config
type ConfigStatus struct {
	// The version of the deployed operator
	OperatorVersion string `json:"operatorVersion,omitempty"`

	// The version of the deployed operands
	ObservedVersion string `json:"observedVersion,omitempty"`

	// The desired version of the deployed operands
	TargetVersion string `json:"targetVersion,omitempty"`

	// Reported states of the controller
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`

	// Containers used in the current deployment
	Containers []Container `json:"containers,omitempty"`
}

// Defines a container
type Container struct {
	// Container namespace
	Namespace string `json:"namespace"`

	// Parent kind
	ParentKind string `json:"parentKind"`

	// Parent image
	ParentName string `json:"parentName"`

	// Container name
	Name string `json:"name"`

	// Image path
	Image string `json:"image"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kvct
// KubevirtCommonTemplatesBundle defines the CommonTemplates CR
type KubevirtCommonTemplatesBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the configuration of Common Templates
	Spec VersionSpec `json:"spec,omitempty"`

	// Status holds the current status of Common Templates
	Status ConfigStatus `json:"status,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kvnl
// KubevirtNodeLabellerBundle defines the NodeLabeller CR
type KubevirtNodeLabellerBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the configuration of NodeLabeller
	Spec ComponentSpec `json:"spec,omitempty"`

	// Status holds the current status of NodeLabeller
	Status ConfigStatus `json:"status,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kvtv
// KubevirtTemplateValidator defines the TemplateValidator CR
type KubevirtTemplateValidator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the configuration of TemplateValidator
	Spec TemplateValidatorSpec `json:"spec,omitempty"`

	// Status holds the current status of TemplateValidator
	Status ConfigStatus `json:"status,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=kvma
// KubevirtMetricsAggregation defines the MetricsAggregation CR
type KubevirtMetricsAggregation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec contains the configuration of MetricsAggregation
	Spec VersionSpec `json:"spec,omitempty"`

	// Status holds the current status of MetricsAggregation
	Status ConfigStatus `json:"status,omitempty"`
}

// Defines the version of the operand
type VersionSpec struct {
	// Defines the version of the operand
	Version string `json:"version,omitempty"`
}

// Defines the configuration of the NodeLabeller
type ComponentSpec struct {
	// Defines the version of the NodeLabeller
	Version string `json:"version,omitempty"`

	// Define the node affinity for NodeLabeller pods
	Affinity v1.Affinity `json:"affinity,omitempty"`

	// Define node selector labels for NodeLabeller pods
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Define tolerations for NodeLabeller pods
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
}

// Defines the configuration of Template Validator
type TemplateValidatorSpec struct {
	// Defines the version of TemplateValidaotr
	Version string `json:"version,omitempty"`

	// Defines the desired number of replicas for TemplateValidator
	TemplateValidatorReplicas int `json:"templateValidatorReplicas,omitempty"`

	// Define the node affinity for TemplateValidator pods
	Affinity v1.Affinity `json:"affinity,omitempty"`

	// Define node selector labels for TemplateValidator
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Define tolerations for TemplateValidator
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`
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
