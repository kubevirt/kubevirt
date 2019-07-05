package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkAddonsConfigSpec defines the desired state of NetworkAddonsConfig
// +k8s:openapi-gen=true
type NetworkAddonsConfigSpec struct {
	Multus          *Multus           `json:"multus,omitempty"`
	LinuxBridge     *LinuxBridge      `json:"linuxBridge,omitempty"`
	Sriov           *Sriov            `json:"sriov,omitempty"`
	KubeMacPool     *KubeMacPool      `json:"kubeMacPool,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	NMState         *NMState          `json:"nmstate,omitempty"`
}

// +k8s:openapi-gen=true
type Multus struct{}

// +k8s:openapi-gen=true
type LinuxBridge struct{}

type NMState struct{}

type KubeMacPool struct {
	RangeStart string `json:"rangeStart,omitempty"`
	RangeEnd   string `json:"rangeEnd,omitempty"`
}

// +k8s:openapi-gen=true
type Sriov struct{}

// NetworkAddonsConfigStatus defines the observed state of NetworkAddonsConfig
// +k8s:openapi-gen=true
type NetworkAddonsConfigStatus struct {
	OperatorVersion string                   `json:"operatorVersion,omitempty"`
	ObservedVersion string                   `json:"observedVersion,omitempty"`
	TargetVersion   string                   `json:"targetVersion,omitempty"`
	Conditions      []NetworkAddonsCondition `json:"conditions,omitempty" optional:"true"`
	Containers      []Container              `json:"containers,omitempty"`
}

// NetworkAddonsCondition represents a condition of a NetworkAddons deployment
// ---
// +k8s:openapi-gen=true
type NetworkAddonsCondition struct {
	Type               NetworkAddonsConditionType `json:"type"`
	Status             corev1.ConditionStatus     `json:"status"`
	LastProbeTime      metav1.Time                `json:"lastProbeTime,omitempty"`
	LastTransitionTime metav1.Time                `json:"lastTransitionTime,omitempty"`
	Reason             string                     `json:"reason,omitempty"`
	Message            string                     `json:"message,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type NetworkAddonsConditionType string

// These are the valid NetworkAddons condition types
const (
	// Whether operator failed during deployment
	NetworkAddonsConditionFailing NetworkAddonsConditionType = "Failing"
	// Whether is the deployment progressing
	NetworkAddonsConditionProgressing NetworkAddonsConditionType = "Progressing"
	// Whether all components were ready
	NetworkAddonsConditionAvailable NetworkAddonsConditionType = "Ready"
)

type Container struct {
	Namespace  string `json:"namespace"`
	ParentKind string `json:"parentKind"`
	ParentName string `json:"parentName"`
	Name       string `json:"name"`
	Image      string `json:"image"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkAddonsConfig is the Schema for the networkaddonsconfigs API
// +k8s:openapi-gen=true
type NetworkAddonsConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkAddonsConfigSpec   `json:"spec,omitempty"`
	Status NetworkAddonsConfigStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NetworkAddonsConfigList contains a list of NetworkAddonsConfig
type NetworkAddonsConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkAddonsConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NetworkAddonsConfig{}, &NetworkAddonsConfigList{})
}
