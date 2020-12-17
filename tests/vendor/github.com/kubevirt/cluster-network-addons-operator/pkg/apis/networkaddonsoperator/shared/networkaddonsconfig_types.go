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
// SelfSignConfiguration defines self sign configuration
type SelfSignConfiguration struct {
	// CARotateInterval defines duration for CA and certificate
	CARotateInterval   string `json:"caRotateInterval,omitempty"`
	// CAOverlapInterval defines the duration of CA Certificates at CABundle if not set it will default to CARotateInterval
	CAOverlapInterval  string `json:"caOverlapInterval,omitempty"`
	// CertRotateInterval defines duration for of service certificate
	CertRotateInterval string `json:"certRotateInterval,omitempty"`
}

// +k8s:openapi-gen=true
// PlacementConfiguration defines node placement configuration
type PlacementConfiguration struct {
	// Infra defines placement configuration for master nodes
	Infra     *Placement `json:"infra,omitempty"`
	// Workloads defines placement configuration for worker nodes
	Workloads *Placement `json:"workloads,omitempty"`
}

// +k8s:openapi-gen=true
type Placement struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Affinity     corev1.Affinity     `json:"affinity,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
}

// Multus plugin enables attaching multiple network interfaces to Pods in Kubernetes
// +k8s:openapi-gen=true
type Multus struct{}

// LinuxBridge plugin allows users to create a bridge and add the host and the container to it
// +k8s:openapi-gen=true
type LinuxBridge struct{}

// Ovs plugin allows users to define Kubernetes networks on top of Open vSwitch bridges available on nodes
// +k8s:openapi-gen=true
type Ovs struct{}

// NMState is a declarative node network configuration driven through Kubernetes API
// +k8s:openapi-gen=true
type NMState struct{}

// KubeMacPool plugin manages MAC allocation to Pods and VMs in Kubernetes
// +k8s:openapi-gen=true
type KubeMacPool struct {
	// RangeStart defines the first mac in range
	RangeStart string `json:"rangeStart,omitempty"`
	// RangeEnd defines the last mac in range
	RangeEnd   string `json:"rangeEnd,omitempty"`
}

// MacvtapCni plugin allows users to define Kubernetes networks on top of existing host interfaces
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
// This struct is no exposed/registered as part of the CRD, but is used by the v1alpha1 and v1 as kind of inside-helper struct
type NetworkAddonsConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NetworkAddonsConfigSpec   `json:"spec,omitempty"`
	Status NetworkAddonsConfigStatus `json:"status,omitempty"`
}
