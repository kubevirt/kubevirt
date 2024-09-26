package v1

import (
	operatorv1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDescheduler is the Schema for the deschedulers API
// +k8s:openapi-gen=true
// +genclient
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
type KubeDescheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec holds user settable values for configuration
	// +required
	Spec KubeDeschedulerSpec `json:"spec"`
	// status holds observed values from the cluster. They may not be overridden.
	// +optional
	Status KubeDeschedulerStatus `json:"status"`
}

// KubeDeschedulerSpec defines the desired state of KubeDescheduler
type KubeDeschedulerSpec struct {
	operatorv1.OperatorSpec `json:",inline"`

	// Profiles sets which descheduler strategy profiles are enabled
	Profiles []DeschedulerProfile `json:"profiles"`

	// DeschedulingIntervalSeconds is the number of seconds between descheduler runs
	// +optional
	DeschedulingIntervalSeconds *int32 `json:"deschedulingIntervalSeconds,omitempty"`

	// evictionLimits restrict the number of evictions during each descheduling run
	EvictionLimits *EvictionLimits `json:"evictionLimits,omitempty"`

	// ProfileCustomizations contains various parameters for modifying the default behavior of certain profiles
	ProfileCustomizations *ProfileCustomizations `json:"profileCustomizations,omitempty"`

	// Mode configures the descheduler to either evict pods (Automatic) or to simulate the eviction (Predictive)
	// +optional
	// +kubebuilder:default=Predictive
	Mode Mode `json:"mode"`
}

type EvictionLimits struct {
	// total restricts the maximum number of overall evictions
	Total *int32 `json:"total,omitempty"`
}

// ProfileCustomizations contains various parameters for modifying the default behavior of certain profiles
type ProfileCustomizations struct {
	// PodLifetime is the length of time after which pods should be evicted
	// This field should be used with profiles that enable the PodLifetime strategy, such as LifecycleAndUtilization
	// +kubebuilder:validation:Format=duration
	PodLifetime *metav1.Duration `json:"podLifetime,omitempty"`

	// ThresholdPriority when set will reject eviction of any pod with priority equal or higher
	// It is invalid to set it alongside ThresholdPriorityClassName
	ThresholdPriority *int32 `json:"thresholdPriority,omitempty"`

	// ThresholdPriorityClassName when set will reject eviction of any pod with priority equal or higher
	// It is invalid to set it alongside ThresholdPriority
	ThresholdPriorityClassName string `json:"thresholdPriorityClassName,omitempty"`

	// Namespaces overrides included and excluded namespaces while keeping
	// the default exclusion of all openshift-*, kube-system and hypershift namespaces
	Namespaces Namespaces `json:"namespaces"`

	// LowNodeUtilizationThresholds enumerates predefined experimental thresholds
	// +kubebuilder:validation:Enum=Low;Medium;High;""
	DevLowNodeUtilizationThresholds *LowNodeUtilizationThresholdsType `json:"devLowNodeUtilizationThresholds"`

	// DevEnableEvictionsInBackground enables descheduler's EvictionsInBackground alpha feature.
	// The EvictionsInBackground alpha feature is a subject to change.
	// Currently provided as an experimental feature.
	DevEnableEvictionsInBackground bool `json:"devEnableEvictionsInBackground,omitempty"`

	// devHighNodeUtilizationThresholds enumerates thresholds for node utilization levels.
	// The threshold values are subject to change.
	// Currently provided as an experimental feature.
	// +kubebuilder:validation:Enum=Minimal;Modest;Moderate;""
	DevHighNodeUtilizationThresholds *HighNodeUtilizationThresholdsType `json:"devHighNodeUtilizationThresholds"`
}

type LowNodeUtilizationThresholdsType string

var (
	// LowThreshold sets thresholds:targetThresholds in 10%/30% ratio
	LowThreshold LowNodeUtilizationThresholdsType = "Low"

	// MediumThreshold sets thresholds:targetThresholds in 20%/50% ratio
	MediumThreshold LowNodeUtilizationThresholdsType = "Medium"

	// HighThreshold sets thresholds:targetThresholds in 40%/70% ratio
	HighThreshold LowNodeUtilizationThresholdsType = "High"
)

