package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.13.0 paths=./... object crd output:artifacts:code=./,config=../../../../artifacts

//go:generate go run k8s.io/code-generator/cmd/client-gen@v0.28.0 client-gen --go-header-file ../../../../hack/custom-boilerplate.go.txt --clientset-name versioned --input-base "" --input github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1 --output-package github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1/apis/clientset ..

//go:generate go run k8s.io/code-generator/cmd/lister-gen@v0.28.0 lister-gen --go-header-file ../../../../hack/custom-boilerplate.go.txt --input-dirs github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1 --output-package github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1/apis/listers ..

//go:generate go run k8s.io/code-generator/cmd/informer-gen@v0.28.0 informer-gen --go-header-file ../../../../hack/custom-boilerplate.go.txt --input-dirs github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1 --versioned-clientset-package github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1/apis/clientset/versioned --listers-package github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1/apis/listers --output-package github.com/k8snetworkplumbingwg/ipamclaims/pkg/crd/ipamclaims/v1alpha1/apis/informers ..

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:resource:path=ipamclaims,singular=ipamclaim,scope=Namespaced
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IPAMClaim is the Schema for the IPAMClaim API
type IPAMClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPAMClaimSpec   `json:"spec,omitempty"`
	Status IPAMClaimStatus `json:"status,omitempty"`
}

type IPAMClaimSpec struct {
	// The network name for which this persistent allocation was created
	Network string `json:"network"`
	// The pod interface name for which this allocation was created
	Interface string `json:"interface"`
}

type IPAMClaimStatus struct {
	// The list of IP addresses (v4, v6) that were allocated for the pod interface
	IPs []string `json:"ips"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type IPAMClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IPAMClaim `json:"items"`
}
