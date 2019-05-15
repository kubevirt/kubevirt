package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeControllerManager provides information to configure an operator to manage kube-controller-manager.
type KubeControllerManager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	// +required
	Spec KubeControllerManagerSpec `json:"spec"`
	// +optional
	Status KubeControllerManagerStatus `json:"status"`
}

type KubeControllerManagerSpec struct {
	StaticPodOperatorSpec `json:",inline"`
}

type KubeControllerManagerStatus struct {
	StaticPodOperatorStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeControllerManagerList is a collection of items
type KubeControllerManagerList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	metav1.ListMeta `json:"metadata"`
	// Items contains the items
	Items []KubeControllerManager `json:"items"`
}
