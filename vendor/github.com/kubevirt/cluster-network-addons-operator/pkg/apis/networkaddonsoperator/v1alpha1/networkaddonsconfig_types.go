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
}

// +k8s:openapi-gen=true
type Multus struct{}

// +k8s:openapi-gen=true
type LinuxBridge struct{}

type KubeMacPool struct {
	StartPoolRange string `json:"startPoolRange,omitempty"`
	EndPoolRange   string `json:"endPoolRange,omitempty"`
}

// +k8s:openapi-gen=true
type Sriov struct{}

// NetworkAddonsConfigStatus defines the observed state of NetworkAddonsConfig
// +k8s:openapi-gen=true
type NetworkAddonsConfigStatus struct {
	Conditions []NetworkAddonsCondition `json:"conditions,omitempty" optional:"true"`
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
