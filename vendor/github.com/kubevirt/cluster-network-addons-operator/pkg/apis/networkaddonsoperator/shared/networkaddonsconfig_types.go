package shared

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkAddonsConfigSpec defines the desired state of NetworkAddonsConfig
// +k8s:openapi-gen=true
type NetworkAddonsConfigSpec struct {
	Multus                 *Multus                 `json:"multus,omitempty"`
	LinuxBridge            *LinuxBridge            `json:"linuxBridge,omitempty"`
	Ovs                    *Ovs                    `json:"ovs,omitempty"`
	KubeMacPool            *KubeMacPool            `json:"kubeMacPool,omitempty"`
	ImagePullPolicy        corev1.PullPolicy       `json:"imagePullPolicy,omitempty"`
	NMState                *NMState                `json:"nmstate,omitempty"`
	MacvtapCni             *MacvtapCni             `json:"macvtap,omitempty"`
	SelfSignConfiguration  *SelfSignConfiguration  `json:"selfSignConfiguration,omitempty"`
	PlacementConfiguration *PlacementConfiguration `json:"placementConfiguration,omitempty"`
}

// +k8s:openapi-gen=true
type SelfSignConfiguration struct {
	CARotateInterval   string `json:"caRotateInterval,omitempty"`
	CAOverlapInterval  string `json:"caOverlapInterval,omitempty"`
	CertRotateInterval string `json:"certRotateInterval,omitempty"`
}

// +k8s:openapi-gen=true
type PlacementConfiguration struct {
	Infra     *Placement `json:"infra,omitempty"`
	Workloads *Placement `json:"workloads,omitempty"`
}

// +k8s:openapi-gen=true
type Placement struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Affinity     corev1.Affinity     `json:"affinity,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
}

// +k8s:openapi-gen=true
type Multus struct{}

// +k8s:openapi-gen=true
type LinuxBridge struct{}

// +k8s:openapi-gen=true
type Ovs struct{}

// +k8s:openapi-gen=true
type NMState struct{}

// +k8s:openapi-gen=true
type KubeMacPool struct {
	RangeStart string `json:"rangeStart,omitempty"`
	RangeEnd   string `json:"rangeEnd,omitempty"`
}

// +k8s:openapi-gen=true
type MacvtapCni struct{}

// NetworkAddonsConfigStatus defines the observed state of NetworkAddonsConfig
// +k8s:openapi-gen=true
type NetworkAddonsConfigStatus struct {
	OperatorVersion string                   `json:"operatorVersion,omitempty"`
	ObservedVersion string                   `json:"observedVersion,omitempty"`
	TargetVersion   string                   `json:"targetVersion,omitempty"`
	Conditions      []conditionsv1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`
	Containers      []Container              `json:"containers,omitempty"`
}

type Container struct {
	ParentKind string `json:"parentKind"`
	ParentName string `json:"parentName"`
	Name       string `json:"name"`
	Image      string `json:"image"`
}

// NetworkAddonsConfig is the Schema for the networkaddonsconfigs API
// This struct is no exposed/registered as part of the CRD, but is used
// by the v1alpha1 and v1 as kind of inside-helper struct
type NetworkAddonsConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkAddonsConfigSpec   `json:"spec,omitempty"`
	Status NetworkAddonsConfigStatus `json:"status,omitempty"`
}
