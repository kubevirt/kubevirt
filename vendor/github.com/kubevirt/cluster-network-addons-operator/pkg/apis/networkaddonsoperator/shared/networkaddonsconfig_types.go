package shared

import (
	ocpv1 "github.com/openshift/api/config/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkAddonsConfigSpec defines the desired state of NetworkAddonsConfig
type NetworkAddonsConfigSpec struct {
	Multus                 *Multus                   `json:"multus,omitempty"`
	MultusDynamicNetworks  *MultusDynamicNetworks    `json:"multusDynamicNetworks,omitempty"`
	LinuxBridge            *LinuxBridge              `json:"linuxBridge,omitempty"`
	Ovs                    *Ovs                      `json:"ovs,omitempty"`
	KubeMacPool            *KubeMacPool              `json:"kubeMacPool,omitempty"`
	ImagePullPolicy        corev1.PullPolicy         `json:"imagePullPolicy,omitempty"`
	NMState                *NMState                  `json:"nmstate,omitempty"`
	KubeSecondaryDNS       *KubeSecondaryDNS         `json:"kubeSecondaryDNS,omitempty"`
	MacvtapCni             *MacvtapCni               `json:"macvtap,omitempty"`
	KubevirtIpamController *KubevirtIpamController   `json:"kubevirtIpamController,omitempty"`
	SelfSignConfiguration  *SelfSignConfiguration    `json:"selfSignConfiguration,omitempty"`
	PlacementConfiguration *PlacementConfiguration   `json:"placementConfiguration,omitempty"`
	TLSSecurityProfile     *ocpv1.TLSSecurityProfile `json:"tlsSecurityProfile,omitempty"`
}

// SelfSignConfiguration defines self sign configuration
type SelfSignConfiguration struct {
	// CARotateInterval defines duration for CA expiration
	CARotateInterval string `json:"caRotateInterval,omitempty"`
	// CAOverlapInterval defines the duration where expired CA certificate can overlap with new one, in order to allow fluent CA rotation transitioning
	CAOverlapInterval string `json:"caOverlapInterval,omitempty"`
	// CertRotateInterval defines duration for of service certificate expiration
	CertRotateInterval string `json:"certRotateInterval,omitempty"`
	// CertOverlapInterval defines the duration where expired service certificate can overlap with new one, in order to allow fluent service rotation transitioning
	CertOverlapInterval string `json:"certOverlapInterval,omitempty"`
}

// PlacementConfiguration defines node placement configuration
type PlacementConfiguration struct {
	// Infra defines placement configuration for control-plane nodes
	Infra *Placement `json:"infra,omitempty"`
	// Workloads defines placement configuration for worker nodes
	Workloads *Placement `json:"workloads,omitempty"`
}

type Placement struct {
	NodeSelector map[string]string   `json:"nodeSelector,omitempty"`
	Affinity     corev1.Affinity     `json:"affinity,omitempty"`
	Tolerations  []corev1.Toleration `json:"tolerations,omitempty"`
}

// Multus plugin enables attaching multiple network interfaces to Pods in Kubernetes
type Multus struct{}

// A multus extension enabling hot-plug and hot-unplug of Pod interfaces
type MultusDynamicNetworks struct{}

// LinuxBridge plugin allows users to create a bridge and add the host and the container to it
type LinuxBridge struct{}

// Ovs plugin allows users to define Kubernetes networks on top of Open vSwitch bridges available on nodes
type Ovs struct{}

// NMState is a declarative node network configuration driven through Kubernetes API
type NMState struct{}

// KubeSecondaryDNS plugin allows to support FQDN for VMI's secondary networks
type KubeSecondaryDNS struct {
	// Domain defines the FQDN domain
	Domain string `json:"domain,omitempty"`
	// NameServerIp defines the name server IP
	NameServerIP string `json:"nameServerIP,omitempty"`
}

// KubeMacPool plugin manages MAC allocation to Pods and VMs in Kubernetes
type KubeMacPool struct {
	// RangeStart defines the first mac in range
	RangeStart string `json:"rangeStart,omitempty"`
	// RangeEnd defines the last mac in range
	RangeEnd string `json:"rangeEnd,omitempty"`
}

// MacvtapCni plugin allows users to define Kubernetes networks on top of existing host interfaces
type MacvtapCni struct {
	// DevicePluginConfig allows the user to override the name of the
	// `ConfigMap` where the device plugin configuration is held.
	DevicePluginConfig string `json:"devicePluginConfig,omitempty"`
}

// KubevirtIpamController plugin allows to support IPAM for secondary networks
type KubevirtIpamController struct{}

// NetworkAddonsConfigStatus defines the observed state of NetworkAddonsConfig
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
