package v1

// This code is copied from
// https://github.com/kubevirt/controller-lifecycle-operator-sdk/blob/master/pkg/sdk/api/types.go
// in order to avoid dependency loops

import (
	corev1 "k8s.io/api/core/v1"
)

// NodePlacement describes node scheduling configuration.
//
// +k8s:openapi-gen=true
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
	//+listType=map
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// ---
// +k8s:openapi-gen=true
type ComponentConfig struct {
	// nodePlacement decsribes scheduling confiuguration for specific
	// KubeVirt components
	//+optional
	NodePlacement *NodePlacement `json:"nodePlacement,omitempty"`
}
