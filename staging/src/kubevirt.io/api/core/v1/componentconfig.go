package v1

// This code is copied from
// https://github.com/kubevirt/controller-lifecycle-operator-sdk/blob/master/pkg/sdk/api/types.go
// in order to avoid dependency loops

import (
	corev1 "k8s.io/api/core/v1"
)

// NodePlacement describes node scheduling configuration.
type NodePlacement struct {
	// nodeSelector is the node selector applied to the relevant kind of pods
	// It specifies a map of key-value pairs: for the pod to be eligible to run on a node,
	// the node must have each of the indicated key-value pairs as labels
	// (it can have additional labels as well).
	// See https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#nodeselector
	// +kubebuilder:validation:Optional
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// affinity enables pod affinity/anti-affinity placement expanding the types of constraints
	// that can be expressed with nodeSelector.
	// affinity is going to be applied to the relevant kind of pods in parallel with nodeSelector
	// See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity
	// +kubebuilder:validation:Optional
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// tolerations is a list of tolerations applied to the relevant kind of pods
	// See https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ for more info.
	// These are additional tolerations other than default ones.
	// +kubebuilder:validation:Optional
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

type ComponentConfig struct {
	// nodePlacement describes scheduling configuration for specific
	// KubeVirt components
	//+optional
	NodePlacement *NodePlacement `json:"nodePlacement,omitempty"`
	// replicas indicates how many replicas should be created for each KubeVirt infrastructure
	// component (like virt-api or virt-controller). Defaults to 2.
	// WARNING: this is an advanced feature that prevents auto-scaling for core kubevirt components. Please use with caution!
	//+optional
	Replicas *uint8 `json:"replicas,omitempty"`
}