type HighNodeUtilizationThresholdsType string

var (
	// CompactLowThreshold sets thresholds to 10% ratio.
	// The threshold value is subject to change.
	CompactMinimalThreshold HighNodeUtilizationThresholdsType = "Minimal"

	// CompactMediumThreshold sets thresholds to 20% ratio.
	// The threshold value is subject to change.
	CompactModestThreshold HighNodeUtilizationThresholdsType = "Modest"

	// CompactHighThreshold sets thresholds to 30% ratio.
	// The threshold value is subject to change.
	CompactModerateThreshold HighNodeUtilizationThresholdsType = "Moderate"
)

// Namespaces overrides included and excluded namespaces while keeping
// the default exclusion of all openshift-*, kube-system and hypershift namespaces
type Namespaces struct {
	Included []string `json:"included"`
	Excluded []string `json:"excluded"`
}

// DeschedulerProfile allows configuring the enabled strategy profiles for the descheduler
// it allows multiple profiles to be enabled at once, which will have cumulative effects on the cluster.
// +kubebuilder:validation:Enum=AffinityAndTaints;TopologyAndDuplicates;LifecycleAndUtilization;DevPreviewLongLifecycle;LongLifecycle;SoftTopologyAndDuplicates;EvictPodsWithLocalStorage;EvictPodsWithPVC;CompactAndScale
type DeschedulerProfile string

var (
	// AffinityAndTaints enables descheduling strategies that balance pods based on affinity and
	// node taint violations.
	AffinityAndTaints DeschedulerProfile = "AffinityAndTaints"

	// TopologyAndDuplicates attempts to spread pods evenly among nodes based on topology spread
	// constraints and duplicate replicas on the same node.
	TopologyAndDuplicates DeschedulerProfile = "TopologyAndDuplicates"

	// SoftTopologyAndDuplicates attempts to spread pods evenly similar to TopologyAndDuplicates, but includes
	// soft ("ScheduleAnyway") topology spread constraints
	SoftTopologyAndDuplicates DeschedulerProfile = "SoftTopologyAndDuplicates"

	// LifecycleAndUtilization attempts to balance pods based on node resource usage, pod age, and pod restarts
	LifecycleAndUtilization DeschedulerProfile = "LifecycleAndUtilization"

	// EvictPodsWithLocalStorage enables pods with local storage to be evicted by the descheduler by all other profiles
	EvictPodsWithLocalStorage DeschedulerProfile = "EvictPodsWithLocalStorage"

	// EvictPodsWithPVC prevents pods with PVCs from being evicted by all other profiles
	EvictPodsWithPVC DeschedulerProfile = "EvictPodsWithPVC"

	// DevPreviewLongLifecycle handles cluster lifecycle over a long term
	// Deprecated: use LongLifecycle instead
	DevPreviewLongLifecycle DeschedulerProfile = "DevPreviewLongLifecycle"

	// LongLifecycle handles cluster lifecycle over a long term
	LongLifecycle DeschedulerProfile = "LongLifecycle"

	// CompactAndScale seeks to evict pods to enable the same workload to run on a smaller set of nodes.
	CompactAndScale DeschedulerProfile = "CompactAndScale"
)

// DeschedulerProfile allows configuring the enabled strategy profiles for the descheduler
// it allows multiple profiles to be enabled at once, which will have cumulative effects on the cluster.
// +kubebuilder:validation:Enum=Automatic;Predictive
type Mode string

var (
	// Automatic mode evicts pods from the cluster
	Automatic Mode = "Automatic"

	// Predictive mode simulates eviction of pods
	Predictive Mode = "Predictive"
)

// KubeDeschedulerStatus defines the observed state of KubeDescheduler
type KubeDeschedulerStatus struct {
	operatorv1.OperatorStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeDeschedulerList contains a list of KubeDescheduler
type KubeDeschedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeDescheduler `json:"items"`
}
