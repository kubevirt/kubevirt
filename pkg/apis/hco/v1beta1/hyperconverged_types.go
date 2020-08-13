package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

// HyperConvergedName is the name of the HyperConverged resource that will be reconciled
const HyperConvergedName = "kubevirt-hyperconverged"

// HyperConvergedConfig is the schema for pod configuration that is going to
// be propagated by HCO to its managed operators.
type HyperConvergedConfig struct {
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
	Affinity corev1.Affinity `json:"affinity,omitempty"`

	// tolerations is a list of tolerations applied to the relevant kind of pods
	// See https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/ for more info.
	// These are additional tolerations other than default ones.
	// +kubebuilder:validation:Optional
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// HyperConvergedSpec defines the desired state of HyperConverged
// +k8s:openapi-gen=true
type HyperConvergedSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// BareMetalPlatform indicates whether the infrastructure is baremetal.
	BareMetalPlatform bool `json:"bareMetalPlatform,omitempty"`

	// LocalStorageClassName the name of the local storage class.
	LocalStorageClassName string `json:"localStorageClassName,omitempty"`

	// infra HyperConvergedConfig influences the pod configuration (currently only placement)
	// for all the infra components needed on the virtualization enabled cluster
	// but not necessarely directly on each node running VMs/VMIs.
	// +optional
	Infra HyperConvergedConfig `json:"infra,omitempty"`

	// workloads HyperConvergedConfig influences the pod configuration (currently only placement) of components
	// which need to be running on a node where virtualization workloads should be able to run.
	// Changes to Workloads HyperConvergedConfig can be applied only without existing workload.
	// +optional
	Workloads HyperConvergedConfig `json:"workloads,omitempty"`

	// operator version
	Version string `json:"version,omitempty"`
}

// HyperConvergedStatus defines the observed state of HyperConverged
// +k8s:openapi-gen=true
type HyperConvergedStatus struct {
	// Conditions describes the state of the HyperConverged resource.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +optional
	// +listType=set
	Conditions []conditionsv1.Condition `json:"conditions,omitempty"  patchStrategy:"merge" patchMergeKey:"type"`

	// RelatedObjects is a list of objects created and maintained by this
	// operator. Object references will be added to this list after they have
	// been created AND found in the cluster.
	// +optional
	// +listType=set
	RelatedObjects []corev1.ObjectReference `json:"relatedObjects,omitempty"`

	// Versions is a list of HCO component versions, as name/version pairs. The version with a name of "operator"
	// is the HCO version itself, as described here:
	// https://github.com/openshift/cluster-version-operator/blob/master/docs/dev/clusteroperator.md#version
	// +optional
	Versions Versions `json:"versions,omitempty"`
}

func (hcs *HyperConvergedStatus) UpdateVersion(name, version string) {
	if hcs.Versions == nil {
		hcs.Versions = Versions{}
	}
	hcs.Versions.updateVersion(name, version)
}

func (hcs *HyperConvergedStatus) GetVersion(name string) (string, bool) {
	return hcs.Versions.getVersion(name)
}

type Version struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

func newVersion(name, version string) Version {
	return Version{Name: name, Version: version}
}

type Versions []Version

func (vs *Versions) updateVersion(name, version string) {
	for i, v := range *vs {
		if v.Name == name {
			(*vs)[i].Version = version
			return
		}
	}
	*vs = append(*vs, newVersion(name, version))
}

func (vs *Versions) getVersion(name string) (string, bool) {
	for _, v := range *vs {
		if v.Name == name {
			return v.Version, true
		}
	}
	return "", false
}

// ConditionReconcileComplete communicates the status of the HyperConverged resource's
// reconcile functionality. Basically, is the Reconcile function running to completion.
const ConditionReconcileComplete conditionsv1.ConditionType = "ReconcileComplete"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HyperConverged is the Schema for the hyperconvergeds API
// +k8s:openapi-gen=true
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:resource:scope=Namespaced,categories={all},shortName={hco,hcos}
// +kubebuilder:subresource:status
type HyperConverged struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HyperConvergedSpec   `json:"spec,omitempty"`
	Status HyperConvergedStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// HyperConvergedList contains a list of HyperConverged
type HyperConvergedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HyperConverged `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HyperConverged{}, &HyperConvergedList{})
}
